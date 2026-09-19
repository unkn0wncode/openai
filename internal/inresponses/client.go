// Package inresponses / client.go implements Responses requests, tool handling, polling, and SSE streaming.
package inresponses

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/tools"
)

// Client is the client for the Responses API.
type Client struct {
	*openai.Config
}

// NewClient creates a new ResponsesClient.
func NewClient(config *openai.Config) *Client {
	return &Client{Config: config}
}

// builtinTools is a list of tools that are built into the Responses API.
var builtinTools = []string{
	"web_search",
	"web_search_preview",
	"file_search",
	"computer_use_preview",
	"mcp",
	"local_shell",
	"code_interpreter",
	"shell",
	"apply_patch",
	"image_generation",
	"programmatic_tool_calling",
}

// interface compliance checks
var _ responses.Service = (*Client)(nil)

// marshalRequest marshals the request into a JSON object, including tools by name.
func (c *Client) marshalRequest(data *responses.Request) ([]byte, error) {
	if data == nil {
		return nil, errors.New("request is nil")
	}

	if len(data.Tools) == 0 {
		type Alias responses.Request
		return openai.Marshal((*Alias)(data))
	}

	var toolList []tools.Tool
	for _, name := range data.Tools {
		// if given tool is builtin, add it by type
		if slices.Contains(builtinTools, name) {
			for _, t := range c.Tools.Tools {
				if t.Type == name {
					toolList = append(toolList, t)
					break
				}
			}
			continue
		}

		// try to get tool by name, if not found try to get function by name
		t, ok := c.Tools.GetTool(name)
		if ok {
			toolList = append(toolList, t)
			continue
		}

		f, ok := c.Tools.GetFunction(name)
		if ok {
			toolList = append(toolList, tools.Tool{
				Type:           "function",
				Name:           f.Name,
				Description:    f.Description,
				Parameters:     f.ParamsSchema,
				Strict:         f.Strict,
				Async:          f.Async,
				AllowedCallers: f.AllowedCallers,
				OutputSchema:   f.OutputSchema,
				Function:       f,
			})
			continue
		}

		return nil, fmt.Errorf("tool/function '%s' is not registered", name)
	}

	type Alias responses.Request
	return openai.Marshal(&struct {
		Tools []tools.Tool `json:"tools"`
		*Alias
	}{
		Tools: toolList,
		Alias: (*Alias)(data),
	})
}

// executeRequest sends one request and returns:
//   - result: the decoded API response, or nil if decoding did not succeed.
//   - outcomeKnown: true for a decoded response, a local failure before sending,
//     or an HTTP rejection below status 500. False means processing may have
//     occurred without a decoded response, so billing evidence is incomplete.
//   - err: a validation, transport, HTTP status, body-read, or decoding error.
//     Errors and terminal statuses inside a decoded response are handled by send.
func (c *Client) executeRequest(data *responses.Request) (result *response, outcomeKnown bool, err error) {
	if data == nil {
		return nil, true, fmt.Errorf("request is nil")
	}

	if data.Model == "" {
		data.Model = models.Default
	}

	// Check if we have input
	if data.Input == nil {
		return nil, true, fmt.Errorf("input is required")
	}

	if data.Stream {
		return nil, true, fmt.Errorf("request has 'stream' parameter but was invoked with Send method, use Stream method instead")
	}

	b, err := c.marshalRequest(data)
	if err != nil {
		return nil, true, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseAPI+"v1/responses", bytes.NewBuffer(b))
	if err != nil {
		return nil, true, fmt.Errorf("failed to create request: %w", err)
	}
	c.addResponseHeaders(req, data)

	var resp *http.Response
	before := time.Now()
	resp, err = c.HTTPClient.Do(req)
	duration := time.Since(before)
	if err != nil {
		return nil, false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && (resp.StatusCode != http.StatusAccepted || !data.Background) {
		return nil, resp.StatusCode < http.StatusInternalServerError,
			fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(body))
	}

	var res response
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, false, fmt.Errorf("failed to decode response: %w", err)
	}
	res.ProcessingRegion = processingRegion(req.URL)
	res.Duration = duration
	if res.Tools == nil {
		// Preserve the resolved definitions actually sent, even when not echoed.
		var sent struct {
			Tools []tools.Tool `json:"tools"`
		}
		if err := json.Unmarshal(b, &sent); err != nil {
			c.Log.Warn("Failed to decode sent tool definitions", slog.Any("error", err))
		} else {
			res.Tools = sent.Tools
		}
	}
	return &res, true, nil
}

