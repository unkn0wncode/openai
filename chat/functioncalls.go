// Package chat / functioncalls.go handles function calls, or tools, in chat API.
// Allows external packages to register functions that AI can request to be executed.
package chat

import (
	"encoding/json"
	"errors"
	"fmt"
)

// funcCalls stores all registered functions.
var funcCalls = map[string]FunctionCall{}

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
func CreateFunction(fc FunctionCall) {
	if fc.Name == "" || fc.ParamsSchema == nil || fc.Description == "" {
		panic("function '" + fc.Name + "' is missing required field(s)")
	}

	if _, ok := funcCalls[fc.Name]; ok {
		panic("function '" + fc.Name + "' is already registered, names must be unique")
	}

	funcCalls[fc.Name] = fc
}

// CountFunctions returns the number of registered functions.
func CountFunctions() int {
	return len(funcCalls)
}

// GetFunction returns a registered function by its name.
// Returns false if the function is not registered.
func GetFunction(name string) (FunctionCall, bool) {
	fc, ok := funcCalls[name]
	return fc, ok
}

// FunctionCall represents a function call in API response.
// Contains name of the function to be called and its arguments
// as a JSON object encoded into a string.
// Must be verified as AI may provide invalid JSON or incorrect arguments.
type FunctionCallData struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // has JSON object, but as a string
}

type ToolCallData struct {
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"` // only "function" now
	Function FunctionCallData `json:"function,omitempty"`
}

// UnmarshalArguments decodes JSON-encoded arguments into target.
func (data FunctionCallData) UnmarshalArguments(target any) error {
	return json.Unmarshal([]byte(data.Arguments), target)
}

// ToolChoiceOption represents a choice of tool to be forced for used by AI.
type ToolChoiceOption string

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
