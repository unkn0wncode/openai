// Package output provides types that can be parsed as output when receiving messages.
//
// Some of these types can also be sent to the model as inputs, but they are intentionally
// defined here to enable parsing through the Any type.
package output

import (
	"encoding/json"
	"fmt"

	openai "github.com/unkn0wncode/openai/internal"
)

// Any is a partial representation of a content object with only the "type" field unmarshaled.
// It can be used to find a correct type and further unmarshal the raw content.
type Any struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (a *Any) UnmarshalJSON(data []byte) error {
	// Extract only the "type" field, then save raw JSON for later.
	var tmp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Type = tmp.Type
	a.raw = data
	return nil
}

// UnmarshalToTarget unmarshals the content into a given target.
func (a *Any) UnmarshalToTarget(target any) error {
	return json.Unmarshal(a.raw, target)
}

// Unmarshal unmarshals the full content into a type specified in the "type" field.
func (a *Any) Unmarshal() (any, error) {
	switch a.Type {
	case "text":
		return unmarshalToType[Text](a)
	case "output_text":
		return unmarshalToType[OutputText](a)
	case "image_url":
		return unmarshalToType[ImageURL](a)
	case "image_file":
		return unmarshalToType[ImageFile](a)
	case "refusal":
		return unmarshalToType[Refusal](a)
	case "message":
		return unmarshalToType[Message](a)
	case "file_search_call":
		return unmarshalToType[FileSearchCall](a)
	case "computer_call":
		return unmarshalToType[ComputerCall](a)
	case "computer_call_output":
		return unmarshalToType[ComputerCallOutput](a)
	case "web_search_call":
		return unmarshalToType[WebSearchCall](a)
	case "function_call":
		return unmarshalToType[FunctionCall](a)
	case "function_call_output":
		return unmarshalToType[FunctionCallOutput](a)
	case "custom_tool_call":
		return unmarshalToType[CustomToolCall](a)
	case "custom_tool_call_output":
		return unmarshalToType[CustomToolCallOutput](a)
	case "reasoning":
		return unmarshalToType[Reasoning](a)
	case "apply_patch_call":
		return unmarshalToType[ApplyPatchCall](a)
	case "apply_patch_call_output":
		return unmarshalToType[ApplyPatchCallOutput](a)
	case "mcp_list_tools":
		return unmarshalToType[MCPListTools](a)
	case "mcp_approval_request":
		return unmarshalToType[MCPApprovalRequest](a)
	case "mcp_approval_response":
		return unmarshalToType[MCPApprovalResponse](a)
	case "mcp_call":
		return unmarshalToType[MCPCall](a)
	case "local_shell_call":
		return unmarshalToType[LocalShellCall](a)
	case "local_shell_call_output":
		return unmarshalToType[LocalShellCallOutput](a)
	case "shell_call":
		return unmarshalToType[ShellCall](a)
	case "shell_call_output":
		return unmarshalToType[ShellCallOutput](a)
	case "code_interpreter_call":
		return unmarshalToType[CodeInterpreterCall](a)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", a.Type)
	}
}

// unmarshalToType is a generic function that unmarshals Any into a given type.
func unmarshalToType[T any](a interface{ UnmarshalToTarget(any) error }) (T, error) {
	var t T
	if err := a.UnmarshalToTarget(&t); err != nil {
		return t, err
	}
	return t, nil
}

// MarshalJSON implements the json.Marshaler interface.
// It just returns the saved raw content.
func (a Any) MarshalJSON() ([]byte, error) {
	return a.raw, nil
}

// String implements the fmt.Stringer interface.
// Returns the raw content as a string.
func (a Any) String() string {
	return string(a.raw)
}