// response is the response body from the Responses API.
type response struct {
	// Core Properties
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	CreatedAt         int                     `json:"created_at"` // Unix timestamp
	Status            string                  `json:"status"`     // "completed", "failed", "in_progress", or "incomplete"
	Error             any                     `json:"error"`      // Error object with code and message
	IncompleteDetails any                     `json:"incomplete_details"`
	Instructions      any                     `json:"instructions"` // string, []output.Any
	Conversation      *responses.Conversation `json:"conversation"`
	MaxOutputTokens   int                     `json:"max_output_tokens"`
	Model             string                  `json:"model"`
	ServiceTier       string                  `json:"service_tier"`

	// Output Content
	Output []output.Any `json:"output"`

	// Tool and Configuration Properties
	ParallelToolCalls      bool                              `json:"parallel_tool_calls"`
	PreviousResponseID     any                               `json:"previous_response_id"`
	Reasoning              *responses.ReasoningConfig        `json:"reasoning"`
	MultiAgent             *responses.MultiAgentConfig       `json:"multi_agent"`
	PromptCacheOptions     *responses.PromptCacheOptions     `json:"prompt_cache_options"`
	PromptCacheDiagnostics *responses.PromptCacheDiagnostics `json:"prompt_cache_diagnostics"`
	Store                  bool                              `json:"store"`
	Temperature            float64                           `json:"temperature"`
	Text                   struct {
		Format struct {
			Type string `json:"type"`
		} `json:"format"`
	} `json:"text"`
	ToolChoice json.RawMessage `json:"tool_choice"`
	Tools      []tools.Tool    `json:"tools"`
	TopP       float64         `json:"top_p"`
	Truncation string          `json:"truncation"`

	// Usage Information
	Usage *responses.Usage `json:"usage"`

	// Other Properties
	User             string         `json:"user"`
	Metadata         map[string]any `json:"metadata"`
	ProcessingRegion string         `json:"-"`
	Duration         time.Duration  `json:"-"`
}

// project preserves billing evidence before output parsing or local tool execution.
func (data *response) project() *responses.Response {
	return &responses.Response{
		ID:                     data.ID,
		Model:                  data.Model,
		ServiceTier:            data.ServiceTier,
		Status:                 data.Status,
		Outputs:                data.Output,
		Usage:                  data.Usage,
		Tools:                  data.Tools,
		ProcessingRegion:       data.ProcessingRegion,
		Reasoning:              data.Reasoning,
		MultiAgent:             data.MultiAgent,
		PromptCacheOptions:     data.PromptCacheOptions,
		PromptCacheDiagnostics: data.PromptCacheDiagnostics,
	}
}

// addResponseHeaders includes the beta opt-in when a Multi-agent configuration is supplied.
func (c *Client) addResponseHeaders(req *http.Request, data *responses.Request) {
	c.AddHeaders(req)
	if data.MultiAgent != nil {
		req.Header.Set("OpenAI-Beta", responses.MultiAgentBeta)
	}
}

func (data *response) checkResponseData(resp *responses.Response) error {
	var statusErr error
	switch data.Status {
	case "queued", "in_progress":
		return nil
	case "completed":
	case "failed":
		statusErr = fmt.Errorf("response %s failed: %v", data.ID, data.Error)
	case "incomplete":
		statusErr = fmt.Errorf("response %s incomplete: %v", data.ID, data.IncompleteDetails)
	case "cancelled":
		statusErr = fmt.Errorf("response %s cancelled", data.ID)
	default:
		statusErr = fmt.Errorf("response %s has unexpected status %q", data.ID, data.Status)
	}
	if data.Error != nil && statusErr == nil {
		statusErr = fmt.Errorf("got API error: %v", data.Error)
	}
	if err := resp.Parse(); err != nil {
		return errors.Join(statusErr, fmt.Errorf("failed to parse output: %w", err))
	}
	if statusErr != nil {
		return statusErr
	}
	if len(resp.Outputs) == 0 {
		return fmt.Errorf("no output returned")
	}

	for _, o := range resp.ParsedOutputs {
		if m, ok := o.(output.Message); ok {
			if m.Content == nil {
				return fmt.Errorf("no content in output message (nil content)")
			}
			if content, ok := m.Content.([]any); ok && len(content) == 0 {
				return fmt.Errorf("no content in output message (zero length []any content)")
			}
			switch m.Status {
			case "", "completed", "incomplete", "error":
			default:
				return fmt.Errorf("got unexpected status: %s", m.Status)
			}
		}
	}
	return nil
}

