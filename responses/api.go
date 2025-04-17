package responses

// API docs: https://platform.openai.com/docs/api-reference/responses/create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"openai/roles"
	"testing"
)

const (
	responseAPI = baseAPI + "v1/responses"

	// Text format types
	TextFormatTypeText       = "text"
	TextFormatTypeJSONObject = "json_object"
	TextFormatTypeJSONSchema = "json_schema"

	// Finish reasons specific to responses API
	finishReasonToolCalls = "tool_calls"
)

// ResponseRequest is the request body for the Responses API.
type ResponseRequest struct {
	// Required
	Model string `json:"model"`
	Input any    `json:"input"` // Can be a string or a structured input

	// Optional
	Include            []string          `json:"include,omitempty"`              // Additional data to include in response
	Instructions       string            `json:"instructions,omitempty"`         // System message for context
	MaxOutputTokens    int               `json:"max_output_tokens,omitempty"`    // Max tokens to generate
	Metadata           map[string]string `json:"metadata,omitempty"`             // Key-value pairs
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`  // Allow parallel tool calls
	PreviousResponseID string            `json:"previous_response_id,omitempty"` // ID of previous response
	Reasoning          *ReasoningConfig  `json:"reasoning,omitempty"`            // Reasoning configuration
	Store              *bool             `json:"store,omitempty"`                // Whether to store the response
	Stream             bool              `json:"stream,omitempty"`               // Stream the response
	Temperature        float64           `json:"temperature,omitempty"`          // default 1
	Text               *TextFormat       `json:"text,omitempty"`                 // Text format configuration
	ToolChoice         json.RawMessage   `json:"tool_choice,omitempty"`          // default "auto"
	Tools              []Tool            `json:"tools,omitempty"`                // default []
	TopP               float64           `json:"top_p,omitempty"`                // default 1
	Truncation         string            `json:"truncation,omitempty"`           // "auto" or "disabled"
	User               string            `json:"user,omitempty"`                 // default ""

	// Custom (not part of the API)
	ReturnToolCalls bool `json:"-"` // default false
	// if set, will be called on text received along with tool calls, that otherwise is ignored
	IntermediateTextHandler func(string) `json:"-"`
}

// ReasoningConfig represents configuration options for reasoning models.
type ReasoningConfig struct {
	Effort          string `json:"effort,omitempty"`           // "low", "medium", or "high"
	GenerateSummary string `json:"generate_summary,omitempty"` // "concise" or "detailed"
}

// TextFormat represents the format configuration for text responses.
type TextFormat struct {
	Format TextFormatType `json:"format"`
}

// TextFormatType represents the type of text format.
type TextFormatType struct {
	Type        string          `json:"type"`                  // "text", "json_object", or "json_schema"
	Schema      json.RawMessage `json:"schema,omitempty"`      // Schema for json_schema type
	Name        string          `json:"name,omitempty"`        // Name for json_schema type
	Description string          `json:"description,omitempty"` // Description for json_schema type
	Strict      bool            `json:"strict,omitempty"`      // Whether to enforce strict schema validation
}

// Tool represents a tool that can be used by the model.
type Tool struct {
	Type           string          `json:"type"`                       // "function", "file_search", "web_search_preview", "computer_use_preview"
	Name           string          `json:"name,omitempty"`             // Name of the tool (required for function type)
	Description    string          `json:"description,omitempty"`      // Description of the tool (required for function type)
	Parameters     json.RawMessage `json:"parameters,omitempty"`       // Parameters schema (required for function type)
	VectorStoreIDs []string        `json:"vector_store_ids,omitempty"` // For file_search type
	DisplayWidth   int             `json:"display_width,omitempty"`    // For computer_use_preview type
	DisplayHeight  int             `json:"display_height,omitempty"`   // For computer_use_preview type
	Environment    string          `json:"environment,omitempty"`      // For computer_use_preview type
	Strict         bool            `json:"strict,omitempty"`           // Whether to enforce strict schema validation
	Function       FunctionCall    `json:"-"`                          // Reusing FunctionCall from chatapi.go (not sent to API)
}

// MarshalJSON implements custom JSON marshaling for Tool to ensure parameters are properly included
func (t Tool) MarshalJSON() ([]byte, error) {
	// Create a map to hold the JSON fields
	jsonMap := make(map[string]any)

	// Add the type field (required)
	jsonMap["type"] = t.Type

	// Handle different tool types
	switch t.Type {
	case "function":
		// Function tools require name, description, and parameters
		if t.Name != "" {
			jsonMap["name"] = t.Name
		} else if t.Function.Name != "" {
			jsonMap["name"] = t.Function.Name
		}

		if t.Description != "" {
			jsonMap["description"] = t.Description
		} else if t.Function.Description != "" {
			jsonMap["description"] = t.Function.Description
		}

		if t.Parameters != nil {
			jsonMap["parameters"] = json.RawMessage(t.Parameters)
		} else if t.Function.ParamsSchema != nil {
			jsonMap["parameters"] = json.RawMessage(t.Function.ParamsSchema)
		}

		if t.Strict {
			jsonMap["strict"] = t.Strict
		}

	case "file_search":
		// File search tools require vector_store_ids
		if len(t.VectorStoreIDs) > 0 {
			jsonMap["vector_store_ids"] = t.VectorStoreIDs
		}

	case "web_search_preview":
		// Web search preview doesn't require additional fields

	case "computer_use_preview":
		// Computer use preview can have display dimensions and environment
		if t.DisplayWidth > 0 {
			jsonMap["display_width"] = t.DisplayWidth
		}
		if t.DisplayHeight > 0 {
			jsonMap["display_height"] = t.DisplayHeight
		}
		if t.Environment != "" {
			jsonMap["environment"] = t.Environment
		}
	}

	return json.Marshal(jsonMap)
}

// ToolCall represents a call to a tool.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`     // Currently only "function" is supported
	Function FunctionCallData `json:"function"` // Reusing FunctionCallData from chatapi.go
}

