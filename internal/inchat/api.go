// Package inchat provides a wrapper for the OpenAI Chat API.
package inchat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/unkn0wncode/openai/chat"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/roles"
	"github.com/unkn0wncode/openai/tools"
)

type Client struct {
	Config *openai.Config
}

// NewClient creates a new Chat client.
func NewClient(config *openai.Config) *Client {
	return &Client{Config: config}
}

// interface conformity checks
var _ chat.Service = (*Client)(nil)

// ResponseFormatStr represents a format that the model must output.
// Should be one of:
//   - "text" (default, normal text)
//   - "json_object" (deprecated, output is valid JSON but no specific schema)
//   - JSON schema as a string (output will match schema which must follow supported rule subset)
//
// Is encoded as {"type": "json_object"}, or {"type": "text"},
// or {"type": "json_schema", "json_schema": ...}.
type ResponseFormatStr string

func (rfs ResponseFormatStr) MarshalJSON() ([]byte, error) {
	if rfs == "" {
		return nil, nil
	}

	rf := struct {
		Type   string          `json:"type,omitempty"`
		Schema json.RawMessage `json:"json_schema,omitempty"`
	}{
		Type: string(rfs),
	}
	switch rfs {
	case "text", "json_object":
	default:
		rf.Type = "json_schema"
		rf.Schema = []byte(rfs)
	}

	return openai.Marshal(rf)
}

// responseUsage contains token usage returned by Chat Completions.
type responseUsage struct {
	Prompt              int `json:"prompt_tokens"`
	Completion          int `json:"completion_tokens"`
	Total               int `json:"total_tokens"`
	PromptTokensDetails struct {
		CachedTokens     int `json:"cached_tokens"`
		CacheWriteTokens int `json:"cache_write_tokens"`
	} `json:"prompt_tokens_details"`
}

// response is the response body for the Chat Completion API.
type response struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int            `json:"created"` // Unix timestamp
	Model   string         `json:"model"`
	Usage   *responseUsage `json:"usage"`
	Choices []struct {
		Message      chat.Message `json:"message"`
		FinishReason string       `json:"finish_reason"` // stop/length/content_filter/null
		Index        int          `json:"index"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   string `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

// countTokens returns the number of tokens in the request.
func countTokens(data chat.Request) int {
	dup := data
	dup.Messages = make([]chat.Message, len(data.Messages))

	for i, msg := range data.Messages {
		newMsg := msg

		var images []chat.Image
		for _, img := range msg.Images {
			if !strings.HasPrefix(img.URL, "data:image/") {
				images = append(images, img)
			}
		}
		newMsg.Images = images

		dup.Messages[i] = newMsg
	}

	b, err := openai.Marshal(dup)
	if err != nil {
		panic("failed to marshal request body: " + err.Error())
	}

	if err := openai.LoadTokenEncoders(); err != nil {
		panic("failed to load token encoders: " + err.Error())
	}

	return len(openai.TokenEncoderChat.Encode(string(b), nil, nil))
}

func contextTokenLimit(model string) int {
	modelData, ok := models.Data[model]
	if !ok {
		return models.Data[""].LimitContext
	}
	return modelData.LimitContext
}

// trimMessages cuts off the oldest messages if the request is too long.
func trimMessages(data chat.Request) []chat.Message {
	hasSystemPrompt := len(data.Messages) > 0 &&
		(data.Messages[0].Role == roles.System || data.Messages[0].Role == roles.Developer)
	minMessages := 1
	if hasSystemPrompt {
		minMessages = 2
	}

	messages := data.Messages
	maxTokens := data.MaxCompletionTokens
	if maxTokens == 0 {
		maxTokens = data.MaxTokens //nolint:staticcheck // Honor the legacy field when MaxCompletionTokens is unset.
	}
	for len(data.Messages) > minMessages && countTokens(data) > contextTokenLimit(data.Model)-maxTokens {
		messages = nil
		if hasSystemPrompt {
			messages = append(messages, data.Messages[0])
		}

		for i := minMessages; i < len(data.Messages); i++ {
			messages = append(messages, data.Messages[i])
		}

		data.Messages = messages
	}

	return messages
}

func (c *Client) execute(data chat.Request) (*response, error) {
	if data.Model == "" {
		data.Model = models.Default
	}

	// Trim messages if the request is too long
	data.Messages = trimMessages(data)
	inputTokens := countTokens(data)
	if inputTokens > contextTokenLimit(data.Model) {
		return nil, fmt.Errorf("prompt is likely too long: ~%d tokens, max %d tokens", inputTokens, contextTokenLimit(data.Model))
	}

	// drop images of unsupported types from messages
	for i, msg := range data.Messages {
		newImages := []chat.Image{}
		for _, img := range msg.Images {
			// check if URL contains a supported file extension
			addr := strings.ToLower(img.URL)
			isSupported := false
			for _, ext := range openai.SupportedImageTypes {
				if strings.Contains(addr, "."+ext) {
					isSupported = true
					break
				}
			}
			if !isSupported && !strings.HasPrefix(addr, "data:image/") {
				c.Config.Log.Warn(fmt.Sprintf(
					"Drop image URL '%s' due to lack of supported file extension",
					img.URL,
				))
				continue
			}
			newImages = append(newImages, img)
		}
		data.Messages[i].Images = newImages
	}

	b, err := c.marshalRequest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodPost,
		c.Config.BaseAPI+"v1/chat/completions",
		bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.Config.AddHeaders(req)

	start := time.Now()
	resp, err := c.Config.HTTPClient.Do(req)
	duration := time.Since(start)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, c.handleBadRequest(resp, data.Model, duration)
	}

	// Read the response body
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var res response
	if err := json.Unmarshal(rb, &res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	cost, costErr := c.cost(&res)
	if costErr != nil {
		c.Config.Log.Debug(fmt.Sprintf(
			"Chat token-cost estimate unavailable on model %q in %s: %v",
			res.Model, duration, costErr,
		))
	} else {
		c.Config.Log.Debug(fmt.Sprintf(
			"Consumed OpenAI tokens: %d + %d = %d (standard-tier token-cost estimate $%.9f) on model %q in %s",
			res.Usage.Prompt, res.Usage.Completion,
			res.Usage.Total, cost, res.Model, duration,
		))
	}

	return &res, nil
}

// handleBadRequest handles unsuccessful HTTP responses.
// Logs the request duration and returns an error with the response body.
func (c *Client) handleBadRequest(resp *http.Response, model string, duration time.Duration) error {
	c.Config.Log.Debug(fmt.Sprintf("Chat request timing: %s", duration))
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf(
		"request (model %s) failed with status: %s, response body: %s",
		model, resp.Status, string(body),
	)
}

// checkFirst checks if API response is valid,
// returns raw content or function call of first choice and error.
func (c *Client) checkFirst(resp *response) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}

	if resp.Error.Message != "" {
		return "", fmt.Errorf("got API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	if resp.Choices[0].Message.Refusal != "" {
		return "", fmt.Errorf(
			"AI returned refusal: %s",
			resp.Choices[0].Message.Refusal,
		)
	}

	finishReason := resp.Choices[0].FinishReason
	content := resp.Choices[0].Message.Content
	expectedFinishReasons := []string{
		"",
		openai.FinishReasonStop,
		openai.FinishReasonFunctionCall,
		openai.FinishReasonToolCalls,
	}
	if !slices.Contains(expectedFinishReasons, finishReason) {
		return content, fmt.Errorf("got unexpected finish reason: %s", finishReason)
	}
	if content != "" {
		c.Config.Log.Debug(fmt.Sprintf("OpenAI response: %s", content))
	}
	if resp.Choices[0].Message.FunctionCall.Name != "" {
		c.Config.Log.Info(fmt.Sprintf(
			"OpenAI called function: %+v",
			resp.Choices[0].Message.FunctionCall,
		))
	}
	if len(resp.Choices[0].Message.ToolCalls) != 0 {
		var funcCalls []string
		for _, tc := range resp.Choices[0].Message.ToolCalls {
			funcCalls = append(funcCalls, fmt.Sprintf("%+v", tc.Function))
		}
		c.Config.Log.Info(fmt.Sprintf(
			"OpenAI called functions:\n%s",
			strings.Join(funcCalls, "\n"),
		))
	}

	return content, nil
}