// processingRegion retyrns region name recognized by documented OpenAI endpoints; arbitrary proxies
// do not establish the region used for billing.
func processingRegion(endpoint *url.URL) string {
	host := strings.ToLower(endpoint.Hostname())
	switch host {
	case "api.openai.com":
		return "global"
	case "us.api.openai.com", "eu.api.openai.com", "au.api.openai.com", "ca.api.openai.com",
		"jp.api.openai.com", "in.api.openai.com", "sg.api.openai.com", "kr.api.openai.com",
		"gb.api.openai.com", "ae.api.openai.com":
		return strings.TrimSuffix(host, ".api.openai.com")
	default:
		return ""
	}
}

// logCost records terminal usage and the estimate already stored on resp.
func (c *Client) logCost(resp *responses.Response, duration time.Duration, metadata any) {
	switch resp.Status {
	case "queued", "in_progress":
		return
	case "completed", "failed", "incomplete", "cancelled":
	default:
		c.Log.Warn(fmt.Sprintf("Unexpected response status %q", resp.Status), slog.String("responseID", resp.ID))
	}
	if _, ok := models.Data[resp.Model]; !ok {
		c.Log.Warn(fmt.Sprintf("No pricing found for model %q", resp.Model))
	}
	amount, err := resp.EstimatedCost, resp.CostError
	attrs := []any{
		slog.String("responseID", resp.ID),
		slog.String("model", resp.Model),
		slog.String("serviceTier", resp.ServiceTier),
		slog.String("status", resp.Status),
		slog.Any("duration", duration),
		slog.Any("metadata", metadata),
	}
	message := "OpenAI Responses usage unavailable"
	if resp.Usage != nil {
		usage := resp.Usage
		message = fmt.Sprintf("Consumed OpenAI Responses tokens: %d + %d = %d", usage.InputTokens, usage.OutputTokens, usage.TotalTokens)
	}
	switch {
	case err == nil:
		message += fmt.Sprintf(" (estimated cost $%.9f)", amount)
	case amount != 0:
		message += fmt.Sprintf(" (known cost subtotal $%.9f; estimate incomplete)", amount)
	default:
		message += " (cost estimate unavailable)"
	}
	if err != nil {
		attrs = append(attrs, slog.Any("costError", err))
	}
	c.Log.Debug(message, attrs...)
}

// logStreamingCost uses the same accounting for all terminal response events.
func (c *Client) logStreamingCost(req *responses.Request, sentTools []tools.Tool, region string, event any) {
	var streamed streaming.Response
	switch event := event.(type) {
	case streaming.ResponseCompleted:
		streamed = event.Response
	case streaming.ResponseIncomplete:
		streamed = event.Response
	case streaming.ResponseFailed:
		streamed = event.Response
	default:
		return
	}
	resp := &responses.Response{
		ID:               streamed.ID,
		Model:            streamed.Model,
		Status:           streamed.Status,
		ProcessingRegion: region,
		Tools:            sentTools,
	}
	if streamed.ServiceTier != nil {
		resp.ServiceTier = *streamed.ServiceTier
	}
	if streamed.Usage != nil {
		usage := responses.Usage(*streamed.Usage)
		resp.Usage = &usage
	}
	if streamed.Reasoning != nil {
		resp.Reasoning = &responses.ReasoningConfig{}
		if streamed.Reasoning.Effort != nil {
			resp.Reasoning.Effort = *streamed.Reasoning.Effort
		}
		if streamed.Reasoning.Mode != nil {
			resp.Reasoning.Mode = *streamed.Reasoning.Mode
		}
		if streamed.Reasoning.Context != nil {
			resp.Reasoning.Context = *streamed.Reasoning.Context
		}
		if streamed.Reasoning.Summary != nil {
			resp.Reasoning.Summary = *streamed.Reasoning.Summary
		}
		if streamed.Reasoning.GenerateSummary != nil {
			resp.Reasoning.GenerateSummary = *streamed.Reasoning.GenerateSummary
		}
	}
	if streamed.MultiAgent != nil {
		resp.MultiAgent = &responses.MultiAgentConfig{
			Enabled:                streamed.MultiAgent.Enabled,
			MaxConcurrentSubagents: streamed.MultiAgent.MaxConcurrentSubagents,
		}
	}
	if streamed.PromptCacheOptions != nil {
		resp.PromptCacheOptions = &responses.PromptCacheOptions{
			Mode:                 streamed.PromptCacheOptions.Mode,
			TTL:                  streamed.PromptCacheOptions.TTL,
			ComparisonResponseID: streamed.PromptCacheOptions.ComparisonResponseID,
		}
	}
	if streamed.PromptCacheDiagnostics != nil {
		resp.PromptCacheDiagnostics = &responses.PromptCacheDiagnostics{
			Type:                     streamed.PromptCacheDiagnostics.Type,
			Reason:                   streamed.PromptCacheDiagnostics.Reason,
			ComparisonReusableTokens: streamed.PromptCacheDiagnostics.ComparisonReusableTokens,
			CacheMissedTokens:        streamed.PromptCacheDiagnostics.CacheMissedTokens,
		}
	}
	if len(streamed.Output) > 0 {
		if err := json.Unmarshal(streamed.Output, &resp.Outputs); err != nil {
			c.Log.Warn("Failed to decode streaming billing outputs", slog.Any("error", err))
			resp.BillingIncomplete = true
		}
	}
	if len(streamed.Tools) > 0 && string(streamed.Tools) != "null" {
		if err := json.Unmarshal(streamed.Tools, &resp.Tools); err != nil {
			c.Log.Warn("Failed to decode streaming tool definitions", slog.Any("error", err))
			resp.BillingIncomplete = true
		}
	}
	resp.EstimatedCost, resp.CostError = req.EstimateCost(resp)
	c.logCost(resp, 0, streamed.Metadata)
}

