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
	"github.com/unkn0wncode/openai/responses/streaming"
)

const (
	// Text format types
	TextFormatTypeText       = "text"
	TextFormatTypeJSONObject = "json_object"
	TextFormatTypeJSONSchema = "json_schema"

	// Service tiers
	ServiceTierAuto     = "auto"     // service tier configured in the Project settings
	ServiceTierDefault  = "default"  // standard pricing and performance for the selected model
	ServiceTierFlex     = "flex"     // slower but cheaper
	ServiceTierPriority = "priority" // faster but more expensive
)

// Service is the service layer for OpenAI responses API.
type Service interface {
	// Send sends a request to the Responses API.
	Send(req *Request) (response *Response, err error)

	// Stream sends a request with parameter "stream":true and returns a streaming iterator.
	Stream(ctx context.Context, req *Request) (*streaming.StreamIterator, error)

	// NewMessage creates a new empty message.
	NewMessage() *output.Message

	// NewRequest creates a new empty request.
	NewRequest() *Request

	// Poll continuously fetches a background response until completion or failure.
	// ctx controls cancellation; interval is time to wait between subsequent polls.
	Poll(ctx context.Context, responseID string, interval time.Duration) (*Response, error)

	// CreateConversation creates a new persistent conversation container.
	CreateConversation(metadata map[string]string, items ...any) (*Conversation, error)

	// Conversation retrieves a conversation by ID.
	Conversation(id string) (*Conversation, error)
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
		output.CustomToolCall |
		output.CustomToolCallOutput |
		output.Reasoning |
		output.ApplyPatchCall |
		output.ApplyPatchCallOutput |
		output.ShellCall |
		output.ShellCallOutput |
		input.ItemReference
}

// Request is the request body for the Responses API.
type Request struct {
	// Required
	Model string `json:"model"`
	Input any    `json:"input"` // string or []Any

	// Optional
	Include              []string          `json:"include,omitempty"`                // Additional data to include in response: "file_search_call.results", "message.input_image.image_url", "computer_call_output.output.image_url"
	Instructions         string            `json:"instructions,omitempty"`           // System message for context
	Conversation         any               `json:"conversation,omitempty"`           // ID or a Conversation object containing an ID
	MaxOutputTokens      int               `json:"max_output_tokens,omitempty"`      // Max tokens to generate
	Metadata             map[string]string `json:"metadata,omitempty"`               // Key-value pairs
	ParallelToolCalls    *bool             `json:"parallel_tool_calls,omitempty"`    // Allow parallel tool calls, default true
	PreviousResponseID   string            `json:"previous_response_id,omitempty"`   // ID of previous response
	Prompt               *Prompt           `json:"prompt,omitempty"`                 // Reference to a prompt template and its variables
	PromptCacheKey       string            `json:"prompt_cache_key,omitempty"`       // Used for matching similar requests with cached input
	PromptCacheRetention string            `json:"prompt_cache_retention,omitempty"` // Prompt cache retention policy: "in_memory" (default) or "24h"
	Reasoning            *ReasoningConfig  `json:"reasoning,omitempty"`              // Reasoning configuration
	SafetyIdentifier     string            `json:"safety_identifier,omitempty"`      // Stable unique identifier for end user, preferably anonymized
	ServiceTier          string            `json:"service_tier,omitempty"`           // Service tier to use, default "auto"
	Store                *bool             `json:"store,omitempty"`                  // Whether to store the response, default true
	Stream               bool              `json:"stream,omitempty"`                 // Stream the response, default false
	StreamOptions        *StreamOptions    `json:"stream_options,omitempty"`         // Streaming configuration
	Temperature          float64           `json:"temperature,omitempty"`            // default 1
	Text                 *TextOptions      `json:"text,omitempty"`                   // Text format configuration
	ToolChoice           json.RawMessage   `json:"tool_choice,omitempty"`            // default "auto", can be "none", "required", or an object
	TopP                 float64           `json:"top_p,omitempty"`                  // default 1
	Truncation           string            `json:"truncation,omitempty"`             // "auto" or "disabled"
	User                 string            `json:"user,omitempty"`                   // Deprecated: use SafetyIdentifier and PromptCacheKey instead
	Background           bool              `json:"background,omitempty"`             // if true, the API returns immediately with only a response ID

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

// Prompt is a reference to a prompt template and its variables.
type Prompt struct {
	ID        string                     `json:"id"`
	Variables map[string]json.RawMessage `json:"variables"`
	Version   *string                    `json:"version"`
}

// StreamOptions is a set of options for streaming responses.
type StreamOptions struct {
	// Stream obfuscation adds random characters to an obfuscation field on streaming delta events
	// to normalize payload sizes as a mitigation to certain side-channel attacks.
	// These obfuscation fields are included by default, but add a small amount of overhead
	// to the data stream.
	// You can set include_obfuscation to false to optimize for bandwidth if you trust the network
	// links between your application and the OpenAI API.
	IncludeObfuscation bool `json:"include_obfuscation,omitempty"`
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
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
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
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}

	var functionCalls []output.FunctionCall
	for _, o := range r.ParsedOutputs {
		if call, ok := o.(output.FunctionCall); ok {
			functionCalls = append(functionCalls, call)
		}
	}
	return functionCalls
}