// ToolOutput represents the output of a tool call.
type ToolOutput struct {
	Type   string `json:"type"` // "function_call_output"
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

// ResponseMessage extends ChatMessage with tool calls.
type ResponseMessage struct {
	ChatMessage
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

// responseResponse is the response body from the Responses API.
type responseResponse struct {
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
	Output []struct {
		Type      string `json:"type"` // "message", "function_call", etc.
		ID        string `json:"id"`
		Status    string `json:"status"` // "completed", "incomplete", "error"
		Role      string `json:"role,omitempty"`
		CallID    string `json:"call_id,omitempty"`
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
		Content   []struct {
			Type        string `json:"type"`
			Text        string `json:"text"`
			Annotations []any  `json:"annotations"`
		} `json:"content,omitempty"`
	} `json:"output"`

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
	ToolChoice string  `json:"tool_choice"`
	Tools      []Tool  `json:"tools"`
	TopP       float64 `json:"top_p"`
	Truncation string  `json:"truncation"`

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

// ForceToolChoice generates parameter value for ResponseRequest.ToolChoice field that forces the use of one specified tool.
func ForceToolChoice(toolType string, name string) json.RawMessage {
	switch toolType {
	case "function":
		return json.RawMessage(fmt.Sprintf(`{"type": "function", "function": {"name": "%s"}}`, name))
	case "file_search":
		return json.RawMessage(`{"type": "file_search"}`)
	case "web_search_preview":
		return json.RawMessage(`{"type": "web_search_preview"}`)
	case "computer_use_preview":
		return json.RawMessage(`{"type": "computer_use_preview"}`)
	default:
		return json.RawMessage(`"auto"`)
	}
}

// ForceFunction is a convenience function to force the use of a specific function tool.
// This is a backward-compatible wrapper around ForceToolChoice.
func ForceFunction(name string) json.RawMessage {
	return ForceToolChoice("function", name)
}

// countTokens returns the number of tokens in the request.
func (data ResponseRequest) countTokens() int {
	b, err := json.Marshal(data)
	if err != nil {
		panic("failed to marshal request body: " + err.Error())
	}

	return len(b) / charPerToken
}

func (data ResponseRequest) tokenLimit() int {
	max, ok := chatMaxTokens[data.Model]
	if !ok {
		return chatMaxTokens[""]
	}
	return max
}

// trimMessages cuts off the oldest messages if the request is too long.
func trimMessages(messages []ChatMessage, tokenLimit int) []ChatMessage {
	if len(messages) == 0 {
		return messages
	}

	hasSystemPrompt := messages[0].Role == roles.System
	minMessages := 1
	if hasSystemPrompt {
		minMessages = 2
	}

	result := messages
	// Create a temporary request to check token count
	tempReq := ResponseRequest{
		Model: DefaultModel,
		Input: result,
	}

	for len(result) > minMessages && tempReq.countTokens() > tokenLimit {
		newMessages := make([]ChatMessage, 0, len(result)-1)
		if hasSystemPrompt {
			newMessages = append(newMessages, result[0])
		}

		// Skip the oldest non-system message
		startIdx := 1
		if !hasSystemPrompt {
			startIdx = 2
		}

		for i := startIdx; i < len(result); i++ {
			newMessages = append(newMessages, result[i])
		}

		result = newMessages
		tempReq.Input = result
	}

	return result
}

// execute sends request to the Responses API and returns the response.
func (data ResponseRequest) execute() (*responseResponse, error) {
	if data.Model == "" {
		data.Model = DefaultModel
	}

	// Check if we have input
	if data.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	// If input is a slice of ChatMessage, trim it if needed
	if messages, ok := data.Input.([]ChatMessage); ok {
		trimmedMessages := trimMessages(messages, data.tokenLimit())
		data.Input = trimmedMessages
	}

	if tokens := data.countTokens(); tokens > data.tokenLimit() {
		return nil, fmt.Errorf("prompt is likely too long: ~%d tokens, max %d tokens", tokens, data.tokenLimit())
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	if testing.Testing() {
		fmt.Printf("Request body: %s\n", string(b))
	}

	req, err := http.NewRequest(http.MethodPost, responseAPI, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	addHeaders(req)

	var resp *http.Response
	resp, err = cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if testing.Testing() {
		fmt.Printf("Response status: %s\n", resp.Status)
		fmt.Printf("Response body: %s\n", string(body))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(body))
	}

	var res responseResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// checkFirst checks if API response is valid, returns raw content or tool call of first choice and error.
func (resp *responseResponse) checkFirst() (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}

	if resp.Error != nil {
		return "", fmt.Errorf("got API error: %v", resp.Error)
	}

	if len(resp.Output) == 0 {
		return "", fmt.Errorf("no output returned")
	}

	// Handle different output types
	outputType := resp.Output[0].Type

	// Handle message type (text response)
	if outputType == "message" {
		if len(resp.Output[0].Content) == 0 {
			return "", fmt.Errorf("no content in output")
		}

		status := resp.Output[0].Status
		content := resp.Output[0].Content[0].Text

		// Check if status is unexpected
		isValidStatus := status == "" ||
			status == "completed" ||
			status == "incomplete" ||
			status == "error"

		if !isValidStatus {
			return content, fmt.Errorf("got unexpected status: %s", status)
		}

		return content, nil
	}

	// Handle function call type
	if outputType == "function_call" {
		return fmt.Sprintf("Function call: %s(%s)", resp.Output[0].Name, resp.Output[0].Arguments), nil
	}

	// Handle file search call type
	if outputType == "file_search_call" {
		return fmt.Sprintf("File search call: %s", resp.Output[0].ID), nil
	}

	// Handle web search call type
	if outputType == "web_search_call" {
		return fmt.Sprintf("Web search call: %s", resp.Output[0].ID), nil
	}

	// Handle computer call type
	if outputType == "computer_call" {
		return fmt.Sprintf("Computer call: %s", resp.Output[0].ID), nil
	}

	// Handle reasoning type
	if outputType == "reasoning" {
		return fmt.Sprintf("Reasoning: %s", resp.Output[0].ID), nil
	}

	return "", fmt.Errorf("unsupported output type: %s", outputType)
}

// toolRegistry stores all registered tools.
var toolRegistry = map[string]Tool{}

// RegisterTool registers a tool that can be used by the model.
// To allow the model to use a tool in a particular request,
// add the tool to the request's "tools" field.
func RegisterTool(tool Tool) {
	// Validate the tool based on its type
	switch tool.Type {
	case "function":
		// Function tools require name, description, and parameters
		if tool.Function.Name == "" || tool.Function.ParamsSchema == nil || tool.Function.Description == "" {
			panic("tool function '" + tool.Name + "' is missing required field(s)")
		}

		// Set the Name field to match the Function.Name if not already set
		if tool.Name == "" {
			tool.Name = tool.Function.Name
		}

	case "file_search":
		// File search tools require vector_store_ids
		if len(tool.VectorStoreIDs) == 0 {
			panic("file_search tool requires vector_store_ids")
		}

	case "web_search_preview":
		// Web search preview doesn't require additional fields

	case "computer_use_preview":
		// Computer use preview can have display dimensions and environment
		// No specific validation required

	default:
		panic("unsupported tool type: " + tool.Type)
	}

	// Check for duplicate tool names
	if _, ok := toolRegistry[tool.Name]; ok {
		panic("tool '" + tool.Name + "' is already registered, names must be unique")
	}

	toolRegistry[tool.Name] = tool
}

// CountTools returns the number of registered tools.
func CountTools() int {
	return len(toolRegistry)
}

// SingleResponsePrompt sends a request to the Responses API
// with a single user prompt and no additional context or settings.
// Returns the AI reply, request ID, and any error.
func SingleResponsePrompt(prompt, userID string) (string, string, error) {
	req := ResponseRequest{
		Model: DefaultModel,
		Input: prompt,
		User:  userID,
	}

	return CustomResponsePrompt(&req)
}

// PrimedResponsePrompt sends a request to the Responses API
// with a single user prompt primed by a given "system" message.
// Returns the AI reply, request ID, and any error.
func PrimedResponsePrompt(systemMessage, prompt, userID string) (string, string, error) {
	// For the Responses API, we need to format the system message and prompt together
	messages := []ChatMessage{
		{Role: roles.System, Content: systemMessage},
		{Role: roles.User, Content: prompt},
	}

	req := ResponseRequest{
		Model: DefaultModel,
		Input: messages,
		User:  userID,
	}

	return CustomResponsePrompt(&req)
}

// MessagesResponsePrompt sends a request to the Responses API with a given sequence of messages.
// Returns the AI reply, request ID, and any error.
func MessagesResponsePrompt(messages []ChatMessage, userID string) (string, string, error) {
	req := ResponseRequest{
		Model: DefaultModel,
		Input: messages,
		User:  userID,
	}

	return CustomResponsePrompt(&req)
}

// Clone creates a copy of the ResponseRequest with all fields copied.
func (data *ResponseRequest) Clone() *ResponseRequest {
	clone := *data // Shallow copy

	// Deep copy slices and maps if needed
	if data.Include != nil {
		clone.Include = make([]string, len(data.Include))
		copy(clone.Include, data.Include)
	}

	if data.Metadata != nil {
		clone.Metadata = make(map[string]string, len(data.Metadata))
		for k, v := range data.Metadata {
			clone.Metadata[k] = v
		}
	}

	if data.Tools != nil {
		clone.Tools = make([]Tool, len(data.Tools))
		copy(clone.Tools, data.Tools)
	}

	// Copy any other reference types as needed

	return &clone
}

// CustomResponsePrompt sends a request to the Responses API with custom data.
// Returns the AI reply, request ID, and any error.
func CustomResponsePrompt(req *ResponseRequest) (string, string, error) {
	respData, err := req.execute()
	if err != nil {
		return "", "", err
	}

	// Check if we have output
	if len(respData.Output) == 0 {
		return "", "", fmt.Errorf("no output returned")
	}

	// Check for function calls and text responses in the output
	type returnedCall struct {
		CallID    string
		Name      string
		Arguments string
	}
	var functionCalls []returnedCall

	var toolOutputs []ToolOutput
	for _, output := range respData.Output {
		switch output.Type {
		case "message":
			// check content
			if len(output.Content) == 0 {
				return "", "", fmt.Errorf("text output with no content")
			}

			// if we got only one output and it's text, return it
			if len(respData.Output) == 1 {
				content, err := respData.checkFirst()
				return content, respData.ID, err
			}

			// for a text output given alongside other outputs, we have a handler
			if req.IntermediateTextHandler != nil {
				req.IntermediateTextHandler(output.Content[0].Text)
			}
		case "function_call":
			if req.ReturnToolCalls {
				// if calls should be returned, gather them up
				functionCalls = append(functionCalls, returnedCall{
					CallID:    output.CallID,
					Name:      output.Name,
					Arguments: output.Arguments,
				})
				continue
			}

			// Get the function from the registered function calls
			f, ok := funcCalls[output.Name]
			if !ok || f.F == nil {
				return "", "", fmt.Errorf(
					"function '%s' is not registered or has no implementation",
					output.Name,
				)
			}

			// Execute the function
			fResult, err := f.F([]byte(output.Arguments))
			switch {
			case err == nil:
			case errors.Is(err, ErrDoNotRespond):
				// here we return ID despite error
				// because this error indicates intended behavior
				return "", respData.ID, err
			default:
				return "", "", fmt.Errorf(
					"failed to execute function '%s': %w",
					output.Name, err,
				)
			}

			// Add the tool output
			toolOutputs = append(toolOutputs, ToolOutput{
				Type:   "function_call_output",
				CallID: output.CallID,
				Output: fResult,
			})
		default:
			return "", "", fmt.Errorf("unsupported output type: %s", output.Type)
		}
	}

	// If ReturnToolCalls is true and we have function calls, encode them into JSON and return them
	if req.ReturnToolCalls && len(functionCalls) > 0 {
		var toolCalls []ToolCall
		for _, fc := range functionCalls {
			toolCalls = append(toolCalls, ToolCall{
				ID:   fc.CallID,
				Type: "function",
				Function: FunctionCallData{
					Name:      fc.Name,
					Arguments: fc.Arguments,
				},
			})
		}

		b, err := json.Marshal(toolCalls)
		if err != nil {
			return "", "", fmt.Errorf("failed to marshal tool calls: %w", err)
		}

		return string(b), respData.ID, nil
	}

	// If we have tool outputs, send them to the API in a follow-up request
	if len(toolOutputs) > 0 {
		// Create a follow-up request by cloning the original request
		followUpReq := req.Clone()

		// Update only the fields that need to change
		followUpReq.Input = toolOutputs
		followUpReq.PreviousResponseID = respData.ID

		// Make another request with the function results
		response, respID, err := CustomResponsePrompt(followUpReq)
		if err != nil {
			return "", "", err
		}
		return response, respID, nil
	}

	// this place should be unreachable
	return "", "", fmt.Errorf("logic error: unreachable code")
}