// executableFunctionCall is an intermediate representation of a function call that can be executed.
type executableFunctionCall struct {
	Name      string
	CallID    string
	Arguments json.RawMessage
	F         func(params json.RawMessage) (string, error)
	CallLimit int
	Caller    *output.Caller
	Agent     *output.Agent
}

// executableCustomToolCall is an intermediate representation of a custom tool call that can be executed.
type executableCustomToolCall struct {
	Name   string
	CallID string
	Input  string
	F      func(input string) (string, error)
	Caller *output.Caller
	Agent  *output.Agent
}

// sendContext tracks per-Send state across follow-up requests.
type sendContext struct {
	callCounts        map[string]int
	blockedTools      map[string]struct{}
	calls             []responses.Response
	billingIncomplete bool
}

// newSendContext initializes per-Send tracking state.
func newSendContext() *sendContext {
	return &sendContext{
		callCounts:   map[string]int{},
		blockedTools: map[string]struct{}{},
	}
}

// Send sends a request to the Responses API with custom data.
// Returns the AI reply, request ID, and any error.
func (c *Client) Send(req *responses.Request) (*responses.Response, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	sc := newSendContext()
	resp, err := c.send(req.Clone(), sc)
	if resp == nil && sc.billingIncomplete {
		resp = &responses.Response{}
	}
	if resp != nil {
		resp.Calls = sc.calls
		resp.BillingIncomplete = sc.billingIncomplete
		resp.EstimatedCost, resp.CostError = req.EstimateCost(resp)
	}
	return resp, err
}