// CustomToolCalls returns a slice of CustomToolCall objects from the response.
func (r *Response) CustomToolCalls() []output.CustomToolCall {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}

	var customFunctionCalls []output.CustomToolCall
	for _, o := range r.ParsedOutputs {
		if call, ok := o.(output.CustomToolCall); ok {
			customFunctionCalls = append(customFunctionCalls, call)
		}
	}
	return customFunctionCalls
}

// Refusals returns a slice of Refusal objects from the response.
func (r *Response) Refusals() []string {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
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
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
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
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}
	var approvalRequests []output.MCPApprovalRequest
	for _, o := range r.ParsedOutputs {
		if approvalRequest, ok := o.(output.MCPApprovalRequest); ok {
			approvalRequests = append(approvalRequests, approvalRequest)
		}
	}
	return approvalRequests
}

// Conversation represents a persisted conversation container on the server.
// It embeds the ConversationCli interface and implements the Conversation object methods.
type Conversation struct {
	ConversationCli `json:"-"` // implements API methods of the Conversation object

	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int               `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ConversationCli is a client that implements API methods of the Conversation object.
type ConversationCli interface {
	// Update sends the current state of the conversation to the API.
	// Effectively saves changes in the metadata.
	Update() error

	// Delete removes the conversation from the API.
	Delete() error

	// ListItems retrieves items stored in the conversation.
	ListItems(opts *ConversationListOptions) (*ConversationItemList, error)

	// AppendItems adds new items to the conversation.
	AppendItems(include *ConversationItemsInclude, items ...any) (*ConversationItemList, error)

	// Item retrieves a single item from the conversation.
	Item(include *ConversationItemsInclude, itemID string) (any, error)

	// DeleteItem removes a single item from the conversation.
	DeleteItem(itemID string) error
}

// ConversationListOptions configures pagination when listing conversation items.
type ConversationListOptions struct {
	Limit   int
	FirstID string
	LastID  string
	Include *ConversationItemsInclude
}

// ConversationItemsInclude lists true/false flags for which items to include in a
// ConversationItemList response. It prepares a slice of flags set to true for URL query values.
type ConversationItemsInclude struct {
	// Include the sources of the web search tool call.
	WebSearchCallActionSources bool

	// Includes the outputs of python code execution in code interpreter tool call items.
	CodeInterpreterCallOutputs bool

	// Include image urls from the computer call output.
	ComputerCallOutputImageURL bool

	// Include the search results of the file search tool call.
	FileSearchCallResults bool

	// Include image urls from the input message.
	MessageInputImageURL bool

	// Include logprobs with assistant messages.
	MessageOutputTextLogprobs bool

	// Includes an encrypted version of reasoning tokens in reasoning item outputs.
	// This enables reasoning items to be used in multi-turn conversations when using the Responses
	// API statelessly (like when the store parameter is set to false, or when an organization is
	// enrolled in the zero data retention program).
	ReasoningEncryptedContent bool
}

// Values returns a slice of strings for flags set to true.
func (i ConversationItemsInclude) Values() []string {
	var flags []string
	if i.WebSearchCallActionSources {
		flags = append(flags, "web_search_call.action.sources")
	}
	if i.CodeInterpreterCallOutputs {
		flags = append(flags, "code_interpreter_call.outputs")
	}
	if i.ComputerCallOutputImageURL {
		flags = append(flags, "computer_call_output.output.image_url")
	}
	if i.FileSearchCallResults {
		flags = append(flags, "file_search_call.results")
	}
	if i.MessageInputImageURL {
		flags = append(flags, "message.input_image.image_url")
	}
	if i.MessageOutputTextLogprobs {
		flags = append(flags, "message.output_text.logprobs")
	}
	if i.ReasoningEncryptedContent {
		flags = append(flags, "reasoning.encrypted_content")
	}
	return flags
}

// ConversationItemList is the paginated response returned when listing conversation items.
type ConversationItemList struct {
	Object  string       `json:"object"` // always "list"
	Data    []output.Any `json:"data"`
	FirstID string       `json:"first_id"`
	LastID  string       `json:"last_id"`
	HasMore bool         `json:"has_more"`

	ParsedData []any `json:"-"` // parsed data from the Data field
}

// Parse parses the []output.Any and places the parsed objects in ParsedData.
func (l *ConversationItemList) Parse() error {
	l.ParsedData = nil

	for _, o := range l.Data {
		parsed, err := o.Unmarshal()
		if err != nil {
			return err
		}
		l.ParsedData = append(l.ParsedData, parsed)
	}

	return nil
}

// ShellCalls returns a slice of ShellCall objects from the response.
func (r *Response) ShellCalls() []output.ShellCall {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}
	var shellCalls []output.ShellCall
	for _, o := range r.ParsedOutputs {
		if call, ok := o.(output.ShellCall); ok {
			shellCalls = append(shellCalls, call)
		}
	}
	return shellCalls
}

// ApplyPatchCalls returns a slice of ApplyPatchCall objects from the response.
func (r *Response) ApplyPatchCalls() []output.ApplyPatchCall {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}
	var applyPatchCalls []output.ApplyPatchCall
	for _, o := range r.ParsedOutputs {
		if call, ok := o.(output.ApplyPatchCall); ok {
			applyPatchCalls = append(applyPatchCalls, call)
		}
	}
	return applyPatchCalls
}

// ReasoningConfig represents configuration options for reasoning models.
type ReasoningConfig struct {
	Effort          string `json:"effort,omitempty"`           // "none", "minimal", "low", "medium", or "high"
	GenerateSummary string `json:"generate_summary,omitempty"` // "concise" or "detailed"
}

// TextOptions represents the format configuration for text responses.
type TextOptions struct {
	Format    TextFormatType `json:"format"`
	Verbosity string         `json:"verbosity,omitempty"` // "low", "medium", or "high", default "medium"
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
	case "custom":
		return json.RawMessage(fmt.Sprintf(`{"type": "custom", "name": "%s"}`, name))
	case "file_search":
		return json.RawMessage(`{"type": "file_search"}`)
	case "web_search", "web_search_preview":
		return json.RawMessage(`{"type": "web_search"}`)
	case "computer_use_preview":
		return json.RawMessage(`{"type": "computer_use_preview"}`)
	case "mcp":
		return json.RawMessage(`{"type": "mcp"}`)
	case "local_shell":
		return json.RawMessage(`{"type": "local_shell"}`)
	case "code_interpreter":
		return json.RawMessage(`{"type": "code_interpreter"}`)
	case "shell":
		return json.RawMessage(`{"type": "shell"}`)
	case "apply_patch":
		return json.RawMessage(`{"type": "apply_patch"}`)
	default:
		return json.RawMessage(`"auto"`)
	}
}

// ForceFunction is a convenience function to force the use of a specific function tool.
// This is a backward-compatible wrapper around ForceToolChoice.
func ForceFunction(name string) json.RawMessage {
	return ForceToolChoice("function", name)
}
