// Package streaming provides types for streaming responses from the OpenAI Responses API.
package streaming

import (
	"encoding/json"
	"fmt"
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

// Unmarshal unmarshals event bytes into a type specified in the "type" field.
func Unmarshal(data []byte) (any, error) {
	var a Any
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return a.Unmarshal()
}

// Unmarshal unmarshals the full content into a type specified in the "type" field.
func (a *Any) Unmarshal() (any, error) {
	switch a.Type {
	case "response.created":
		return unmarshalToType[ResponseCreated](a)
	case "response.in_progress":
		return unmarshalToType[ResponseInProgress](a)
	case "response.completed":
		return unmarshalToType[ResponseCompleted](a)
	case "response.failed":
		return unmarshalToType[ResponseFailed](a)
	case "response.incomplete":
		return unmarshalToType[ResponseIncomplete](a)
	case "response.output_item.added":
		return unmarshalToType[ResponseOutputItemAdded](a)
	case "response.output_item.done":
		return unmarshalToType[ResponseOutputItemDone](a)
	case "response.content_part.added":
		return unmarshalToType[ResponseContentPartAdded](a)
	case "response.content_part.done":
		return unmarshalToType[ResponseContentPartDone](a)
	case "response.output_text.delta":
		return unmarshalToType[ResponseOutputTextDelta](a)
	case "response.output_text.done":
		return unmarshalToType[ResponseOutputTextDone](a)
	case "response.refusal.delta":
		return unmarshalToType[ResponseRefusalDelta](a)
	case "response.refusal.done":
		return unmarshalToType[ResponseRefusalDone](a)
	case "response.function_call_arguments.delta":
		return unmarshalToType[ResponseFunctionCallArgumentsDelta](a)
	case "response.function_call_arguments.done":
		return unmarshalToType[ResponseFunctionCallArgumentsDone](a)
	case "response.file_search_call.in_progress":
		return unmarshalToType[ResponseFileSearchCallInProgress](a)
	case "response.file_search_call.searching":
		return unmarshalToType[ResponseFileSearchCallSearching](a)
	case "response.file_search_call.completed":
		return unmarshalToType[ResponseFileSearchCallCompleted](a)
	case "response.web_search_call.in_progress":
		return unmarshalToType[ResponseWebSearchCallInProgress](a)
	case "response.web_search_call.searching":
		return unmarshalToType[ResponseWebSearchCallSearching](a)
	case "response.web_search_call.completed":
		return unmarshalToType[ResponseWebSearchCallCompleted](a)
	case "response.reasoning_summary_part.added":
		return unmarshalToType[ResponseReasoningSummaryPartAdded](a)
	case "response.reasoning_summary_part.done":
		return unmarshalToType[ResponseReasoningSummaryPartDone](a)
	case "response.reasoning_summary_text.delta":
		return unmarshalToType[ResponseReasoningSummaryTextDelta](a)
	case "response.reasoning_summary_text.done":
		return unmarshalToType[ResponseReasoningSummaryTextDone](a)
	case "response.image_generation_call.completed":
		return unmarshalToType[ResponseImageGenerationCallCompleted](a)
	case "response.image_generation_call.generating":
		return unmarshalToType[ResponseImageGenerationCallGenerating](a)
	case "response.image_generation_call.in_progress":
		return unmarshalToType[ResponseImageGenerationCallInProgress](a)
	case "response.image_generation_call.partial_image":
		return unmarshalToType[ResponseImageGenerationCallPartialImage](a)
	case "response.mcp_call_arguments.delta":
		return unmarshalToType[ResponseMCPCallArgumentsDelta](a)
	case "response.mcp_call_arguments.done":
		return unmarshalToType[ResponseMCPCallArgumentsDone](a)
	case "response.mcp_call.completed":
		return unmarshalToType[ResponseMCPCallCompleted](a)
	case "response.mcp_call.failed":
		return unmarshalToType[ResponseMCPCallFailed](a)
	case "response.mcp_call.in_progress":
		return unmarshalToType[ResponseMCPCallInProgress](a)
	case "response.mcp_list_tools.completed":
		return unmarshalToType[ResponseMCPListToolsCompleted](a)
	case "response.mcp_list_tools.failed":
		return unmarshalToType[ResponseMCPListToolsFailed](a)
	case "response.mcp_list_tools.in_progress":
		return unmarshalToType[ResponseMCPListToolsInProgress](a)
	case "response.code_interpreter_call.in_progress":
		return unmarshalToType[ResponseCodeInterpreterCallInProgress](a)
	case "response.code_interpreter_call.interpreting":
		return unmarshalToType[ResponseCodeInterpreterCallInterpreting](a)
	case "response.code_interpreter_call.completed":
		return unmarshalToType[ResponseCodeInterpreterCallCompleted](a)
	case "response.code_interpreter_call_code.delta":
		return unmarshalToType[ResponseCodeInterpreterCallCodeDelta](a)
	case "response.code_interpreter_call_code.done":
		return unmarshalToType[ResponseCodeInterpreterCallCodeDone](a)
	case "response.output_text.annotation.added":
		return unmarshalToType[ResponseOutputTextAnnotationAdded](a)
	case "response.queued":
		return unmarshalToType[ResponseQueued](a)
	case "response.reasoning.delta":
		return unmarshalToType[ResponseReasoningDelta](a)
	case "response.reasoning.done":
		return unmarshalToType[ResponseReasoningDone](a)
	case "response.reasoning_summary.delta":
		return unmarshalToType[ResponseReasoningSummaryDelta](a)
	case "response.reasoning_summary.done":
		return unmarshalToType[ResponseReasoningSummaryDone](a)
	case "response.custom_tool_call_input.delta":
		return unmarshalToType[ResponseCustomToolCallInputDelta](a)
	case "response.custom_tool_call_input.done":
		return unmarshalToType[ResponseCustomToolCallInputDone](a)
	case "error":
		return unmarshalToType[Error](a)
	default:
		return nil, fmt.Errorf("unsupported event type: %s", a.Type)
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

// types containing repeating fields for embedding
type (
	// BaseEvent contains the common fields for all streaming events.
	BaseEvent struct {
		Type           string `json:"type"`
		SequenceNumber int    `json:"sequence_number"`
	}

	// OutputItemReference contains the common fields for events referencing an output item.
	OutputItemReference struct {
		BaseEvent
		OutputIndex int    `json:"output_index"`
		ItemID      string `json:"item_id"`
	}

	// ResponseEvent contains the common fields for events with response object payload.
	ResponseEvent struct {
		BaseEvent
		Response Response `json:"response"`
	}

	// Response represents a response object payload in streaming events.
	Response struct {
		ID        string `json:"id"`
		Object    string `json:"object"`     // "response"
		CreatedAt int    `json:"created_at"` // Unix timestamp (seconds)
		Status    string `json:"status"`     // "completed", "failed", "in_progress", "cancelled", "queued", or "incomplete"
		Error     *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
		IncompleteDetails *struct {
			Reason string `json:"reason"`
		} `json:"incomplete_details"`
		Instructions       json.RawMessage `json:"instructions"` // nil, string, or array of: InputMessage, Item, or ItemReference
		MaxOutputTokens    *int            `json:"max_output_tokens"`
		Model              string          `json:"model"`
		Output             json.RawMessage `json:"output"` // array of: OutputMessage, FileSearchCall, FunctionCall, WebSearchCall, ComputerCall, Reasoning, ImageGenerationCall, CodeInterpreterCall, LocalShellCall, MCPToolCall, MCPListTools, MCPApprovalRequest
		ParallelToolCalls  bool            `json:"parallel_tool_calls"`
		PreviousResponseID *string         `json:"previous_response_id"`
		Reasoning          *struct {
			Effort          *string `json:"effort"`           // "low", "medium", or "high"
			GenerateSummary *string `json:"generate_summary"` // Deprecated: use "summary" instead
			Summary         *string `json:"summary"`          // "auto", "concise", or "detailed"
		} `json:"reasoning"`
		Store       bool     `json:"store"`
		Temperature *float64 `json:"temperature"`
		Text        struct {
			Format struct {
				string `json:"type"` // "text", "json_schema", "json_object"

				// for json_schema only

				Name        string          `json:"name"`
				Schema      json.RawMessage `json:"schema"`
				Description string          `json:"description"`
				Strict      *bool           `json:"strict"`
			} `json:"format"`
		} `json:"text"`
		ToolChoice json.RawMessage `json:"tool_choice"` // string, ToolChoiceMode, HostedTool, FunctionTool, or MCPTool
		Tools      json.RawMessage `json:"tools"`       // array of Tool
		TopP       *float64        `json:"top_p"`
		Truncation *string         `json:"truncation"` // "auto", "disabled" (default)
		Usage      *struct {
			InputTokens        int `json:"input_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
				PromptTokens int `json:"prompt_tokens"`
			} `json:"input_tokens_details"`
			OutputTokens        int `json:"output_tokens"`
			OutputTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"output_tokens_details"`
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
		User         string            `json:"user"`
		Metadata     map[string]string `json:"metadata"`
		Background   *bool             `json:"background"`
		MaxToolCalls *int              `json:"max_tool_calls"`
		Prompt       *struct {
			ID        string                     `json:"id"`
			Variables map[string]json.RawMessage `json:"variables"`
			Version   *string                    `json:"version"`
		} `json:"prompt"`
		ServiceTier *string `json:"service_tier"` // "default", "auto", or "flex"
		TopLogprobs *int    `json:"top_logprobs"`
	}
)

// event types
type (
	ResponseCreated         ResponseEvent // response.created
	ResponseInProgress      ResponseEvent // response.in_progress
	ResponseCompleted       ResponseEvent // response.completed
	ResponseFailed          ResponseEvent // response.failed
	ResponseIncomplete      ResponseEvent // response.incomplete
	ResponseQueued          ResponseEvent // response.queued
	ResponseOutputItemAdded struct {      // response.output_item.added
		BaseEvent
		OutputIndex int             `json:"output_index"`
		Item        json.RawMessage `json:"item"` // OutputMessage, FileSearchCall, FunctionCall, WebSearchCall, ComputerCall, Reasoning, ImageGenerationCall, CodeInterpreterCall, LocalShellCall, MCPToolCall, MCPListTools, MCPApprovalRequest
	}
	ResponseOutputItemDone   ResponseOutputItemAdded // same fields // response.output_item.done
	ResponseContentPartAdded struct {                // response.content_part.added
		OutputItemReference
		ContentIndex int             `json:"content_index"`
		Part         json.RawMessage `json:"part"` // OutputText or Refusal
	}
	ResponseContentPartDone ResponseContentPartAdded // same fields // response.content_part.done
	ResponseOutputTextDelta struct {                 // response.output_text.delta
		OutputItemReference
		ContentIndex int    `json:"content_index"`
		Delta        string `json:"delta"`
	}
	ResponseOutputTextDone struct { // response.output_text.done
		OutputItemReference
		ContentIndex int    `json:"content_index"`
		Text         string `json:"text"`
	}
	ResponseRefusalDelta ResponseOutputTextDelta // same fields // response.refusal.delta
	ResponseRefusalDone  struct {                // response.refusal.done
		OutputItemReference
		ContentIndex int    `json:"content_index"`
		Refusal      string `json:"refusal"`
	}
	ResponseFunctionCallArgumentsDelta struct { // response.function_call_arguments.delta
		OutputItemReference
		Delta string `json:"delta"`
	}
	ResponseFunctionCallArgumentsDone struct { // response.function_call_arguments.done
		OutputItemReference
		Arguments json.RawMessage `json:"arguments"`
	}
	ResponseFileSearchCallInProgress  OutputItemReference // response.file_search_call.in_progress
	ResponseFileSearchCallSearching   OutputItemReference // response.file_search_call.searching
	ResponseFileSearchCallCompleted   OutputItemReference // response.file_search_call.completed
	ResponseWebSearchCallInProgress   OutputItemReference // response.web_search_call.in_progress
	ResponseWebSearchCallSearching    OutputItemReference // response.web_search_call.searching
	ResponseWebSearchCallCompleted    OutputItemReference // response.web_search_call.completed
	ResponseReasoningSummaryPartAdded struct {            // response.reasoning_summary_part.added
		OutputItemReference
		SummaryIndex int `json:"summary_index"`
		Part         struct {
			Type string `json:"type"` // "summary_text"
			Text string `json:"text"`
		} `json:"part"` // ReasoningSummaryPart
	}
	ResponseReasoningSummaryPartDone  ResponseReasoningSummaryPartAdded // same fields // response.reasoning_summary_part.done
	ResponseReasoningSummaryTextDelta struct {                          // response.reasoning_summary_text.delta
		OutputItemReference
		SummaryIndex int    `json:"summary_index"`
		Delta        string `json:"delta"`
	}
	ResponseReasoningSummaryTextDone struct { // response.reasoning_summary_text.done
		OutputItemReference
		SummaryIndex int    `json:"summary_index"`
		Text         string `json:"text"`
	}
	ResponseImageGenerationCallCompleted    OutputItemReference // response.image_generation_call.completed
	ResponseImageGenerationCallGenerating   OutputItemReference // response.image_generation_call.generating
	ResponseImageGenerationCallInProgress   OutputItemReference // response.image_generation_call.in_progress
	ResponseImageGenerationCallPartialImage struct {            // response.image_generation_call.partial_image
		OutputItemReference
		PartialImageIndex int    `json:"partial_image_index"`
		PartialImageB64   string `json:"partial_image_b64"`
	}
	ResponseMCPCallArgumentsDelta struct { // response.mcp_call_arguments.delta
		OutputItemReference
		Delta map[string]json.RawMessage `json:"delta"` // type of values is not documented, likely any json value
	}
	ResponseMCPCallArgumentsDone            ResponseMCPCallArgumentsDelta      // same fields // response.mcp_call_arguments.done
	ResponseMCPCallCompleted                BaseEvent                          // response.mcp_call.completed
	ResponseMCPCallFailed                   BaseEvent                          // response.mcp_call.failed
	ResponseMCPCallInProgress               OutputItemReference                // response.mcp_call.in_progress
	ResponseMCPListToolsCompleted           BaseEvent                          // response.mcp_list_tools.completed
	ResponseMCPListToolsFailed              BaseEvent                          // response.mcp_list_tools.failed
	ResponseMCPListToolsInProgress          BaseEvent                          // response.mcp_list_tools.in_progress
	ResponseCodeInterpreterCallInProgress   OutputItemReference                // response.code_interpreter_call.in_progress
	ResponseCodeInterpreterCallInterpreting OutputItemReference                // response.code_interpreter_call.interpreting
	ResponseCodeInterpreterCallCompleted    OutputItemReference                // response.code_interpreter_call.completed
	ResponseCodeInterpreterCallCodeDelta    ResponseFunctionCallArgumentsDelta // same fields // response.code_interpreter_call_code.delta
	ResponseCodeInterpreterCallCodeDone     struct {                           // response.code_interpreter_call_code.done
		OutputItemReference
		Code string `json:"code"`
	}
	ResponseOutputTextAnnotationAdded struct { // response.output_text.annotation.added
		OutputItemReference
		ContentIndex    int `json:"content_index"`
		AnnotationIndex int `json:"annotation_index"`
		Annotation      struct {
			Type  string `json:"type"` // "text_annotation"
			Text  string `json:"text"`
			Start int    `json:"start"`
			End   int    `json:"end"`
		} `json:"annotation"`
	}
	ResponseReasoningDelta struct { // response.reasoning.delta
		OutputItemReference
		ContentIndex int `json:"content_index"`
		Delta        struct {
			Text string `json:"text"`
		} `json:"delta"`
	}
	ResponseReasoningDone         ResponseOutputTextDone // same fields // response.reasoning.done
	ResponseReasoningSummaryDelta struct {               // response.reasoning_summary.delta
		OutputItemReference
		SummaryIndex int `json:"summary_index"`
		Delta        struct {
			Text string `json:"text"`
		} `json:"delta"`
	}
	ResponseReasoningSummaryDone     ResponseReasoningSummaryTextDone // same fields // response.reasoning_summary.done
	ResponseCustomToolCallInputDelta struct {                         // response.custom_tool_call_input.delta
		OutputItemReference
		Delta string `json:"delta"`
	}
	ResponseCustomToolCallInputDone struct { // response.custom_tool_call_input.done
		OutputItemReference
		Input string `json:"input"`
	}
	Error struct { // error
		BaseEvent
		Message string  `json:"message"`
		Code    string  `json:"code"`
		Param   *string `json:"param"`
	}
)
