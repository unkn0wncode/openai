// Package responses / service.go contains the service layer for OpenAI responses API.
package responses

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/unkn0wncode/openai/content/input"
	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/tools"
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
	ServiceTierFast     = "fast"     // same request behavior as priority
)

// Service is the service layer for OpenAI responses API.
type Service interface {
	// Send sends a request to the Responses API.
	Send(req *Request) (response *Response, err error)

	// CountInputTokens counts the input for a request without generating a response.
	CountInputTokens(ctx context.Context, req *Request) (int, error)

	// Stream sends a request with parameter "stream":true and returns a streaming iterator.
	Stream(ctx context.Context, req *Request) (*streaming.StreamIterator, error)

	// WebSocket opens a persistent connection, optionally enabling beta features
	// through the OpenAI-Beta header.
	WebSocket(ctx context.Context, betas ...string) (WSConn, error)

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
		output.Compaction |
		output.ApplyPatchCall |
		output.ApplyPatchCallOutput |
		output.ShellCall |
		output.ShellCallOutput |
		output.ImageGenerationCall |
		output.Program |
		output.ProgramOutput |
		output.MultiAgentCall |
		output.MultiAgentCallOutput |
		output.AgentMessage |
		input.ItemReference |
		input.ConfigurationUpdate
}

// Request is the request body for the Responses API.
type Request struct {
	// Required
	Model string `json:"model"`
	Input any    `json:"input"` // string or []Any

	// Optional
	Include              []string            `json:"include,omitempty"`                // Additional data to include in response: "file_search_call.results", "message.input_image.image_url", "computer_call_output.output.image_url"
	Instructions         string              `json:"instructions,omitempty"`           // System message for context
	Conversation         any                 `json:"conversation,omitempty"`           // ID or a Conversation object containing an ID
	ContextManagement    []ContextConfig     `json:"context_management,omitempty"`     // Compaction configuration
	MaxOutputTokens      int                 `json:"max_output_tokens,omitempty"`      // Max tokens to generate
	Metadata             map[string]string   `json:"metadata,omitempty"`               // Key-value pairs
	ParallelToolCalls    *bool               `json:"parallel_tool_calls,omitempty"`    // Allow parallel tool calls, default true
	PreviousResponseID   string              `json:"previous_response_id,omitempty"`   // ID of previous response
	Prompt               *Prompt             `json:"prompt,omitempty"`                 // Reference to a prompt template and its variables
	PromptCacheKey       string              `json:"prompt_cache_key,omitempty"`       // Used for matching similar requests with cached input
	PromptCacheRetention string              `json:"prompt_cache_retention,omitempty"` // Legacy maximum retention policy: "in_memory" or "24h"; independent of PromptCacheOptions.TTL
	PromptCacheOptions   *PromptCacheOptions `json:"prompt_cache_options,omitempty"`
	Reasoning            *ReasoningConfig    `json:"reasoning,omitempty"`         // Reasoning configuration
	MultiAgent           *MultiAgentConfig   `json:"multi_agent,omitempty"`       // Enables the Responses multi-agent beta
	SafetyIdentifier     string              `json:"safety_identifier,omitempty"` // Stable unique identifier for end user, preferably anonymized
	ServiceTier          string              `json:"service_tier,omitempty"`      // Service tier to use, default "auto"
	Store                *bool               `json:"store,omitempty"`             // Whether to store the response, default true
	Stream               bool                `json:"stream,omitempty"`            // Stream the response, default false
	StreamOptions        *StreamOptions      `json:"stream_options,omitempty"`    // Streaming configuration
	Temperature          float64             `json:"temperature,omitempty"`       // default 1
	Text                 *TextOptions        `json:"text,omitempty"`              // Text format configuration
	ToolChoice           json.RawMessage     `json:"tool_choice,omitempty"`       // default "auto", can be "none", "required", or an object
	TopP                 float64             `json:"top_p,omitempty"`             // default 1
	Truncation           string              `json:"truncation,omitempty"`        // "auto" or "disabled"
	User                 string              `json:"user,omitempty"`              // Deprecated: use SafetyIdentifier and PromptCacheKey instead
	Background           bool                `json:"background,omitempty"`        // if true, the API returns immediately with only a response ID
	Generate             *bool               `json:"generate,omitempty"`          // if false, warm up request state without generating model output

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

	if data.ContextManagement != nil {
		clone.ContextManagement = slices.Clone(data.ContextManagement)
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
	ID                     string
	Model                  string
	ServiceTier            string
	Status                 string
	Outputs                []output.Any
	ParsedOutputs          []any
	Usage                  *Usage
	Tools                  []tools.Tool
	Reasoning              *ReasoningConfig
	MultiAgent             *MultiAgentConfig
	PromptCacheOptions     *PromptCacheOptions
	PromptCacheDiagnostics *PromptCacheDiagnostics

	// Calls contains the individual API responses observed by Send, including
	// automatic tool follow-ups, or the latest response returned by Poll.
	// Their outputs are not combined, and their own Calls slices are empty.
	Calls []Response
	// BillingIncomplete means an API request was sent without observable usage.
	// It is independent of the API's response status.
	BillingIncomplete bool
	// ProcessingRegion is the region recorded from the API endpoint. "global"
	// means the global endpoint; an empty value means the routing is unknown.
	ProcessingRegion string
	// EstimatedCost and CostError are populated by Request.EstimateCost.
	// If CostError is non-nil, EstimatedCost is only the known subtotal in USD.
	EstimatedCost float64
	CostError     error
}

// Usage contains token usage for a response.
type Usage models.Usage

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

// isFinalAnswer selects assistant answers, excluding commentary and subagent messages.
func isFinalAnswer(message output.Message) bool {
	return message.Role == "assistant" && (message.Phase == "" || message.Phase == "final_answer") &&
		(message.Agent == nil || message.Agent.AgentName == "/root")
}

// Texts returns text from the root assistant's final answers.
// The agent must be absent or /root, and the phase must be absent or final_answer.
// Outputs and ParsedOutputs are unchanged by this selection.
// Parsing of the content is done automatically if not already done, and errors are ignored. To
// have errors checked, use Response.Parse() first.
func (r *Response) Texts() []string {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}

	var texts []string
	for _, o := range r.ParsedOutputs {
		if msg, ok := o.(output.Message); ok && isFinalAnswer(msg) {
			if msg.Content == nil {
				continue
			}

			if text, ok := msg.Content.(string); ok {
				texts = append(texts, text)
				continue
			}

			// this type assertion is safe because all other cases are checked, mostly during unmarshalling
			// a check without comma ok is better than a continue because it will surface an error as a panic
			for _, content := range msg.Content.([]any) {
				if text, ok := content.(output.OutputText); ok {
					texts = append(texts, text.String())
				}
			}
		}
	}

	return texts
}

