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
	case "reasoning":
		return unmarshalToType[Reasoning](a)
	case "mcp_list_tools":
		return unmarshalToType[MCPListTools](a)
	case "mcp_approval_request":
		return unmarshalToType[MCPApprovalRequest](a)
	case "mcp_approval_response":
		return unmarshalToType[MCPApprovalResponse](a)
	case "mcp_call":
		return unmarshalToType[MCPCall](a)
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
	Type   string `json:"type"` // "web_search_call"
	ID     string `json:"id"`
	Status string `json:"status"` // "in_progress", "completed", "incomplete"
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "web_search_call", discarding any prior value.
func (w WebSearchCall) MarshalJSON() ([]byte, error) {
	w.Type = "web_search_call"
	type alias WebSearchCall
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

// TODO: Add Code interpreter tool call type