func (c *Client) send(req *responses.Request, sc *sendContext) (*responses.Response, error) {
	respData, outcomeKnown, err := c.executeRequest(req)
	if !outcomeKnown {
		sc.billingIncomplete = true
	}
	if err != nil {
		return nil, err
	}

	resp := respData.project()
	sc.calls = append(sc.calls, *resp)
	call := &sc.calls[len(sc.calls)-1]
	call.EstimatedCost, call.CostError = req.EstimateCost(call)
	c.logCost(call, respData.Duration, respData.Metadata)
	resp.EstimatedCost, resp.CostError = call.EstimatedCost, call.CostError
	if err := respData.checkResponseData(resp); err != nil {
		return resp, err
	}
	if resp.Status == "queued" || resp.Status == "in_progress" {
		return resp, nil
	}

	// log refusals as warnings
	for _, refusal := range resp.Refusals() {
		c.Log.Warn(fmt.Sprintf("got refusal: %s", refusal))
	}
	if hasPendingClientWork(resp.Outputs) {
		return resp, nil
	}

	// First pass: analyze outputs and categorize them
	var messages []output.Message
	var messageOutputs []output.Any
	var executableCalls []executableFunctionCall
	var executableCustomCalls []executableCustomToolCall
	var returnableCalls []output.FunctionCall
	var returnableCustomCalls []output.CustomToolCall
	var otherOutputs []output.Any
	var otherParsedOutputs []any

	for i, anyOutput := range resp.ParsedOutputs {
		switch o := anyOutput.(type) {
		case output.Message:
			messages = append(messages, o)
			messageOutputs = append(messageOutputs, resp.Outputs[i])
		case output.FunctionCall:
			if req.ReturnToolCalls || o.Async {
				returnableCalls = append(returnableCalls, o)
				continue
			}

			// Get the tool or function from the registered function calls
			var (
				F         func(params json.RawMessage) (string, error)
				callLimit int
			)
			if t, ok := c.Tools.GetTool(o.Name); ok {
				if t.Function.F != nil {
					F = t.Function.F
					callLimit = t.Function.CallLimit
				} else {
					returnableCalls = append(returnableCalls, o)
					continue
				}
			} else if f, ok := c.Tools.GetFunction(o.Name); ok {
				if f.F != nil {
					F = f.F
					callLimit = f.CallLimit
				} else {
					returnableCalls = append(returnableCalls, o)
					continue
				}
			} else {
				return resp, fmt.Errorf("tool/function '%s' is not registered", o.Name)
			}

			executableCalls = append(executableCalls, executableFunctionCall{
				Name:      o.Name,
				CallID:    o.CallID,
				Arguments: []byte(o.Arguments),
				F:         F,
				CallLimit: callLimit,
				Caller:    o.Caller,
				Agent:     o.Agent,
			})
		case output.CustomToolCall:
			if req.ReturnToolCalls || o.Async {
				returnableCustomCalls = append(returnableCustomCalls, o)
				continue
			}

			// Get the tool by name from the registered tools
			t, ok := c.Tools.GetTool(o.Name)
			if !ok {
				return resp, fmt.Errorf("tool '%s' is not registered", o.Name)
			}
			if t.Type != "custom" {
				return resp, fmt.Errorf("tool '%s' is not a custom tool", o.Name)
			}
			if t.Custom == nil {
				returnableCustomCalls = append(returnableCustomCalls, o)
				continue
			}

			executableCustomCalls = append(executableCustomCalls, executableCustomToolCall{
				Name:   o.Name,
				CallID: o.CallID,
				Input:  o.Input,
				F:      t.Custom,
				Caller: o.Caller,
				Agent:  o.Agent,
			})
		default:
			otherOutputs = append(otherOutputs, resp.Outputs[i])
			otherParsedOutputs = append(otherParsedOutputs, o)
		}
	}

	switch {
	// Case 1: All outputs are messages/other outputs
	case len(executableCalls) == 0 && len(executableCustomCalls) == 0 && len(returnableCalls) == 0 && len(returnableCustomCalls) == 0:
		if !hasFinalMessage(messages) && hasProgrammaticWork(resp) && !req.ReturnToolCalls {
			if req.IntermediateMessageHandler != nil {
				for _, msg := range messages {
					req.IntermediateMessageHandler(msg)
				}
				return c.continueResponse(req, sc, resp, nil, otherOutputs, otherParsedOutputs)
			}
			return c.continueResponse(req, sc, resp, nil, resp.Outputs, resp.ParsedOutputs)
		}
		return resp, nil

	// Case 2: Any returnable function/custom calls present
	case len(returnableCalls) > 0 || len(returnableCustomCalls) > 0:
		return resp, nil

	// Case 3: Mix of messages and executable function/custom calls
	case len(executableCalls) > 0 || len(executableCustomCalls) > 0:
		// Handle messages with intermediate handler if set
		if req.IntermediateMessageHandler != nil {
			for _, msg := range messages {
				req.IntermediateMessageHandler(msg)
			}
		}

		// Execute calls and collect outputs (mixed types)
		var toolOutputs []output.Any
		// function calls
		for _, call := range executableCalls {
			fResult, err := call.F(call.Arguments)
			switch {
			case err == nil:
			case errors.Is(err, tools.ErrDoNotRespond):
				// Here we return ID despite error because this error indicates intended behavior
				return resp, nil
			default:
				return resp, fmt.Errorf("failed to execute function '%s': %w", call.Name, err)
			}
			// Add function_call_output
			var anyOut output.Any
			b, _ := json.Marshal(output.FunctionCallOutput{
				Type:   "function_call_output",
				CallID: call.CallID,
				Output: fResult,
				Caller: call.Caller,
				Agent:  call.Agent,
			})
			if err := json.Unmarshal(b, &anyOut); err != nil {
				return resp, fmt.Errorf("failed to prepare function_call_output: %w", err)
			}
			toolOutputs = append(toolOutputs, anyOut)

			if call.CallLimit > 0 {
				sc.callCounts[call.Name]++
				if sc.callCounts[call.Name] >= call.CallLimit {
					c.Log.Warn(fmt.Sprintf(
						"Function '%s' has reached its CallLimit (%d) times, excluding from further tool calls",
						call.Name, sc.callCounts[call.Name],
					))
					sc.blockedTools[call.Name] = struct{}{}
				}
			}
		}
		// custom tool calls
		for _, call := range executableCustomCalls {
			fResult, err := call.F(call.Input)
			switch {
			case err == nil:
			case errors.Is(err, tools.ErrDoNotRespond):
				return resp, nil
			default:
				return resp, fmt.Errorf("failed to execute custom tool '%s': %w", call.Name, err)
			}
			// Add custom_tool_call_output
			var anyOut output.Any
			b, _ := json.Marshal(output.CustomToolCallOutput{
				Type:   "custom_tool_call_output",
				CallID: call.CallID,
				Output: fResult,
				Caller: call.Caller,
				Agent:  call.Agent,
			})
			if err := json.Unmarshal(b, &anyOut); err != nil {
				return resp, fmt.Errorf("failed to prepare custom_tool_call_output: %w", err)
			}
			toolOutputs = append(toolOutputs, anyOut)
		}

		// Keep outputs the caller has not already handled.
		var previousOutputs []output.Any
		var previousParsed []any
		if req.IntermediateMessageHandler == nil {
			previousOutputs = append(previousOutputs, messageOutputs...)
			for _, msg := range messages {
				previousParsed = append(previousParsed, msg)
			}
		}
		previousOutputs = append(previousOutputs, otherOutputs...)
		previousParsed = append(previousParsed, otherParsedOutputs...)
		return c.continueResponse(req, sc, resp, toolOutputs, previousOutputs, previousParsed)

	// Case 4: Only other outputs
	case len(otherOutputs) > 0:
		return resp, nil
	}

	// This should be unreachable
	return resp, fmt.Errorf("logic error: unreachable code, stack: %s", string(debug.Stack()))
}