// JoinedTexts returns the root assistant's final texts joined with newlines.
// It uses the same message selection as Texts.
func (r *Response) JoinedTexts() string {
	return strings.Join(r.Texts(), "\n")
}

// FirstText returns the first text selected by Texts, or an empty string.
func (r *Response) FirstText() string {
	texts := r.Texts()
	if len(texts) == 0 {
		return ""
	}
	return texts[0]
}

// LastText returns the last text selected by Texts, or an empty string.
func (r *Response) LastText() string {
	texts := r.Texts()
	if len(texts) == 0 {
		return ""
	}
	return texts[len(texts)-1]
}

// FunctionCalls returns function calls from all agents, preserving their attribution.
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

// CustomToolCalls returns custom tool calls from all agents, preserving their attribution.
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

// Refusals returns refusal texts from the root assistant's final answers.
// It uses the same message selection as Texts.
func (r *Response) Refusals() []string {
	if r.ParsedOutputs == nil {
		//nolint:errcheck // error intentionally ignored because there's no logger for it and it's not critical
		r.Parse()
	}
	var refusals []string
	for _, o := range r.ParsedOutputs {
		if ms, ok := o.(output.Message); ok && isFinalAnswer(ms) {
			if _, ok := ms.Content.([]any); !ok {
				continue
			}

			// this type assertion is safe because all other cases are checked, mostly during unmarshalling
			// a check without comma ok is better than a continue because it will surface an error as a panic
			for _, content := range ms.Content.([]any) {
				if refusal, ok := content.(output.Refusal); ok {
					refusals = append(refusals, refusal.String())
				}
			}
		}
	}
	return refusals
}

