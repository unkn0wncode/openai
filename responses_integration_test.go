// Package openai / responses_integration_test.go exercises current Responses API request contracts.
package openai

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/content/input"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func TestClient_Responses_CacheControls(t *testing.T) {
	t.Parallel()
	client := NewClient(integrationToken(t))
	req := &responses.Request{
		Model: models.Default, Input: "Reply with one short greeting.", MaxOutputTokens: 1024,
		ServiceTier:        responses.ServiceTierDefault,
		PromptCacheOptions: &responses.PromptCacheOptions{Mode: "explicit", TTL: "30m"},
		Reasoning:          &responses.ReasoningConfig{Mode: "standard", Context: "all_turns", Effort: "low", Summary: "auto"},
	}
	count, err := client.Responses.CountInputTokens(t.Context(), req)
	require.NoError(t, err)
	require.Positive(t, count)
	first, err := client.Responses.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, first.JoinedTexts())
	require.NotNil(t, first.Usage)
	// Explicit-only caching without breakpoints must not create paid cache writes.
	require.Zero(t, first.Usage.InputTokensDetails.CacheWriteTokens)

	req.PreviousResponseID = first.ID
	req.PromptCacheOptions.ComparisonResponseID = first.ID
	req.Input = []any{
		input.ConfigurationUpdate{Reasoning: &input.ReasoningUpdate{Effort: "medium"}},
		output.Message{Role: "user", Content: "Give a different short greeting."},
	}
	second, err := client.Responses.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, second.JoinedTexts())
	require.NotNil(t, second.PromptCacheDiagnostics)
	require.NotEmpty(t, second.PromptCacheDiagnostics.Type)
	require.NotNil(t, second.Usage)
	require.Zero(t, second.Usage.InputTokensDetails.CacheWriteTokens)
	require.NoError(t, second.CostError)
}

func TestClient_Responses_ProgrammaticTools(t *testing.T) {
	t.Parallel()
	client := NewClient(integrationToken(t))
	var called atomic.Int32
	require.NoError(t, client.Tools().RegisterTool(tools.Tool{Type: "programmatic_tool_calling"}))
	require.NoError(t, client.Tools().CreateFunction(tools.FunctionCall{
		Name: "lookup_value", Description: "Read the value required to answer the user.", ParamsSchema: tools.EmptyParamsSchema,
		AllowedCallers: []string{"programmatic"}, OutputSchema: json.RawMessage(`{"type":"object","properties":{"value":{"type":"integer"}},"required":["value"]}`),
		F: func(json.RawMessage) (string, error) {
			called.Add(1)
			return `{"value":42}`, nil
		},
	}))
	store := false
	resp, err := client.Responses.Send(&responses.Request{
		Model: models.Default, Input: "Use programmatic tool calling to call lookup_value and report the value it returns.",
		Tools: []string{"programmatic_tool_calling", "lookup_value"}, Store: &store, MaxOutputTokens: 2048,
		Reasoning: &responses.ReasoningConfig{Effort: "low"},
	})
	require.NoError(t, err)
	require.Positive(t, called.Load())
	require.NotEmpty(t, resp.JoinedTexts())
	var programSeen bool
	for _, call := range resp.Calls {
		for _, item := range call.Outputs {
			programSeen = programSeen || item.Type == "program"
		}
	}
	require.True(t, programSeen, "expected the function to be invoked through a hosted program")
}

func TestClient_Responses_AsyncToolContinuation(t *testing.T) {
	t.Parallel()
	client := NewClient(integrationToken(t))
	require.NoError(t, client.Tools().CreateFunction(tools.FunctionCall{
		Name: "start_lookup", Description: "Start an asynchronous lookup of the requested value.", ParamsSchema: tools.EmptyParamsSchema, Async: true,
		F: func(json.RawMessage) (string, error) {
			return "", errors.New("async call must be returned for application-owned execution")
		},
	}))
	req := &responses.Request{
		Model: models.Default, Input: "Start the lookup and briefly acknowledge that it is running.",
		Tools: []string{"start_lookup"}, ToolChoice: responses.ForceFunction("start_lookup"), MaxOutputTokens: 1024,
		Reasoning: &responses.ReasoningConfig{Effort: "low"},
	}
	first, err := client.Responses.Send(req)
	require.NoError(t, err)
	calls := first.FunctionCalls()
	require.Len(t, calls, 1)
	require.True(t, calls[0].Async)
	require.NotEmpty(t, calls[0].CallID)

	req.PreviousResponseID = first.ID
	req.ToolChoice = nil
	req.Input = []output.FunctionCallOutput{{CallID: calls[0].CallID, Output: `{"value":42}`}}
	final, err := client.Responses.Send(req)
	require.NoError(t, err)
	require.NotEmpty(t, final.JoinedTexts())
	require.NotEqual(t, first.ID, final.ID)
}

func TestClient_Responses_MultiAgent(t *testing.T) {
	t.Parallel()
	client := NewClient(integrationToken(t))
	client.Config().HTTPClient.Timeout = 2 * time.Minute
	resp, err := client.Responses.Send(&responses.Request{
		Model: models.Default, Input: "Ask a subagent to compute 7 times 9, then give the result in one short sentence.",
		MultiAgent: &responses.MultiAgentConfig{Enabled: true, MaxConcurrentSubagents: 2},
		Reasoning:  &responses.ReasoningConfig{Effort: "low"}, MaxOutputTokens: 2048,
	})
	require.NoError(t, err)
	var rootAnswer bool
	for _, item := range resp.ParsedOutputs {
		if message, ok := item.(output.Message); ok && message.Agent != nil && message.Agent.AgentName == "/root" {
			rootAnswer = rootAnswer || message.Phase == "final_answer"
		}
	}
	require.True(t, rootAnswer, "expected an attributed final answer from the root agent")
	require.NotEmpty(t, resp.JoinedTexts())
}

func TestClient_Responses_ImageGenerationTool(t *testing.T) {
	t.Parallel()
	client := NewClient(integrationToken(t))
	client.Config().HTTPClient.Timeout = 2 * time.Minute
	require.NoError(t, client.Tools().RegisterTool(tools.Tool{
		Type: "image_generation", Model: models.GPTImage25Flare, Quality: "low", Size: "1024x1024", OutputFormat: "png",
	}))
	resp, err := client.Responses.Send(&responses.Request{
		Model: models.Default, Input: "Generate a simple flat blue circle on a white background.", Tools: []string{"image_generation"},
		ToolChoice: responses.ForceToolChoice("image_generation", ""), MaxOutputTokens: 1024,
		Reasoning: &responses.ReasoningConfig{Effort: "low"},
	})
	require.NoError(t, err)
	var generated bool
	for _, item := range resp.ParsedOutputs {
		if image, ok := item.(output.ImageGenerationCall); ok {
			data, decodeErr := image.Data()
			require.NoError(t, decodeErr)
			require.NotEmpty(t, data)
			generated = true
		}
	}
	require.True(t, generated, "expected a decoded image-generation call")
}
