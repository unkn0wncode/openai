package responses

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

// ResponsesClient is the client for the Responses API.
type ResponsesClient struct {
	*openai.Config
}

// NewResponsesClient creates a new ResponsesClient.
func NewResponsesClient(config *openai.Config) *ResponsesClient {
	return &ResponsesClient{Config: config}
}

// interface compliance checks
var _ responses.ResponsesService = (*ResponsesClient)(nil)

// marshalRequest marshals the request into a JSON object, including tools by name.
func (c *ResponsesClient) marshalRequest(data *responses.Request) ([]byte, error) {
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
func (c *ResponsesClient) executeRequest(data *responses.Request) (*response, error) {
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
	resp, err = c.HTTPClient.Do(req)
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
			if len(m.Content) == 0 {
				return nil, fmt.Errorf("no content in output message")
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

// Response sends a request to the Responses API with custom data.
// Returns the AI reply, request ID, and any error.
func (c *ResponsesClient) Response(req *responses.Request) (*responses.Response, error) {
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

	// we'll gather outputs to return here
	var aggregatedOutputs []output.Any
	var aggregatedParsedOutputs []any

	// keep track of whether we have function calls to return
	var haveReturnableFunctionCalls bool

	var toolOutputs []output.FunctionCallOutput
	for i, anyOutput := range resp.ParsedOutputs {
		switch o := anyOutput.(type) {
		case output.Message:
			// if we got only one output and it's a message, just return it
			if len(respData.Output) == 1 {
				return respData.checkResponseData()
			}

			// for a messages given alongside other outputs, we may have a handler
			if req.IntermediateMessageHandler != nil {
				req.IntermediateMessageHandler(o)
				continue
			}

			// otherwise, add to the aggregated outputs
			aggregatedOutputs = append(aggregatedOutputs, resp.Outputs[i])
			aggregatedParsedOutputs = append(aggregatedParsedOutputs, o)
		case output.FunctionCall:
			if req.ReturnToolCalls {
				// if calls should be returned, add to the aggregated outputs
				aggregatedOutputs = append(aggregatedOutputs, resp.Outputs[i])
				aggregatedParsedOutputs = append(aggregatedParsedOutputs, o)
				haveReturnableFunctionCalls = true
				continue
			}

			// Get the tool or function from the registered function calls
			var F func(params json.RawMessage) (string, error)
			if t, ok := c.Tools.GetTool(o.Name); ok && t.Function.F != nil {
				F = t.Function.F
			} else if f, ok := c.Tools.GetFunction(o.Name); ok && f.F != nil {
				F = f.F
			} else {
				return nil, fmt.Errorf(
					"tool/function '%s' is not registered or has no implementation",
					o.Name,
				)
			}

			// Execute the function
			fResult, err := F([]byte(o.Arguments))
			switch {
			case err == nil:
			case errors.Is(err, tools.ErrDoNotRespond):
				// here we return ID despite error
				// because this error indicates intended behavior
				return resp, nil
			default:
				return nil, fmt.Errorf(
					"failed to execute function '%s': %w",
					o.Name, err,
				)
			}

			// Add the tool output that will be sent in a follow-up request
			toolOutputs = append(toolOutputs, output.FunctionCallOutput{
				Type:   "function_call_output",
				CallID: o.CallID,
				Output: fResult,
			})

			// we don't need to add it to the aggregated outputs,
			// because it's supposed to be handled, not returned
		default:
			// other outputs are just returned as is
			aggregatedOutputs = append(aggregatedOutputs, resp.Outputs[i])
			aggregatedParsedOutputs = append(aggregatedParsedOutputs, o)
		}
	}

	// if we have function calls to return, return everything right now
	if haveReturnableFunctionCalls {
		resp.Outputs = aggregatedOutputs
		resp.ParsedOutputs = aggregatedParsedOutputs

		return resp, nil
	}

	// If we have tool outputs, send them to the API in a follow-up request
	if len(toolOutputs) > 0 {
		// Create a follow-up request by cloning the original request
		followUpReq := req.Clone()

		// Update only the fields that need to change
		followUpReq.Input = toolOutputs
		followUpReq.PreviousResponseID = resp.ID

		// Make another request with the function results
		followupResp, err := c.Response(followUpReq)
		if err != nil {
			return nil, err
		}

		// update our response with the data from the follow-up response
		aggregatedOutputs = append(aggregatedOutputs, followupResp.Outputs...)
		aggregatedParsedOutputs = append(aggregatedParsedOutputs, followupResp.ParsedOutputs...)
		resp.Outputs = aggregatedOutputs
		resp.ParsedOutputs = aggregatedParsedOutputs
		resp.ID = followupResp.ID

		// return the updated response
		return resp, nil
	}

	// this place should be unreachable
	return nil, fmt.Errorf("logic error: unreachable code")
}