// Reasonings returns reasoning objects from all agents, preserving their attribution.
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

// ReasoningSummaries returns summary texts from the root assistant's reasoning.
// Reasoning without agent metadata is included. Use Reasonings for all agents' summaries.
func (r *Response) ReasoningSummaries() []string {
	var summaries []string
	for _, rr := range r.Reasonings() {
		if rr.Agent != nil && rr.Agent.AgentName != "/root" {
			continue
		}
		for _, s := range rr.Summary {
			summaries = append(summaries, s.Text)
		}
	}
	return summaries
}

// JoinedReasoningSummaries returns the root assistant's reasoning summaries joined by newlines.
func (r *Response) JoinedReasoningSummaries() string {
	return strings.Join(r.ReasoningSummaries(), "\n")
}

// MCPApprovalRequests returns approval requests from all agents, preserving their attribution.
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

// ShellCalls returns shell calls from all agents, preserving their attribution.
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

// ApplyPatchCalls returns patch calls from all agents, preserving their attribution.
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
	Effort          string `json:"effort,omitempty"`           // "none", "minimal", "low", "medium", "high", "xhigh", or "max"; model-dependent
	Mode            string `json:"mode,omitempty"`             // "standard" or "pro"
	Context         string `json:"context,omitempty"`          // "auto", "current_turn", or "all_turns"
	Summary         string `json:"summary,omitempty"`          // "auto", "concise", or "detailed"
	GenerateSummary string `json:"generate_summary,omitempty"` // Deprecated: use Summary instead
}

// PromptCacheOptions controls cache breakpoints and optional reuse diagnostics.
type PromptCacheOptions struct {
	Mode                 string `json:"mode,omitempty"` // "implicit" (default) or "explicit"
	TTL                  string `json:"ttl,omitempty"`  // minimum cache lifetime; currently "30m"
	ComparisonResponseID string `json:"comparison_response_id,omitempty"`
}

// PromptCacheDiagnostics explains cache reuse relative to a comparison response.
// Token counts are diagnostic estimates, not billing usage; nil means unavailable.
type PromptCacheDiagnostics struct {
	Type                     string `json:"type"` // "cache_hit", "cache_miss", "comparison_response_not_found", or "unavailable"
	Reason                   string `json:"reason,omitempty"`
	ComparisonReusableTokens *int   `json:"comparison_reusable_tokens,omitempty"`
	CacheMissedTokens        *int   `json:"cache_missed_tokens,omitempty"`
}

// MultiAgentConfig enables hosted collaboration within a Responses request.
type MultiAgentConfig struct {
	Enabled                bool `json:"enabled"`
	MaxConcurrentSubagents int  `json:"max_concurrent_subagents,omitempty"`
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
	case "image_generation":
		return json.RawMessage(`{"type": "image_generation"}`)
	default:
		return json.RawMessage(`"auto"`)
	}
}

// ForceFunction is a convenience function to force the use of a specific function tool.
// This is a backward-compatible wrapper around ForceToolChoice.
func ForceFunction(name string) json.RawMessage {
	return ForceToolChoice("function", name)
}

// ContextConfig represents configuration options for context management.
type ContextConfig struct {
	// The context management entry type. Currently only "compaction" is supported.
	Type string `json:"type"`

	// Token threshold at which compaction should be triggered for this entry.
	// Minimum 1000.
	CompactThreshold int `json:"compact_threshold,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// Fills the "type" field with "compaction" if empty.
func (c ContextConfig) MarshalJSON() ([]byte, error) {
	if c.Type == "" {
		c.Type = "compaction"
	}
	type alias ContextConfig
	return openai.Marshal(alias(c))
}
