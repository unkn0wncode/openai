package assistants

import (
	"context"
	"time"

	"github.com/unkn0wncode/openai/content/input"
	"github.com/unkn0wncode/openai/content/output"
)

// CreateParams holds the fields needed to create a new assistant.
type CreateParams struct {
	// required
	Model string `json:"model"`

	// optional
	Name            string     `json:"name,omitempty"`
	Instructions    string     `json:"instructions,omitempty"`
	Description     string     `json:"description,omitempty"`
	Metadata        Metadata   `json:"metadata,omitempty"`
	ReasoningEffort string     `json:"reasoning_effort,omitempty"` // "low", "medium", "high"
	ResponseFormat  string     `json:"response_format,omitempty"`  // "auto" or object with "type": "text", "json_object", "json_schema"
	Temperature     float64    `json:"temperature,omitempty"`
	TopP            float64    `json:"top_p,omitempty"`
	ToolResources   any        `json:"tool_resources,omitempty"` // TODO: not yet implemented
	Tools           []ToolSpec `json:"tools,omitempty"`
}

// Service defines methods to operate on assistants.
type Service interface {
	// CreateAssistant creates a new assistant.
	CreateAssistant(params CreateParams) (Assistant, error)

	// LoadAssistant retrieves an assistant by ID.
	LoadAssistant(id string) (Assistant, error)

	// ListAssistant returns all assistants.
	ListAssistant() ([]Assistant, error)

	// DeleteAssistant deletes an assistant by ID.
	DeleteAssistant(id string) error

	// AssistantsRunRefreshInterval returns the interval between status polls in Await.
	AssistantsRunRefreshInterval() time.Duration

	// SetAssistantsRunRefreshInterval sets the interval between status polls in Await.
	SetAssistantsRunRefreshInterval(interval time.Duration)
}

// Content is an interface listing all types that can be used as content in the Assistants API.
// These types may appear in `Message.Content` and `InputMessage.Content` fields.
type Content interface {
	string |
	input.Text |
	input.ImageFile |
	input.ImageURL |
	output.Text |
	output.ImageFile |
	output.ImageURL |
	output.Refusal
}

// Assistant is a live handle on a server-side assistant.
type Assistant interface {
	// ID returns the unique identifier of the assistant.
	ID() string

	// Model returns the assistant's model.
	Model() string

	// Name returns the assistant's name.
	Name() string
	// SetName updates the assistant's name on the server.
	SetName(name string) error

	// Description returns the assistant's description.
	Description() string
	// SetDescription updates the assistant's description on the server.
	SetDescription(desc string) error

	// Instructions returns the system instructions for the assistant.
	Instructions() string
	// SetInstructions updates the assistant's instructions on the server.
	SetInstructions(ins string) error

	// Tools lists the tools enabled for this assistant.
	Tools() []ToolSpec
	// AddTool enables a new tool on the assistant.
	AddTool(tool ToolSpec) error
	// RemoveTool disables a tool by name.
	RemoveTool(name string) error

	// FileIDs lists files attached to this assistant.
	FileIDs() []string
	// AttachFile attaches a file to the assistant.
	AttachFile(fileID string) error
	// DetachFile removes a file from the assistant.
	DetachFile(fileID string) error

	// NewThread creates a new conversation thread under this assistant.
	NewThread(meta Metadata, messages ...InputMessage) (Thread, error)
	// LoadThread fetches an existing thread by ID.
	LoadThread(id string) (Thread, error)
}

// Thread represents a conversation thread under an assistant.
type Thread interface {
	// ID returns the unique identifier of the thread.
	ID() string

	// AddMessage adds a user message to the thread.
	AddMessage(msg InputMessage) (Message, error)
	// Messages lists messages in the thread with pagination.
	Messages(limit int, after string) ([]Message, bool, error)

	// Run creates a new run on this thread.
	Run(opts *RunOptions) (Run, error)
	// RunAndFetch runs the assistant and fetches the next assistant message.
	RunAndFetch(ctx context.Context, opts *RunOptions, messages ...InputMessage) (Run, *Message, error)
}

// Run represents a single execution of an assistant on a thread.
type Run interface {
	// ID returns the unique identifier of the run.
	ID() string

	// SubmitToolOutputs supplies results for any expected tool calls.
	SubmitToolOutputs(outputs ...ToolOutput) error
	// Await waits until the run completes or context is done.
	Await(ctx context.Context) error

	// IsPending reports if the run is still in a non-terminal state.
	IsPending() bool
	// IsExpectingToolOutputs reports if the run is paused awaiting tool outputs.
	IsExpectingToolOutputs() bool
}

// ToolSpec describes a tool that an assistant can use.
type ToolSpec struct {
	Name         string // unique tool name
	Description  string // human-readable description
	ParamsSchema string // JSON schema for the tool's parameters
}

// Metadata is a map of key/value pairs for annotations.
type Metadata map[string]string

// InputMessage is a user message to send to a thread.
type InputMessage struct {
	// required
	Content any    `json:"content"` // string or array of inputs
	Role    string `json:"role"`    // only "user" or "assistant"

	// optional
	Attachments []any    `json:"attachments,omitempty"` // TODO: not yet implemented
	Metadata    Metadata `json:"metadata,omitempty"`    // optional metadata map
}

// RunOptions configures how a run is created.
type RunOptions struct {
	Model                  string // override model ID
	Instructions           string // override system instructions
	AdditionalInstructions string // extra instructions appended
	Tools                  []ToolSpec
	Metadata               Metadata
}

// Message is a single message in a thread or run.
type Message struct {
	Role    string // "user" or "assistant" or "tool"
	Content any    // message content: string or array of inputs
}

// ToolOutput is the output of a tool call for a run.
type ToolOutput struct {
	ToolCallID string // ID of the expected tool call
	Output     string // execution result
}
