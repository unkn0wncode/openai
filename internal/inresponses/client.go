package inresponses

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

// Client is the client for the Responses API.
type Client struct {
	*openai.Config
}

// NewClient creates a new ResponsesClient.
func NewClient(config *openai.Config) *Client {
	return &Client{Config: config}
}

// interface compliance checks
var _ responses.Service = (*Client)(nil)

// marshalRequest marshals the request into a JSON object, including tools by name.
func (c *Client) marshalRequest(data *responses.Request) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if len(data.Tools) == 0 {
		type Alias responses.Request
		return openai.Marshal((*Alias)(data))
	}

	var toolList []tools.Tool
	for _, name := range data.Tools {
		// try to get tool by name, if not found try to get function by name
		t, ok := c.Config.Tools.GetTool(name)
		if ok {
			toolList = append(toolList, t)
			continue
		}

		f, ok := c.Config.Tools.GetFunction(name)
		if ok {
			toolList = append(toolList, tools.Tool{
				Type:        "function",
				Name:        f.Name,
				Description: f.Description,
				Parameters:  f.ParamsSchema,
				Strict:      f.Strict,
				Function:    f,
			})
			continue
		}

		return nil, fmt.Errorf("tool/function '%s' is not registered", name)
	}

	type Alias responses.Request
	return openai.Marshal(&struct {
		Tools []tools.Tool `json:"tools"`
		*Alias
	}{
		Tools: toolList,
		Alias: (*Alias)(data),
	})
}

// execute sends request to the Responses API and returns the response.
func (c *Client) executeRequest(data *responses.Request) (*response, error) {
	if data == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if data.Model == "" {
		data.Model = models.Default
	}

	// Check if we have input
	if data.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	b, err := c.marshalRequest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// if testing.Testing() {
	// 	fmt.Printf("Request body: %s\n", string(b))
	// }

	req, err := http.NewRequest(http.MethodPost, c.BaseAPI+"v1/responses", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(req)

	var resp *http.Response
	before := time.Now()
	resp, err = c.HTTPClient.Do(req)
	duration := time.Since(before)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	// if testing.Testing() {
	// 	fmt.Printf("Response status: %s\n", resp.Status)
	// 	fmt.Printf("Response body: %s\n", string(body))
	// }

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(body))
	}

	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.Config.Log.
		With("model", res.Model).
		With("responseID", res.ID).
		Debug(fmt.Sprintf(
			"Consumed OpenAI Responses tokens: %d + %d = %d ($%f) in %s",
			res.Usage.InputTokens, res.Usage.OutputTokens,
			res.Usage.TotalTokens, c.cost(&res), duration,
		))

	return &res, nil
}