// Text is a string content.
type Text struct {
	Type string `json:"type"` // "text"
	Text struct {
		Value       string          `json:"value"`
		Annotations []AnyAnnotation `json:"annotations,omitempty"`
	} `json:"text"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "text", discarding any prior value.
func (t Text) MarshalJSON() ([]byte, error) {
	t.Type = "text"
	type alias Text
	return openai.Marshal(alias(t))
}

// String implements the fmt.Stringer interface.
// Returns the text content.
func (t Text) String() string {
	return t.Text.Value
}

// OutputText is a text content.
type OutputText struct {
	Type        string          `json:"type"` // "output_text"
	Text        string          `json:"text"`
	Annotations []AnyAnnotation `json:"annotations"` // required even when empty
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "output_text", discarding any prior value.
// It also ensures that the "annotations" field is not nil.
func (t OutputText) MarshalJSON() ([]byte, error) {
	t.Type = "output_text"
	if t.Annotations == nil {
		t.Annotations = []AnyAnnotation{}
	}
	type alias OutputText
	return openai.Marshal(alias(t))
}

// String implements the fmt.Stringer interface.
// Returns the text content.
func (t OutputText) String() string {
	return t.Text
}

// ImageURL is an image referenced by a URL or as base64 encoded data.
type ImageURL struct {
	Type  string `json:"type"` // "image_url"
	Image struct {
		URL    string `json:"url"`              // required
		Detail string `json:"detail,omitempty"` // optional; "auto", "high", "low"
	} `json:"image_url"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "image_url", discarding any prior value.
func (i ImageURL) MarshalJSON() ([]byte, error) {
	i.Type = "image_url"
	type alias ImageURL
	return openai.Marshal(alias(i))
}

// String implements the fmt.Stringer interface.
// Returns the image URL content.
func (i ImageURL) String() string {
	return i.Image.URL
}

// ImageFile is an image referenced by a file ID.
type ImageFile struct {
	Type string `json:"type"` // "image_file"
	File struct {
		FileID string `json:"file_id"`          // required
		Detail string `json:"detail,omitempty"` // optional; "auto", "high", "low"
	} `json:"image_file"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "image_file", discarding any prior value.
func (i ImageFile) MarshalJSON() ([]byte, error) {
	i.Type = "image_file"
	type alias ImageFile
	return openai.Marshal(alias(i))
}

// String implements the fmt.Stringer interface.
// Returns the image file content.
func (i ImageFile) String() string {
	return i.File.FileID
}

// AnyAnnotation is an annotation for a text value with only the "type" field unmarshaled.
type AnyAnnotation struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (a *AnyAnnotation) UnmarshalJSON(data []byte) error {
	// Extract only the "type" field, then save raw JSON for later.
	var tmp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Type = tmp.Type
	a.raw = data
	return nil
}

// UnmarshalToTarget unmarshals the annotation into a given target.
func (a *AnyAnnotation) UnmarshalToTarget(target any) error {
	return json.Unmarshal(a.raw, target)
}

// Unmarshal unmarshals the full annotation content into a type specified in the "type" field.
func (a *AnyAnnotation) Unmarshal() (any, error) {
	switch a.Type {
	case "file_citation":
		return unmarshalToType[AnnotationFileCitation](a)
	case "file_path":
		return unmarshalToType[AnnotationFilePath](a)
	case "url_citation":
		return unmarshalToType[AnnotationURLCitation](a)
	default:
		return nil, fmt.Errorf("unsupported annotation type: %s", a.Type)
	}
}

// AnnotationFileCitation is an annotation type that references a part of a file.
type AnnotationFileCitation struct {
	Type         string `json:"type"` // "file_citation"
	Text         string `json:"text"`
	StartIndex   int    `json:"start_index"`
	EndIndex     int    `json:"end_index"`
	FileCitation struct {
		FileID string `json:"file_id"`
	} `json:"file_citation"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "file_citation", discarding any prior value.
func (a AnnotationFileCitation) MarshalJSON() ([]byte, error) {
	a.Type = "file_citation"
	type alias AnnotationFileCitation
	return openai.Marshal(alias(a))
}

// AnnotationFilePath is an annotation type that references a part of a file path.
type AnnotationFilePath struct {
	Type       string `json:"type"` // "file_path"
	Text       string `json:"text"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	FilePath   struct {
		FileID string `json:"file_id"`
	} `json:"file_path"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "file_path", discarding any prior value.
func (a AnnotationFilePath) MarshalJSON() ([]byte, error) {
	a.Type = "file_path"
	type alias AnnotationFilePath
	return openai.Marshal(alias(a))
}

// AnnotationURLCitation is an annotation type that references a URL.
type AnnotationURLCitation struct {
	Type       string `json:"type"` // "url_citation"
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "url_citation", discarding any prior value.
func (a AnnotationURLCitation) MarshalJSON() ([]byte, error) {
	a.Type = "url_citation"
	type alias AnnotationURLCitation
	return openai.Marshal(alias(a))
}

// Refusal is a refusal to process the request.
type Refusal struct {
	Type    string `json:"type"` // "refusal"
	Refusal string `json:"refusal"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "refusal", discarding any prior value.
func (r Refusal) MarshalJSON() ([]byte, error) {
	r.Type = "refusal"
	type alias Refusal
	return openai.Marshal(alias(r))
}

// String implements the fmt.Stringer interface.
// Returns the refusal content.
func (r Refusal) String() string {
	return r.Refusal
}

// Message is a message object, indicating who sent the and its contents.
type Message struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type"`             // "message"
	Role    string `json:"role"`             // "assistant"
	Status  string `json:"status,omitempty"` // "in_progress", "completed", "incomplete"
	Content any    `json:"content"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "message", discarding any prior value.
func (m Message) MarshalJSON() ([]byte, error) {
	m.Type = "message"
	type alias Message
	return openai.Marshal(alias(m))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It tries to unmarshal the content according to possible types that it may have.
func (m *Message) UnmarshalJSON(data []byte) error {
	// First try to unmarshal everything else
	var tmpNoContent struct {
		ID     string `json:"id"`
		Type   string `json:"type"`
		Role   string `json:"role"`
		Status string `json:"status,omitempty"`
	}
	if err := json.Unmarshal(data, &tmpNoContent); err != nil {
		return fmt.Errorf("failed to unmarshal non-content part of message: %w", err)
	}

	m.ID = tmpNoContent.ID
	m.Type = tmpNoContent.Type
	m.Role = tmpNoContent.Role
	m.Status = tmpNoContent.Status

	// then try to unmarshal content as a string
	var tmpString struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &tmpString); err == nil {
		m.Content = tmpString.Content
		return nil
	}

	// if it's not a string, try as []Any
	var tmpAny struct {
		Content []Any `json:"content"`
	}
	if err := json.Unmarshal(data, &tmpAny); err != nil {
		// here we can return an error because it shouldn't be anything else
		return fmt.Errorf("failed to unmarshal content as []Any: %w", err)
	}

	if len(tmpAny.Content) == 0 {
		m.Content = []any(nil)
		return nil
	}

	contentSlice := []any{}
	for _, c := range tmpAny.Content {
		parsed, err := c.Unmarshal()
		if err != nil {
			return fmt.Errorf("failed to unmarshal content element: %w", err)
		}
		contentSlice = append(contentSlice, parsed)
	}
	m.Content = contentSlice

	return nil
}

// FileSearchToolCall describes a use of the file search tool.
type FileSearchCall struct {
	Type    string             `json:"type"` // "file_search_call"
	ID      string             `json:"id"`
	Queries []string           `json:"queries"`
	Status  string             `json:"status"` // "in_progress", "searching", "incomplete", "failed"
	Results []FileSearchResult `json:"results"`
}

// FileSearchResult describes a result of a file search.
type FileSearchResult struct {
	FileID     string            `json:"file_id"`
	FileName   string            `json:"filename"`
	Score      float64           `json:"score"` // from 0 to 1
	Text       string            `json:"text"`
	Attributes map[string]string `json:"attributes"` // 16 pairs: keys 64 chars, values 512 chars
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "file_search_call", discarding any prior value.
func (f FileSearchCall) MarshalJSON() ([]byte, error) {
	f.Type = "file_search_call"
	type alias FileSearchCall
	return openai.Marshal(alias(f))
}

// ComputerCall describes a use of the computer use tool.
type ComputerCall struct {
	Type                string        `json:"type"` // "computer_call"
	ID                  string        `json:"id"`
	CallID              string        `json:"call_id"`
	Action              any           `json:"action"`                // TODO: implement Click, DoubleClick, Drag, KeyPress, Move, Screenshot, Scroll, Type, Wait
	PendingSafetyChecks []SafetyCheck `json:"pending_safety_checks"` // required even when empty
	Status              string        `json:"status"`                // "in_progress", "completed", "incomplete"
}

// SafetyCheck describes a safety check that is pending for a computer call.
type SafetyCheck struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "computer_call", discarding any prior value.
func (c ComputerCall) MarshalJSON() ([]byte, error) {
	c.Type = "computer_call"
	if c.PendingSafetyChecks == nil {
		c.PendingSafetyChecks = []SafetyCheck{}
	}
	type alias ComputerCall
	return openai.Marshal(alias(c))
}

// ComputerCallOutput describes the output of a computer call.
type ComputerCallOutput struct {
	Type                     string               `json:"type"` // "computer_call_output"
	Output                   []ComputerScreenshot `json:"output"`
	CallID                   string               `json:"call_id"`
	ID                       string               `json:"id"`
	Status                   string               `json:"status"` // "in_progress", "completed", "incomplete"
	AcknowledgedSafetyChecks []SafetyCheck        `json:"acknowledged_safety_checks,omitempty"`
}

// ComputerScreenshot describes a screenshot of the computer in a computer use flow.
type ComputerScreenshot struct {
	Type     string `json:"type"` // "computer_screenshot"
	FileID   string `json:"file_id,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "computer_screenshot", discarding any prior value.
func (c ComputerScreenshot) MarshalJSON() ([]byte, error) {
	c.Type = "computer_screenshot"
	type alias ComputerScreenshot
	return openai.Marshal(alias(c))
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "computer_call_output", discarding any prior value.
func (c ComputerCallOutput) MarshalJSON() ([]byte, error) {
	c.Type = "computer_call_output"
	type alias ComputerCallOutput
	return openai.Marshal(alias(c))
}

// WebSearchCall describes a use of the web search tool.
type WebSearchCall struct {
	Type   string             `json:"type"` // "web_search_call"
	ID     string             `json:"id"`
	Status string             `json:"status"` // "in_progress", "completed", "incomplete"
	Action AnyWebSearchAction `json:"action"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "web_search_call", discarding any prior value.
func (w WebSearchCall) MarshalJSON() ([]byte, error) {
	w.Type = "web_search_call"
	type alias WebSearchCall
	return openai.Marshal(alias(w))
}

// AnyWebSearchAction is a union type for all possible web search actions.
type AnyWebSearchAction struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (a *AnyWebSearchAction) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Type = tmp.Type
	a.raw = data
	return nil
}

// UnmarshalToTarget unmarshals the action into a given target.
func (a *AnyWebSearchAction) UnmarshalToTarget(target any) error {
	return json.Unmarshal(a.raw, target)
}

// Unmarshal unmarshals the full action content into a type specified in the "type" field.
func (a *AnyWebSearchAction) Unmarshal() (any, error) {
	switch a.Type {
	case "search":
		return unmarshalToType[WebSearchActionSearch](a)
	case "open_page":
		return unmarshalToType[WebSearchActionOpenPage](a)
	case "find":
		return unmarshalToType[WebSearchActionFind](a)
	default:
		return nil, fmt.Errorf("unsupported web search action type: %s", a.Type)
	}
}

// WebSearchActionSearch describes a search action.
type WebSearchActionSearch struct {
	// required

	Type  string `json:"type"` // "search"
	Query string `json:"query"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "search", discarding any prior value.
func (w WebSearchActionSearch) MarshalJSON() ([]byte, error) {
	w.Type = "search"
	type alias WebSearchActionSearch
	return openai.Marshal(alias(w))
}

// WebSearchActionOpenPage describes an open page action.
type WebSearchActionOpenPage struct {
	// required

	Type string `json:"type"` // "open_page"
	URL  string `json:"url"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "open_page", discarding any prior value.
func (w WebSearchActionOpenPage) MarshalJSON() ([]byte, error) {
	w.Type = "open_page"
	type alias WebSearchActionOpenPage
	return openai.Marshal(alias(w))
}

// WebSearchActionFind describes a find action.
type WebSearchActionFind struct {
	// required

	Type    string `json:"type"` // "find"
	URL     string `json:"url"`
	Pattern string `json:"pattern"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "find", discarding any prior value.
func (w WebSearchActionFind) MarshalJSON() ([]byte, error) {
	w.Type = "find"
	type alias WebSearchActionFind
	return openai.Marshal(alias(w))
}

// FunctionCall describes a use of a function call tool.
type FunctionCall struct {
	// required

	Type      string `json:"type"` // "function_call"
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // has JSON object, but as a string

	// optional

	CallID string `json:"call_id"`
	Status string `json:"status"` // "in_progress", "completed", "incomplete"
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "function_call", discarding any prior value.
func (f FunctionCall) MarshalJSON() ([]byte, error) {
	f.Type = "function_call"
	type alias FunctionCall
	return openai.Marshal(alias(f))
}

// UnmarshalArguments decodes JSON-encoded arguments into target.
func (f FunctionCall) UnmarshalArguments(target any) error {
	return json.Unmarshal([]byte(f.Arguments), target)
}

// FunctionCallOutput describes the output of a function call.
type FunctionCallOutput struct {
	// required

	Type   string `json:"type"` // "function_call_output"
	CallID string `json:"call_id"`
	Output string `json:"output"` // expected to be JSON encoded as a string

	// optional

	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"` // "in_progress", "completed", "incomplete"
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "function_call_output", discarding any prior value.
func (f FunctionCallOutput) MarshalJSON() ([]byte, error) {
	f.Type = "function_call_output"
	type alias FunctionCallOutput
	return openai.Marshal(alias(f))
}

// CustomToolCall describes a use of a custom tool call.
type CustomToolCall struct {
	// required

	Type   string `json:"type"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Input  string `json:"input"`
	Status string `json:"status"`

	// optional

	CallID string `json:"call_id,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "custom_tool_call", discarding any prior value.
func (c CustomToolCall) MarshalJSON() ([]byte, error) {
	c.Type = "custom_tool_call"
	type alias CustomToolCall
	return openai.Marshal(alias(c))
}

// CustomToolCallOutput describes the output of a custom tool call.
type CustomToolCallOutput struct {
	// required

	Type   string `json:"type"`
	CallID string `json:"call_id"`
	Output string `json:"output"`

	// optional

	ID     string `json:"id,omitempty"`
	Status string `json:"status,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "custom_tool_call_output", discarding any prior value.
func (c CustomToolCallOutput) MarshalJSON() ([]byte, error) {
	c.Type = "custom_tool_call_output"
	type alias CustomToolCallOutput
	return openai.Marshal(alias(c))
}

// ApplyPatchCall describes a use of the apply_patch tool.
type ApplyPatchCall struct {
	// required
	Type      string              `json:"type"` // "apply_patch_call"
	ID        string              `json:"id,omitempty"`
	CallID    string              `json:"call_id"`
	Status    string              `json:"status"` // "in_progress" or "completed"
	Operation ApplyPatchOperation `json:"operation"`
}

// ApplyPatchOperation describes a file operation requested by the apply_patch tool.
type ApplyPatchOperation struct {
	// required
	Type string `json:"type"` // "create_file", "update_file", or "delete_file"
	Path string `json:"path"`
	// optional
	Diff string `json:"diff,omitempty"` // V4A diff representing file contents or changes
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "apply_patch_call", discarding any prior value.
func (a ApplyPatchCall) MarshalJSON() ([]byte, error) {
	a.Type = "apply_patch_call"
	type alias ApplyPatchCall
	return openai.Marshal(alias(a))
}

// ApplyPatchCallOutput describes the output of an apply_patch tool call.
type ApplyPatchCallOutput struct {
	// required
	Type   string `json:"type"` // "apply_patch_call_output"
	CallID string `json:"call_id"`
	Status string `json:"status"` // "completed" or "failed"
	// optional
	ID     string `json:"id,omitempty"`
	Output string `json:"output,omitempty"` // human-readable status or error message
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "apply_patch_call_output", discarding any prior value.
func (a ApplyPatchCallOutput) MarshalJSON() ([]byte, error) {
	a.Type = "apply_patch_call_output"
	type alias ApplyPatchCallOutput
	return openai.Marshal(alias(a))
}

// Reasoning describes model's internal thinking process.
type Reasoning struct {
	Type    string             `json:"type"` // "reasoning"
	ID      string             `json:"id"`
	Status  string             `json:"status"`  // "in_progress", "completed", "incomplete"
	Summary []ReasoningSummary `json:"summary"` // required even when empty
}

// ReasoningSummary describes a summary of the reasoning.
type ReasoningSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "summary_text", discarding any prior value.
func (r ReasoningSummary) MarshalJSON() ([]byte, error) {
	r.Type = "summary_text"
	type alias ReasoningSummary
	return openai.Marshal(alias(r))
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "reasoning", discarding any prior value.
func (r Reasoning) MarshalJSON() ([]byte, error) {
	r.Type = "reasoning"
	if r.Summary == nil {
		r.Summary = []ReasoningSummary{}
	}
	type alias Reasoning
	return openai.Marshal(alias(r))
}

// MCPListTools describes tools available on an MCP server.
type MCPListTools struct {
	// required

	Type        string    `json:"type"` // "mcp_list_tools"
	ID          string    `json:"id"`
	ServerLabel string    `json:"server_label"`
	Tools       []MCPTool `json:"tools"`

	// optional

	Error string `json:"error,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "mcp_list_tools", discarding any prior value.
func (m MCPListTools) MarshalJSON() ([]byte, error) {
	m.Type = "mcp_list_tools"
	type alias MCPListTools
	return openai.Marshal(alias(m))
}

// MCPTool describes a tool available on an MCP server.
type MCPTool struct {
	// required

	Name        string          `json:"name"`
	InputSchema json.RawMessage `json:"input_schema"`

	// optional

	Description string          `json:"description,omitempty"`
	Annotations []AnyAnnotation `json:"annotations,omitempty"`
}

// MCPApprovalRequest describes a request to approve an MCP tool call.
type MCPApprovalRequest struct {
	Type        string          `json:"type"` // "mcp_approval_request"
	ID          string          `json:"id"`
	ServerLabel string          `json:"server_label"`
	Name        string          `json:"name"`
	Arguments   json.RawMessage `json:"arguments"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "mcp_approval_request", discarding any prior value.
func (m MCPApprovalRequest) MarshalJSON() ([]byte, error) {
	m.Type = "mcp_approval_request"
	type alias MCPApprovalRequest
	return openai.Marshal(alias(m))
}

// Respond generates a response to an MCP approval request.
// Reason is optional.
func (m MCPApprovalRequest) Respond(approve bool, reason string) MCPApprovalResponse {
	return MCPApprovalResponse{
		Type:              "mcp_approval_response",
		ApprovalRequestID: m.ID,
		Approve:           approve,
		Reason:            reason,
	}
}

// MCPApprovalResponse describes a response to an MCP approval request.
type MCPApprovalResponse struct {
	// required

	Type              string `json:"type"` // "mcp_approval_response"
	ApprovalRequestID string `json:"approval_request_id"`
	Approve           bool   `json:"approve"`

	// optional

	ID     string `json:"id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "mcp_approval_response", discarding any prior value.
func (m MCPApprovalResponse) MarshalJSON() ([]byte, error) {
	m.Type = "mcp_approval_response"
	type alias MCPApprovalResponse
	return openai.Marshal(alias(m))
}

// MCPCall describes a call to an MCP tool.
type MCPCall struct {
	// required

	Type        string          `json:"type"` // "mcp_call"
	ID          string          `json:"id"`
	ServerLabel string          `json:"server_label"`
	Name        string          `json:"name"`
	Arguments   json.RawMessage `json:"arguments"`

	// optional

	Error  string `json:"error,omitempty"`
	Output string `json:"output,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "mcp_call", discarding any prior value.
func (m MCPCall) MarshalJSON() ([]byte, error) {
	m.Type = "mcp_call"
	type alias MCPCall
	return openai.Marshal(alias(m))
}

// LocalShellCall describes a call to a local shell tool.
type LocalShellCall struct {
	Type   string           `json:"type"` // "local_shell_call"
	ID     string           `json:"id"`
	CallID string           `json:"call_id"`
	Action LocalShellAction `json:"action"`
	Status string           `json:"status"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "local_shell_call", discarding any prior value.
func (l LocalShellCall) MarshalJSON() ([]byte, error) {
	l.Type = "local_shell_call"
	type alias LocalShellCall
	return openai.Marshal(alias(l))
}

// LocalShellAction describes an action to be taken by a local shell tool.
type LocalShellAction struct {
	// required

	Type    string            `json:"type"` // "exec"
	Command []string          `json:"command"`
	Env     map[string]string `json:"env"`

	// optional

	TimeoutMilliseconds int    `json:"timeout_milliseconds,omitempty"`
	User                string `json:"user,omitempty"`
	WorkingDirectory    string `json:"working_directory,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "exec", discarding any prior value.
func (l LocalShellAction) MarshalJSON() ([]byte, error) {
	l.Type = "exec"
	type alias LocalShellAction
	return openai.Marshal(alias(l))
}

// LocalShellCallOutput describes the output of a local shell call.
type LocalShellCallOutput struct {
	// required

	Type   string `json:"type"` // "local_shell_call_output"
	ID     string `json:"id"`
	Output string `json:"output"`

	// optional

	Status string `json:"status,omitempty"` // "in_progress", "completed", "incomplete"
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "local_shell_call_output", discarding any prior value.
func (l LocalShellCallOutput) MarshalJSON() ([]byte, error) {
	l.Type = "local_shell_call_output"
	type alias LocalShellCallOutput
	return openai.Marshal(alias(l))
}

// ShellCall describes a call to the GPT-5.1+ shell tool.
type ShellCall struct {
	Type   string      `json:"type"` // "shell_call"
	ID     string      `json:"id,omitempty"`
	CallID string      `json:"call_id"`
	Action ShellAction `json:"action"`
	Status string      `json:"status"` // "in_progress", "completed"
}

// ShellAction describes the action requested by the shell tool.
type ShellAction struct {
	Commands        []string `json:"commands"`                    // commands to execute
	TimeoutMS       int      `json:"timeout_ms,omitempty"`        // optional per-call timeout
	MaxOutputLength int      `json:"max_output_length,omitempty"` // requested max bytes for output
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "shell_call".
func (s ShellCall) MarshalJSON() ([]byte, error) {
	s.Type = "shell_call"
	type alias ShellCall
	return openai.Marshal(alias(s))
}

// ShellCallOutput describes the output of a shell tool call.
type ShellCallOutput struct {
	Type            string `json:"type"` // "shell_call_output"
	ID              string `json:"id,omitempty"`
	CallID          string `json:"call_id"`
	MaxOutputLength int    `json:"max_output_length,omitempty"`
	// Output contains one or more results from executing the requested shell commands.
	Output []ShellCommandResult `json:"output"`
}

// ShellCommandResult describes the result of executing a single shell command.
type ShellCommandResult struct {
	Stdout  string           `json:"stdout"`
	Stderr  string           `json:"stderr"`
	Outcome ShellCallOutcome `json:"outcome"`
}

// ShellCallOutcome describes the outcome of a shell command.
type ShellCallOutcome struct {
	Type     string `json:"type"`                // "exit" or "timeout"
	ExitCode *int   `json:"exit_code,omitempty"` // present only when type == "exit"
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "shell_call_output".
func (s ShellCallOutput) MarshalJSON() ([]byte, error) {
	s.Type = "shell_call_output"
	type alias ShellCallOutput
	return openai.Marshal(alias(s))
}

// CodeInterpreterCall describes a call to a code interpreter tool.
type CodeInterpreterCall struct {
	// required

	Type    string                     `json:"type"` // "code_interpreter_call"
	ID      string                     `json:"id"`
	Code    string                     `json:"code"`
	Status  string                     `json:"status"`
	Results []CodeInterpreterResultAny `json:"results"`

	// optional

	ContainerID string `json:"container_id,omitempty"`
}

// CodeInterpreterResultText describes a log of a call interpreter tool.
type CodeInterpreterResultText struct {
	Type string `json:"type"` // "logs"
	Logs string `json:"logs"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "logs", discarding any prior value.
func (c CodeInterpreterResultText) MarshalJSON() ([]byte, error) {
	c.Type = "logs"
	type alias CodeInterpreterResultText
	return openai.Marshal(alias(c))
}

// CodeInterpreterResultFile describes files made by a call interpreter tool.
type CodeInterpreterResultFile struct {
	Type  string                `json:"type"` // "files"
	Files []CodeInterpreterFile `json:"files"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "files", discarding any prior value.
func (c CodeInterpreterResultFile) MarshalJSON() ([]byte, error) {
	c.Type = "files"
	type alias CodeInterpreterResultFile
	return openai.Marshal(alias(c))
}

// CodeInterpreterFile describes a file made by a call interpreter tool.
type CodeInterpreterFile struct {
	FileID   string `json:"file_id"`
	MimeType string `json:"mime_type"`
}

// CodeInterpreterResultAny is a partial representation of a code interpreter result with only the "type" field unmarshaled.
// It can be used to find a correct type and further unmarshal the raw content.
type CodeInterpreterResultAny struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (c *CodeInterpreterResultAny) UnmarshalJSON(data []byte) error {
	// Extract only the "type" field, then save raw JSON for later.
	var tmp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	c.Type = tmp.Type
	c.raw = data
	return nil
}

// UnmarshalToTarget unmarshals the result into a given target.
func (c *CodeInterpreterResultAny) UnmarshalToTarget(target any) error {
	return json.Unmarshal(c.raw, target)
}

// Unmarshal unmarshals the full result content into a type specified in the "type" field.
func (c *CodeInterpreterResultAny) Unmarshal() (any, error) {
	switch c.Type {
	case "logs":
		return unmarshalToType[CodeInterpreterResultText](c)
	case "files":
		return unmarshalToType[CodeInterpreterResultFile](c)
	default:
		return nil, fmt.Errorf("unsupported code interpreter result type: %s", c.Type)
	}
}

// MarshalJSON implements the json.Marshaler interface.
// It just returns the saved raw content.
func (c CodeInterpreterResultAny) MarshalJSON() ([]byte, error) {
	return c.raw, nil
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "code_interpreter_call", discarding any prior value.
func (c CodeInterpreterCall) MarshalJSON() ([]byte, error) {
	c.Type = "code_interpreter_call"
	type alias CodeInterpreterCall
	return openai.Marshal(alias(c))
}