// hasPendingClientWork keeps approvals and application-owned tools available to
// the caller rather than continuing a paused program with no result. A matching
// output in the same response establishes that a hosted call already finished.
func hasPendingClientWork(items []output.Any) bool {
	type reference struct {
		ID                string `json:"id"`
		CallID            string `json:"call_id"`
		ApprovalRequestID string `json:"approval_request_id"`
	}
	resolved := make(map[string]bool)
	for _, item := range items {
		var ref reference
		switch item.Type {
		case "shell_call_output", "apply_patch_call_output", "computer_call_output", "local_shell_call_output",
			"mcp_approval_response", "mcp_call":
			if item.UnmarshalToTarget(&ref) != nil {
				continue
			}
		default:
			continue
		}
		switch item.Type {
		case "mcp_approval_response", "mcp_call":
			if ref.ApprovalRequestID != "" {
				resolved["mcp_approval_request:"+ref.ApprovalRequestID] = true
			}
		case "local_shell_call_output":
			if ref.ID != "" {
				resolved["local_shell_call:"+ref.ID] = true
			}
		default:
			if ref.CallID != "" {
				resolved[strings.TrimSuffix(item.Type, "_output")+":"+ref.CallID] = true
			}
		}
	}
	for _, item := range items {
		switch item.Type {
		case "mcp_approval_request", "shell_call", "apply_patch_call", "computer_call", "local_shell_call":
		default:
			continue
		}
		var ref reference
		if item.UnmarshalToTarget(&ref) != nil {
			return true
		}
		id := ref.CallID
		if item.Type == "mcp_approval_request" || item.Type == "local_shell_call" {
			id = ref.ID
		}
		if id == "" || !resolved[item.Type+":"+id] {
			return true
		}
	}
	return false
}