// response is the response body from the Responses API.
type response struct {
	// Core Properties
	ID                string `json:"id"`
	Object            string `json:"object"`
	CreatedAt         int    `json:"created_at"` // Unix timestamp
	Status            string `json:"status"`     // "completed", "failed", "in_progress", or "incomplete"
	Error             any    `json:"error"`      // Error object with code and message
	IncompleteDetails any    `json:"incomplete_details"`
	Instructions      string `json:"instructions"`
	MaxOutputTokens   int    `json:"max_output_tokens"`
	Model             string `json:"model"`

	// Output Content
	Output []output.Any `json:"output"`

	// Tool and Configuration Properties
	ParallelToolCalls  bool `json:"parallel_tool_calls"`
	PreviousResponseID any  `json:"previous_response_id"`
	Reasoning          struct {
		Effort          any `json:"effort"`
		GenerateSummary any `json:"generate_summary"`
	} `json:"reasoning"`
	Store       bool    `json:"store"`
	Temperature float64 `json:"temperature"`
	Text        struct {
		Format struct {
			Type string `json:"type"`
		} `json:"format"`
	} `json:"text"`
	ToolChoice string       `json:"tool_choice"`
	Tools      []tools.Tool `json:"tools"`
	TopP       float64      `json:"top_p"`
	Truncation string       `json:"truncation"`

	// Usage Information
	Usage struct {
		InputTokens        int `json:"input_tokens"`
		InputTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"input_tokens_details"`
		OutputTokens        int `json:"output_tokens"`
		OutputTokensDetails struct {
			ReasoningTokens int `json:"reasoning_tokens"`
		} `json:"output_tokens_details"`
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`

	// Other Properties
	User     string         `json:"user"`
	Metadata map[string]any `json:"metadata"`
}

// checkResponseData checks if API response is valid, returns raw content or tool call of first choice and error.
func (data *response) checkResponseData() (*responses.Response, error) {
	if data == nil {
		return nil, fmt.Errorf("response is nil")
	}

	if data.Error != nil {
		return nil, fmt.Errorf("got API error: %v", data.Error)
	}

	if len(data.Output) == 0 {
		return nil, fmt.Errorf("no output returned")
	}

	// parse resp
	resp := &responses.Response{
		ID:      data.ID,
		Outputs: data.Output,
	}
	err := resp.Parse()
	if err != nil {
		return nil, fmt.Errorf("failed to parse output: %w", err)
	}

	// check if outputs are valid
	for _, o := range resp.ParsedOutputs {
		if m, ok := o.(output.Message); ok {
			if m.Content == nil {
				return nil, fmt.Errorf("no content in output message (nil content)")
			}

			if anyContent, ok := m.Content.([]any); ok && len(anyContent) == 0 {
				return nil, fmt.Errorf(
					"no content in output message (zero length []any content)",
				)
			}

			status := m.Status
			isValidStatus := status == "" ||
				status == "completed" ||
				status == "incomplete" ||
				status == "error"

			if !isValidStatus {
				return nil, fmt.Errorf("got unexpected status: %s", status)
			}
		}
	}

	return resp, nil
}

// cost returns the resulting cost of the completed request in USD.
// Returns zero if pricing for the model is not known.
func (c *Client) cost(resp *response) float64 {
	pricing, ok := models.Data[resp.Model]
	if !ok {
		c.Config.Log.Warn(fmt.Sprintf("No pricing for found model '%s'", resp.Model))
		return 0
	}
	total := 0.0
	total += float64(resp.Usage.InputTokens-resp.Usage.InputTokensDetails.CachedTokens) * pricing.PriceIn
	total += float64(resp.Usage.InputTokensDetails.CachedTokens) * pricing.PriceCachedIn
	total += float64(resp.Usage.OutputTokens) * pricing.PriceOut
	return total
}

// executableFunctionCall is an intermediate representation of a function call that can be executed.
type executableFunctionCall struct {
	Name      string
	CallID    string
	Arguments json.RawMessage
	F         func(params json.RawMessage) (string, error)
}

// Send sends a request to the Responses API with custom data.
// Returns the AI reply, request ID, and any error.
func (c *Client) Send(req *responses.Request) (*responses.Response, error) {
	respData, err := c.executeRequest(req)
	if err != nil {
		return nil, err
	}

	// Check if we have output
	if len(respData.Output) == 0 {
		return nil, fmt.Errorf("no output returned")
	}

	// get and parse the outputs
	resp, err := respData.checkResponseData()
	if err != nil {
		return nil, err
	}

	// log refusals as warnings
	for _, refusal := range resp.Refusals() {
		c.Config.Log.Warn(fmt.Sprintf("got refusal: %s", refusal))
	}

	// First pass: analyze outputs and categorize them
	var messages []output.Message
	var executableCalls []executableFunctionCall
	var returnableCalls []output.FunctionCall
	var otherOutputs []output.Any
	var otherParsedOutputs []any

	for i, anyOutput := range resp.ParsedOutputs {
		switch o := anyOutput.(type) {
		case output.Message:
			messages = append(messages, o)
		case output.FunctionCall:
			if req.ReturnToolCalls {
				returnableCalls = append(returnableCalls, o)
				continue
			}

			// Get the tool or function from the registered function calls
			var F func(params json.RawMessage) (string, error)
			if t, ok := c.Tools.GetTool(o.Name); ok {
				if t.Function.F != nil {
					F = t.Function.F
				} else {
					returnableCalls = append(returnableCalls, o)
					continue
				}
			} else if f, ok := c.Tools.GetFunction(o.Name); ok {
				if f.F != nil {
					F = f.F
				} else {
					returnableCalls = append(returnableCalls, o)
					continue
				}
			} else {
				return nil, fmt.Errorf("tool/function '%s' is not registered", o.Name)
			}

			executableCalls = append(executableCalls, executableFunctionCall{
				Name:      o.Name,
				CallID:    o.CallID,
				Arguments: []byte(o.Arguments),
				F:         F,
			})
		default:
			otherOutputs = append(otherOutputs, resp.Outputs[i])
			otherParsedOutputs = append(otherParsedOutputs, o)
		}
	}

	switch {
	// Case 1: All outputs are messages/other outputs
	case len(executableCalls) == 0 && len(returnableCalls) == 0:
		return resp, nil

	// Case 2: Any returnable function calls present
	case len(returnableCalls) > 0:
		return resp, nil

	// Case 3: Mix of messages and executable function calls
	case len(executableCalls) > 0:
		// Handle messages with intermediate handler if set
		if req.IntermediateMessageHandler != nil {
			for _, msg := range messages {
				req.IntermediateMessageHandler(msg)
			}
		}

		// Execute function calls and collect outputs
		var toolOutputs []output.FunctionCallOutput
		for _, call := range executableCalls {
			fResult, err := call.F(call.Arguments)
			switch {
			case err == nil:
			case errors.Is(err, tools.ErrDoNotRespond):
				// here we return ID despite error
				// because this error indicates intended behavior
				return resp, nil
			default:
				return nil, fmt.Errorf(
					"failed to execute function '%s': %w",
					call.Name, err,
				)
			}

			// Add the tool output that will be sent in a follow-up request
			toolOutputs = append(toolOutputs, output.FunctionCallOutput{
				Type:   "function_call_output",
				CallID: call.CallID,
				Output: fResult,
			})
		}

		// we have tool outputs, send them in a follow-up request
		followUpReq := req.Clone()
		followUpReq.Input = toolOutputs
		followUpReq.PreviousResponseID = resp.ID

		followupResp, err := c.Send(followUpReq)
		if err != nil {
			return nil, err
		}

		// Combine unhandled messages (if any) with follow-up response
		var combinedOutputs []output.Any
		var combinedParsedOutputs []any

		// Add unhandled messages first
		if req.IntermediateMessageHandler == nil {
			for _, msg := range messages {
				// Marshal the message to JSON
				b, err := json.Marshal(msg)
				if err != nil {
					return nil, fmt.Errorf("failed to marshal message: %w", err)
				}
				// Create an Any instance from the raw JSON
				var anyMsg output.Any
				if err := json.Unmarshal(b, &anyMsg); err != nil {
					return nil, fmt.Errorf("failed to unmarshal message to Any: %w", err)
				}
				combinedOutputs = append(combinedOutputs, anyMsg)
				combinedParsedOutputs = append(combinedParsedOutputs, msg)
			}
		}

		// Add other outputs
		combinedOutputs = append(combinedOutputs, otherOutputs...)
		combinedParsedOutputs = append(combinedParsedOutputs, otherParsedOutputs...)

		// Add follow-up response outputs
		combinedOutputs = append(combinedOutputs, followupResp.Outputs...)
		combinedParsedOutputs = append(combinedParsedOutputs, followupResp.ParsedOutputs...)

		resp.Outputs = combinedOutputs
		resp.ParsedOutputs = combinedParsedOutputs
		resp.ID = followupResp.ID

		return resp, nil

	// Case 4: Only other outputs
	case len(otherOutputs) > 0:
		return resp, nil
	}

	// This should be unreachable
	return nil, fmt.Errorf("logic error: unreachable code, stack: %s", string(debug.Stack()))
}

// NewMessage creates a new empty message.
func (c *Client) NewMessage() *output.Message {
	return &output.Message{}
}

// NewRequest creates a new empty request.
func (c *Client) NewRequest() *responses.Request {
	return &responses.Request{}
}
