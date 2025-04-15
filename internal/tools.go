// Package openai / internal / tools.go provides internal types for handling tool calls.
package openai

import "encoding/json"

// ToolCall represents a tool call in API response.
type ToolCallData struct {
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"` // only "function" now
	Function *FunctionCallData `json:"function,omitempty"`
}

// FunctionCall represents a function call in API response.
// Contains name of the function to be called and its arguments
// as a JSON object encoded into a string.
// Must be verified as AI may provide invalid JSON or incorrect arguments.
type FunctionCallData struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // has JSON object, but as a string
}

// UnmarshalArguments decodes JSON-encoded arguments into target.
func (data FunctionCallData) UnmarshalArguments(target any) error {
	return json.Unmarshal([]byte(data.Arguments), target)
}