// hasProgrammaticWork identifies a response that may still need a final message.
func hasProgrammaticWork(resp *responses.Response) bool {
	for _, item := range resp.Outputs {
		if item.Type == "program" || item.Type == "program_output" {
			return true
		}
	}
	for _, tool := range resp.Tools {
		if tool.Type == "programmatic_tool_calling" {
			return true
		}
	}
	return false
}

// hasFinalMessage excludes commentary and subagent replies from the root answer.
func hasFinalMessage(messages []output.Message) bool {
	for _, message := range messages {
		if message.Role == "assistant" && (message.Phase == "" || message.Phase == "final_answer") &&
			(message.Agent == nil || message.Agent.AgentName == "/root") {
			return true
		}
	}
	return false
}

// continueResponse preserves the appropriate stored or stateless context before
// sending tool results or asking for the final message after a program finishes.
func (c *Client) continueResponse(req *responses.Request, sc *sendContext, resp *responses.Response,
	toolOutputs, previousOutputs []output.Any, previousParsed []any,
) (*responses.Response, error) {
	followUp := req.Clone()
	followUp.Input = append([]output.Any{}, toolOutputs...)
	switch {
	case req.Conversation != nil:
		// The conversation already stores the prior input and output items.
		followUp.PreviousResponseID = ""
	case req.Store != nil && !*req.Store:
		history, err := replayInput(req.Input)
		if err != nil {
			return resp, err
		}
		history = append(history, resp.Outputs...)
		history = append(history, toolOutputs...)
		followUp.Input = history
	default:
		followUp.PreviousResponseID = resp.ID
	}
	followUp.Tools = filterBlockedTools(followUp.Tools, sc.blockedTools)
	if len(sc.blockedTools) > 0 {
		followUp.ToolChoice = nil
	}
	next, err := c.send(followUp, sc)
	if next == nil {
		return resp, err
	}
	*resp = *next
	resp.Outputs = append(previousOutputs, next.Outputs...)
	resp.ParsedOutputs = append(previousParsed, next.ParsedOutputs...)
	return resp, err
}

// replayInput converts the input to raw items so stateless continuations retain
// all wire fields, including program fingerprints, caller and agent metadata.
func replayInput(value any) ([]output.Any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode replay input: %w", err)
	}
	if len(data) > 0 && data[0] == '"' {
		var text string
		if err := json.Unmarshal(data, &text); err != nil {
			return nil, fmt.Errorf("decode replay text: %w", err)
		}
		data, err = json.Marshal([]output.Message{{Role: "user", Content: text}})
		if err != nil {
			return nil, fmt.Errorf("encode replay message: %w", err)
		}
	}
	var items []output.Any
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("decode replay items: %w", err)
	}
	return items, nil
}

