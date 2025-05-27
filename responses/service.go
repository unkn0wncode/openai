// Package responses / service.go contains the service layer for OpenAI responses API.
package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/unkn0wncode/openai/content/input"
	"github.com/unkn0wncode/openai/content/output"
)

const (
	// Text format types
	TextFormatTypeText       = "text"
	TextFormatTypeJSONObject = "json_object"
	TextFormatTypeJSONSchema = "json_schema"
)

// Service is the service layer for OpenAI responses API.
type Service interface {
	// Send sends a request to the Responses API.
	Send(req *Request) (response *Response, err error)

	// NewMessage creates a new empty message.
	NewMessage() *output.Message

	// NewRequest creates a new empty request.
	NewRequest() *Request

	// Poll continuously fetches a background response until completion or failure.
	// ctx controls cancellation; interval is time to wait between subsequent polls.
	Poll(ctx context.Context, responseID string, interval time.Duration) (*Response, error)
}

// Content is an interface listing all types that can be used as content in Responses API.
// These types may appear in `Request.Input`, `Response.Outputs`,
// and `output.Message.Content` fields.
type Content interface {
	string |
		input.InputText |
		input.InputImage |
		input.InputFile |
		output.OutputText |
		output.Refusal |
		output.FileSearchCall |
		output.ComputerCall |
		output.ComputerCallOutput |
		output.WebSearchCall |
		output.FunctionCall |
		output.FunctionCallOutput |
		output.Reasoning |
		input.ItemReference
}

// Request is the request body for the Responses API.
type Request struct {
	// Required
	Model string `json:"model"`
	Input any    `json:"input"` // string or []Any

	// Optional
	Include            []string          `json:"include,omitempty"`              // Additional data to include in response: "file_search_call.results", "message.input_image.image_url", "computer_call_output.output.image_url"
	Instructions       string            `json:"instructions,omitempty"`         // System message for context
	MaxOutputTokens    int               `json:"max_output_tokens,omitempty"`    // Max tokens to generate
	Metadata           map[string]string `json:"metadata,omitempty"`             // Key-value pairs
	ParallelToolCalls  *bool             `json:"parallel_tool_calls,omitempty"`  // Allow parallel tool calls, default true
	PreviousResponseID string            `json:"previous_response_id,omitempty"` // ID of previous response
	Reasoning          *ReasoningConfig  `json:"reasoning,omitempty"`            // Reasoning configuration
	Store              *bool             `json:"store,omitempty"`                // Whether to store the response, default true
	Stream             bool              `json:"stream,omitempty"`               // Stream the response, default false
	Temperature        float64           `json:"temperature,omitempty"`          // default 1
	Text               *TextFormat       `json:"text,omitempty"`                 // Text format configuration
	ToolChoice         json.RawMessage   `json:"tool_choice,omitempty"`          // default "auto", can be "none", "required", or an object
	TopP               float64           `json:"top_p,omitempty"`                // default 1
	Truncation         string            `json:"truncation,omitempty"`           // "auto" or "disabled"
	User               string            `json:"user,omitempty"`                 // default ""
	Background         bool              `json:"background,omitempty"`           // if true, the API returns immediately with only a response ID

	// names of tools/functions to include, will be marshaled as their full structs from tools registry
	Tools []string `json:"-"`

	// Custom (not part of the API)
	// If set, tool calls will be returned instead of executed.
	ReturnToolCalls bool `json:"-"` // default false
	// If set, will be called on messages received alongside other outputs (e.g., tool calls)
	// that would otherwise be returned in the response but can be handled sooner with this handler.
	IntermediateMessageHandler func(output.Message) `json:"-"`
}

// Clone creates a copy of the ResponseRequest with all fields copied.
func (data *Request) Clone() *Request {
	clone := *data // Shallow copy

	// Deep copy slices and maps if needed
	if data.Include != nil {
		clone.Include = make([]string, len(data.Include))
		copy(clone.Include, data.Include)
	}

	if data.Metadata != nil {
		clone.Metadata = make(map[string]string, len(data.Metadata))
		maps.Copy(clone.Metadata, data.Metadata)
	}

	if data.Tools != nil {
		clone.Tools = make([]string, len(data.Tools))
		copy(clone.Tools, data.Tools)
	}

	// Copy any other reference types as needed

	return &clone
}

// Response is a wrapper for outputs returned from the Responses API.
type Response struct {
	ID            string
	Outputs       []output.Any
	ParsedOutputs []any
}

// Parse parses the []output.Any and places the parsed objects in ParsedOutputs.
func (r *Response) Parse() error {
	r.ParsedOutputs = nil
	for _, o := range r.Outputs {
		parsed, err := o.Unmarshal()
		if err != nil {
			return err
		}

		r.ParsedOutputs = append(r.ParsedOutputs, parsed)
	}
	return nil
}

