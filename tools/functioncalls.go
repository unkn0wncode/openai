// Package tools / functioncalls.go handles function calls, or tools, in chat API.
// Allows external packages to register functions that AI can request to be executed.
package tools

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Registry holds user-defined tools that AI can request to use.
type Registry struct {
	sync.RWMutex
	FunctionCalls map[string]FunctionCall
	Tools         map[string]Tool
}

var (
	// ErrDoNotRespond is to be returned by an AI function when it has completed its work
	// and wishes to prevent the AI from any further actions.
	ErrDoNotRespond = errors.New("function call indicated that further communication with AI is not needed")

	// TextDoNotRespond is a string that can be returned by a function to AI to indicate
	// that further communication is not needed for this function.
	TextDoNotRespond = "Function was executed successfully and requested to end this line of interaction. Do not respond further in this regard."
)

// EmptyParamsSchema is a minimal schema for function calls
// with no parameters that API would accept.
var EmptyParamsSchema = []byte(`{"type":"object","properties":{}}`)

// FunctionCall reperesents a function that AI can request to be executed.
// All fields except F and CallLimit are required.
// If F is not provided, calls to this function will be returned instead of being executed.
// Name must be unique.
// Description will be used by AI to understand what the function does.
// ParamsSchema must countain a valid JSON schema object for the params that F will accept.
// F can return any string but for any complex data an encoded JSON object is preferred.
// CallLimit is the maximum number of times the function can be used at once
// before non-function response is forced (0 is unlimited).
type FunctionCall struct {
	// required

	Name         string          `json:"name"` // ^[a-zA-Z0-9_-]{1,64}$
	Description  string          `json:"description"`
	ParamsSchema json.RawMessage `json:"parameters"`

	// optional

	// forces AI to follow JSON schema strictly, false by default
	// additional limitations for schema apply if set to true:
	// https://platform.openai.com/docs/guides/structured-outputs/supported-schemas
	Strict bool `json:"strict,omitempty"`

	// the function to be executed, if nil then the function call will be returned
	// instead of being executed
	F func(params json.RawMessage) (string, error) `json:"-"`

	// the function will be used no more than this number of times at once
	// and then non-function response is forced
	CallLimit int `json:"-"` // default 0, unlimited
}

// CreateFunction creates a function that can be added to AI request to be run as needed.
// To allow AI to call a function in a particular request, add the function name
// to the request's "functions" field.
func (r *Registry) CreateFunction(fc FunctionCall) error {
	r.Lock()
	defer r.Unlock()

	if fc.Name == "" || fc.ParamsSchema == nil || fc.Description == "" {
		return fmt.Errorf("function '%s' is missing required field(s)", fc.Name)
	}

	if _, ok := r.FunctionCalls[fc.Name]; ok {
		return fmt.Errorf("function '%s' is already registered, names must be unique", fc.Name)
	}

	r.FunctionCalls[fc.Name] = fc
	return nil
}

// DeleteFunction deletes a function from the registry.
func (r *Registry) DeleteFunction(name string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.FunctionCalls[name]; !ok {
		return fmt.Errorf("function '%s' is not found in registry", name)
	}

	delete(r.FunctionCalls, name)
	return nil
}

// CountFunctions returns the number of registered functions.
func (r *Registry) CountFunctions() int {
	r.RLock()
	defer r.RUnlock()

	return len(r.FunctionCalls)
}

// GetFunction returns a registered function by its name.
// Returns false if the function is not registered.
func (r *Registry) GetFunction(name string) (FunctionCall, bool) {
	r.RLock()
	defer r.RUnlock()

	fc, ok := r.FunctionCalls[name]
	return fc, ok
}

// ToolChoiceOption represents a choice of tool to be forced for use by AI.
type ToolChoiceOption string

// MarshalJSON implements json.Marshaler.
func (tco ToolChoiceOption) MarshalJSON() ([]byte, error) {
	switch tco {
	case "none", "auto":
		return json.Marshal(string(tco))
	default:
		return json.Marshal(struct {
			Type     string          `json:"type"`
			Function json.RawMessage `json:"function"`
		}{
			Type:     "function",
			Function: json.RawMessage(fmt.Sprintf(`{"name": "%s"}`, tco)),
		})
	}
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

// RegisterTool registers a tool that can be used by the model.
// To allow the model to use a tool in a particular request,
// add the tool to the request's "tools" field.
func (r *Registry) RegisterTool(tool Tool) error {
	// Validate the tool based on its type
	switch tool.Type {
	case "function":
		// Function tools require name, description, and parameters
		if tool.Function.Name == "" || tool.Function.ParamsSchema == nil || tool.Function.Description == "" {
			return fmt.Errorf("tool function '%s' is missing required field(s)", tool.Name)
		}

		// Set the Name field to match the Function.Name if not already set
		if tool.Name == "" {
			tool.Name = tool.Function.Name
		}

		fc := FunctionCall{
			Name:         tool.Name,
			Description:  tool.Description,
			ParamsSchema: tool.Parameters,
			Strict:       tool.Strict,
			F:            tool.Function.F,
			CallLimit:    tool.Function.CallLimit,
		}

		r.CreateFunction(fc)
		return nil

	case "file_search":
		// File search tools require vector_store_ids
		if len(tool.VectorStoreIDs) == 0 {
			return fmt.Errorf("file_search tool '%s' requires vector_store_ids", tool.Name)
		}

	case "web_search_preview":
		// Web search preview doesn't require additional fields

	case "computer_use_preview":
		// Computer use preview can have display dimensions and environment
		// No specific validation required

	default:
		return fmt.Errorf("unsupported tool type: %s", tool.Type)
	}

	// Check for duplicate tool names
	if _, ok := r.Tools[tool.Name]; ok {
		return fmt.Errorf("tool '%s' is already registered, names must be unique", tool.Name)
	}

	r.Lock()
	defer r.Unlock()

	r.Tools[tool.Name] = tool

	return nil
}

// DeleteTool deletes a tool from the registry.
func (r *Registry) DeleteTool(name string) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Tools[name]; !ok {
		return fmt.Errorf("tool '%s' is not found in registry", name)
	}

	delete(r.Tools, name)
	return nil
}

// CountTools returns the number of registered tools.
func (r *Registry) CountTools() int {
	r.RLock()
	defer r.RUnlock()

	return len(r.Tools)
}

// GetTool returns a registered tool by its name.
func (r *Registry) GetTool(name string) (Tool, bool) {
	r.RLock()
	defer r.RUnlock()

	tool, ok := r.Tools[name]
	return tool, ok
}
