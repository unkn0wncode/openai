// Package openai / internal / stopreasons.go contains stop reasons that can be returned by OpenAI.
package openai

// Stop reason constants.
const (
	FinishReasonStop         = "stop"
	FinishReasonLength       = "length"
	FinishReasonFilter       = "content_filter"
	FinishReasonFunctionCall = "function_call"
	FinishReasonToolCalls    = "tool_calls"
	FinishReasonNull         = "null"
)