// Texts returns a slice of strings gathered from text output objects in the response.
// Parsing of the content is done automatically if not already done, and errors are ignored. To
// have errors checked, use Response.Parse() first.
func (r *Response) Texts() []string {
	if r.ParsedOutputs == nil {
		r.Parse() // ignored error check here
	}

	var texts []string
	for _, o := range r.ParsedOutputs {
		if msg, ok := o.(output.Message); ok {
			if msg.Content == nil {
				continue
			}

			if text, ok := msg.Content.(string); ok {
				texts = append(texts, text)
				continue
			}

			for _, content := range msg.Content.([]any) { // this type assertion is safe because all other cases are checked, mostly during unmarshalling
				if text, ok := content.(output.OutputText); ok {
					texts = append(texts, text.String())
				}
			}
		}
	}

	return texts
}

// JoinedTexts returns a single string joined from all text outputs in the response with newlines.
// Normally there's only one text output.
func (r *Response) JoinedTexts() string {
	return strings.Join(r.Texts(), "\n")
}

// FirstText returns the first text output in the response, or an empty string.
func (r *Response) FirstText() string {
	texts := r.Texts()
	if len(texts) == 0 {
		return ""
	}
	return texts[0]
}

// LastText returns the last text output in the response, or an empty string.
func (r *Response) LastText() string {
	texts := r.Texts()
	if len(texts) == 0 {
		return ""
	}
	return texts[len(texts)-1]
}

// FunctionCalls returns a slice of FunctionCall objects from the response.
func (r *Response) FunctionCalls() []output.FunctionCall {
	if r.ParsedOutputs == nil {
		r.Parse() // ignored error check here
	}

	var functionCalls []output.FunctionCall
	for _, o := range r.ParsedOutputs {
		if call, ok := o.(output.FunctionCall); ok {
			functionCalls = append(functionCalls, call)
		}
	}
	return functionCalls
}

// Refusals returns a slice of Refusal objects from the response.
func (r *Response) Refusals() []string {
	if r.ParsedOutputs == nil {
		r.Parse() // ignored error check here
	}
	var refusals []string
	for _, o := range r.ParsedOutputs {
		if ms, ok := o.(output.Message); ok {
			if _, ok := ms.Content.([]any); !ok {
				continue
			}

			for _, content := range ms.Content.([]any) {
				if refusal, ok := content.(output.Refusal); ok {
					refusals = append(refusals, refusal.String())
				}
			}
		}
	}
	return refusals
}

// Reasonings returns a slice of Reasoning objects from the response.
func (r *Response) Reasonings() []output.Reasoning {
	if r.ParsedOutputs == nil {
		r.Parse()
	}
	var reasonings []output.Reasoning
	for _, o := range r.ParsedOutputs {
		if rr, ok := o.(output.Reasoning); ok {
			reasonings = append(reasonings, rr)
		}
	}
	return reasonings
}

// ReasoningSummaries returns a slice of summary texts from reasoning outputs.
func (r *Response) ReasoningSummaries() []string {
	var summaries []string
	for _, rr := range r.Reasonings() {
		for _, s := range rr.Summary {
			summaries = append(summaries, s.Text)
		}
	}
	return summaries
}

// JoinedReasoningSummaries returns all reasoning summaries joined by newlines.
func (r *Response) JoinedReasoningSummaries() string {
	return strings.Join(r.ReasoningSummaries(), "\n")
}

// MCPApprovalRequests returns a slice of MCPApprovalRequest objects from the response.
func (r *Response) MCPApprovalRequests() []output.MCPApprovalRequest {
	if r.ParsedOutputs == nil {
		r.Parse() // ignored error check here
	}
	var approvalRequests []output.MCPApprovalRequest
	for _, o := range r.ParsedOutputs {
		if approvalRequest, ok := o.(output.MCPApprovalRequest); ok {
			approvalRequests = append(approvalRequests, approvalRequest)
		}
	}
	return approvalRequests
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

// ForceToolChoice generates parameter value for ResponseRequest.ToolChoice field that forces the use of one specified tool.
func ForceToolChoice(toolType string, name string) json.RawMessage {
	switch toolType {
	case "function":
		return json.RawMessage(fmt.Sprintf(`{"type": "function", "name": "%s"}`, name))
	case "file_search":
		return json.RawMessage(`{"type": "file_search"}`)
	case "web_search_preview":
		return json.RawMessage(`{"type": "web_search_preview"}`)
	case "computer_use_preview":
		return json.RawMessage(`{"type": "computer_use_preview"}`)
	case "mcp":
		return json.RawMessage(`{"type": "mcp"}`)
	case "local_shell":
		return json.RawMessage(`{"type": "local_shell"}`)
	default:
		return json.RawMessage(`"auto"`)
	}
}

// ForceFunction is a convenience function to force the use of a specific function tool.
// This is a backward-compatible wrapper around ForceToolChoice.
func ForceFunction(name string) json.RawMessage {
	return ForceToolChoice("function", name)
}
