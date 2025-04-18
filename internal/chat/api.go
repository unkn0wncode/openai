// Package chat provides a wrapper for the OpenAI Chat API.
package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"openai/chat"
	openai "openai/internal"
	"openai/models"
	"openai/roles"
	"openai/tools"
	"openai/util"
	"slices"
	"strings"
	"time"
)

const (
	apiURL = openai.BaseAPI + "v1/chat/completions"
)

type ChatClient struct {
	Config         *openai.Config
	AutoLogTripper bool // if true, LogTripper is enabled on errors and disabled on successes
}

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

// response is the response body for the Chat Completion API.
type response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"` // Unix timestamp
	Model   string `json:"model"`
	Usage   struct {
		Prompt     int `json:"prompt_tokens"`
		Completion int `json:"completion_tokens"`
		Total      int `json:"total_tokens"`
	} `json:"usage"`
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

// // promptPrice returns approximate price of the request's input in USD.
// // Mind that output is not included and is priced higher, but usually is much shorter than input.
// // Returns zero if pricing for the model is not known.
// func (c *Client) promptPrice(data chat.Request) float64 {
// 	pricing, ok := models.Data[data.Model]
// 	if !ok {
// 		c.Config.Log.Warn(fmt.Sprintf("No pricing for found model '%s'", data.Model))
// 		return 0
// 	}
// 	return float64(countTokens(data)) * pricing.PriceIn
// }

func contextTokenLimit(model string) int {
	modelData, ok := models.Data[model]
	if !ok {
		return models.Data[""].LimitContext
	}
	return modelData.LimitContext
}

func outputTokenLimit(model string) int {
	modelData, ok := models.Data[model]
	if !ok {
		return models.Data[""].LimitOutput
	}
	return modelData.LimitOutput
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
	for len(data.Messages) > minMessages && countTokens(data) > contextTokenLimit(data.Model)-data.MaxTokens {
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

func (c *ChatClient) execute(data chat.Request) (*response, error) {
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

	// Ensure MaxTokens is set for specific models
	if data.MaxTokens == 0 && data.Model == models.GPT4Vision {
		data.MaxTokens = min(outputTokenLimit(data.Model), contextTokenLimit(data.Model)-inputTokens)
	}

	b, err := c.marshalRequest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.Config.AddHeaders(req)

	var resp *http.Response
	var duration time.Duration

	err = util.Retry(func() error {
		startTime := time.Now()
		resp, err = c.Config.HTTPClient.Do(req)
		duration = time.Since(startTime)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusOK {
			// Disable LogTripper if it's on auto mode and there was no error
			if c.AutoLogTripper {
				c.disableLogTripper()
			}
			return nil
		}

		if resp.StatusCode != http.StatusOK {
			return c.handleBadRequest(resp, data.Model, duration)
		}

		// Handle other non-OK statuses
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf(
			"request (model %s) failed with status: %s, response body: %s",
			data.Model, resp.Status, string(body),
		)
	}, 3, 3*time.Second)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
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

	c.Config.Log.Info(fmt.Sprintf(
		"Consumed OpenAI tokens: %d + %d = %d ($%f) on model '%s' in %s",
		res.Usage.Prompt, res.Usage.Completion,
		res.Usage.Total, c.cost(&res), res.Model, duration,
	))

	return &res, nil
}

// handleBadRequest handles the case when the API returns a 400 Bad Request status.
// Logs the request duration and returns an error with the response body.
func (c *ChatClient) handleBadRequest(resp *http.Response, model string, duration time.Duration) error {
	c.Config.Log.Debug(fmt.Sprintf("Chat request timing: %s", duration))
	body, _ := io.ReadAll(resp.Body)
	errMsg := fmt.Errorf(
		"request (model %s) failed with status: %s, response body: %s",
		model, resp.Status, string(body),
	)
	c.enableLogTripper()
	return errMsg
}

// enableLogTripper enables LogTripper for the API requests and logs that it's enabled.
func (c *ChatClient) enableLogTripper() {
	c.Config.Log.Debug("Enable LogTripper")
	c.Config.EnableLogTripper()
}

// disableLogTripper disables LogTripper for the API requests and logs that it's disabled.
func (c *ChatClient) disableLogTripper() {
	c.Config.Log.Debug("Disable LogTripper")
	c.Config.DisableLogTripper()
}

// checkFirst checks if API response is valid,
// returns raw content or function call of first choice and error.
func (c *ChatClient) checkFirst(resp *response) (string, error) {
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

// cost returns the resulting cost of the completed request in USD.
// Returns zero if pricing for the model is not known.
func (c *ChatClient) cost(resp *response) float64 {
	pricing, ok := models.Data[resp.Model]
	if !ok {
		c.Config.Log.Warn(fmt.Sprintf("No pricing for found model '%s'", resp.Model))
		return 0
	}
	return float64(resp.Usage.Prompt)*pricing.PriceIn + float64(resp.Usage.Completion)*pricing.PriceOut
}

// marshalRequest builds request body including function calls based on registered tools
func (c *ChatClient) marshalRequest(data chat.Request) ([]byte, error) {
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