// filterBlockedTools removes blocked tool names from the provided list.
func filterBlockedTools(tools []string, blocked map[string]struct{}) []string {
	if len(tools) == 0 || len(blocked) == 0 {
		return tools
	}

	filtered := make([]string, 0, len(tools))
	for _, name := range tools {
		if _, isBlocked := blocked[name]; isBlocked {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

// NewMessage creates a new empty message.
func (c *Client) NewMessage() *output.Message {
	return &output.Message{}
}

// NewRequest creates a new empty request.
func (c *Client) NewRequest() *responses.Request {
	return &responses.Request{}
}

// Poll continuously fetches a previously created background response until
// completion or failure. ctx controls cancellation, interval specifies wait between polls.
func (c *Client) Poll(ctx context.Context, id string, interval time.Duration) (*responses.Response, error) {
	targetURL := fmt.Sprintf("%sv1/responses/%s", c.BaseAPI, id)
	var last *responses.Response
	for {
		if err := ctx.Err(); err != nil {
			return last, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
		if err != nil {
			return last, fmt.Errorf("failed to create poll request: %w", err)
		}
		c.AddHeaders(req)
		before := time.Now()
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return last, fmt.Errorf("failed to send poll request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return last, fmt.Errorf("failed to read poll response: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return last, fmt.Errorf("poll request failed with status: %s, body: %s", resp.Status, string(body))
		}

		var raw response
		if err := json.Unmarshal(body, &raw); err != nil {
			return last, fmt.Errorf("failed to decode poll response: %w", err)
		}
		raw.ProcessingRegion = processingRegion(req.URL)
		last = raw.project()
		last.Calls = []responses.Response{*last}
		last.EstimatedCost, last.CostError = (&responses.Request{}).EstimateCost(last)
		c.logCost(last, time.Since(before), raw.Metadata)
		switch raw.Status {
		case "queued", "in_progress":
		default:
			return last, raw.checkResponseData(last)
		}

		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// sseStream is the streaming.Source backing an SSE /v1/responses request.
// The reader goroutine is the only event writer.
type sseStream struct {
	events   chan any
	done     chan struct{}
	once     sync.Once
	err      error
	cancel   context.CancelFunc
	body     io.Closer
	bodyOnce sync.Once
}

var _ streaming.Source = (*sseStream)(nil)

func (h *sseStream) Events() <-chan any { return h.events }

func (h *sseStream) Done() <-chan struct{} { return h.done }

func (h *sseStream) Err() error {
	<-h.done
	return h.err
}

func (h *sseStream) Close() {
	h.finish(nil)
	h.cancel()
	h.closeBody()
}

func (h *sseStream) closeBody() {
	h.bodyOnce.Do(func() {
		_ = h.body.Close()
	})
}

func (h *sseStream) finish(err error) {
	h.once.Do(func() {
		h.err = err
		close(h.done)
	})
}

// streamEvents sends a request with "stream":true and returns a streaming.Source
// backed by the SSE response body.
func (c *Client) streamEvents(ctx context.Context, data *responses.Request) (streaming.Source, error) {
	if data == nil {
		return nil, fmt.Errorf("request is nil")
	}
	data = data.Clone()

	if data.Model == "" {
		data.Model = models.Default
	}

	if data.Input == nil {
		return nil, fmt.Errorf("input is required")
	}

	data.Stream = true

	b, err := c.marshalRequest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	var sent struct {
		Tools []tools.Tool `json:"tools"`
	}
	if err := json.Unmarshal(b, &sent); err != nil {
		return nil, fmt.Errorf("failed to decode sent tool definitions: %w", err)
	}

	streamCtx, cancel := context.WithCancel(ctx)
	req, err := http.NewRequestWithContext(streamCtx, http.MethodPost, c.BaseAPI+"v1/responses", bytes.NewBuffer(b))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.addResponseHeaders(req, data)

	before := time.Now()
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer cancel()
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("stream request failed with status %s and failed to read body: %w", resp.Status, err)
		}
		return nil, fmt.Errorf("stream request failed with status %s, body: %s", resp.Status, string(body))
	}

	src := &sseStream{
		events: make(chan any),
		done:   make(chan struct{}),
		cancel: cancel,
		body:   resp.Body,
	}

	go func() {
		defer cancel()
		defer src.closeBody()

		eventCount := 0
		defer func() {
			c.Log.Debug(
				fmt.Sprintf("Stream finished after %s, got %d events",
					time.Since(before), eventCount),
			)
		}()

		reader := bufio.NewReader(resp.Body)

		for {
			select {
			case <-streamCtx.Done():
				src.finish(streamCtx.Err())
				return
			case <-src.done:
				return
			default:
			}

			chunk, err := reader.ReadBytes('\n')
			switch {
			case err == nil:
			case errors.Is(err, io.EOF):
				src.finish(nil)
				return
			default:
				src.finish(err)
				return
			}

			line := bytes.TrimRight(chunk, "\r\n")
			switch {
			case len(line) == 0:
				// separator between events, skip
				continue
			case bytes.HasPrefix(line, []byte("event: ")):
				// event header with event type, skip
				continue
			case bytes.HasPrefix(line, []byte("data: ")):
				// event data, handle
				chunk = line
			default:
				src.finish(fmt.Errorf("unexpected payload: %s", string(chunk)))
				return
			}

			eventCount++
			chunk = chunk[len("data: "):]
			event, err := streaming.Unmarshal(chunk)
			if err != nil {
				src.finish(fmt.Errorf("failed to unmarshal event data: %w", err))
				return
			}
			c.logStreamingCost(data, sent.Tools, processingRegion(req.URL), event)

			select {
			case src.events <- event:
			case <-streamCtx.Done():
				src.finish(streamCtx.Err())
				return
			case <-src.done:
				return
			}
		}
	}()

	return src, nil
}

// Stream sends a request with parameter "stream":true and returns a streaming iterator.
func (c *Client) Stream(ctx context.Context, req *responses.Request) (*streaming.StreamIterator, error) {
	src, err := c.streamEvents(ctx, req)
	if err != nil {
		return nil, err
	}
	return streaming.NewStreamIterator(ctx, src), nil
}