// cost estimates the request's token cost at standard-tier prices in USD.
func (c *Client) cost(resp *response) (float64, error) {
	pricing, ok := models.Data[resp.Model]
	if !ok {
		c.Config.Log.Warn(fmt.Sprintf("No pricing found for model %q", resp.Model))
		return 0, fmt.Errorf("no pricing found for model %q", resp.Model)
	}
	var usage *models.Usage
	if resp.Usage != nil {
		usage = &models.Usage{
			InputTokens:  resp.Usage.Prompt,
			OutputTokens: resp.Usage.Completion,
			TotalTokens:  resp.Usage.Total,
		}
		usage.InputTokensDetails.CachedTokens = resp.Usage.PromptTokensDetails.CachedTokens
		usage.InputTokensDetails.CacheWriteTokens = resp.Usage.PromptTokensDetails.CacheWriteTokens
	}
	return pricing.Cost(usage)
}

// marshalRequest builds request body including function calls based on registered tools
func (c *Client) marshalRequest(data chat.Request) ([]byte, error) {
	if len(data.Functions) == 0 {
		type Alias chat.Request
		return openai.Marshal((*Alias)(&data))
	}
	// construct tools array for function calls
	type toolEntry struct {
		Type     string             `json:"type"`
		Function tools.FunctionCall `json:"function"`
	}
	var toolList []toolEntry
	for _, name := range data.Functions {
		f, ok := c.Config.Tools.GetFunction(name)
		if !ok {
			return nil, fmt.Errorf("function '%s' is not registered", name)
		}
		toolList = append(toolList, toolEntry{
			Type:     "function",
			Function: f,
		})
	}
	type Alias chat.Request
	return openai.Marshal(&struct {
		Tools []toolEntry `json:"tools"`
		*Alias
	}{
		Tools: toolList,
		Alias: (*Alias)(&data),
	})
}

// NewMessage creates a new empty message.
func (c *Client) NewMessage() *chat.Message {
	return &chat.Message{}
}

// NewRequest creates a new empty request.
func (c *Client) NewRequest() *chat.Request {
	return &chat.Request{}
}
