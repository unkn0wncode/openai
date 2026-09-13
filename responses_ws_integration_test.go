// Package openai / responses_ws_integration_test.go checks live WebSocket continuation contracts.
package openai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/tools"

	"github.com/stretchr/testify/require"
)

func TestClient_Responses_WebSocket_Steering(t *testing.T) {
	t.Parallel()
	c := NewClient(integrationToken(t))
	ws := newWebSocketTestConn(t, c)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()

	stream, err := ws.Send(ctx, &responses.Request{
		Model:           models.Default,
		Input:           "Draft five brief recommendations for planning a community garden. Explain what to do first.",
		MaxOutputTokens: 768,
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
	})
	require.NoError(t, err)
	defer stream.Close()

	var originalID, continuationID, activeID string
	var continuationText strings.Builder
	var accepted, originalEnded, continuationCompleted bool
	for stream.Next() {
		switch event := stream.Event().(type) {
		case streaming.ResponseCreated:
			activeID = event.Response.ID
			require.NotEmpty(t, activeID)
			if originalID == "" {
				originalID = activeID
				require.NoError(t, ws.Steer(ctx, originalID, "Change the subject to a school library. Give three brief recommendations in at most 100 words."))
			} else {
				continuationID = activeID
				require.NotEqual(t, originalID, continuationID)
			}
		case streaming.ResponseSteerAccepted:
			require.Equal(t, originalID, event.Steer.PreviousResponseID)
			require.NotEmpty(t, event.Steer.ID)
			accepted = true
		case streaming.ResponseOutputTextDelta:
			if activeID == continuationID {
				continuationText.WriteString(event.Delta)
			}
		case streaming.ResponseIncomplete:
			require.Equal(t, originalID, event.Response.ID)
			require.NotNil(t, event.Response.IncompleteDetails)
			require.Equal(t, "steered", event.Response.IncompleteDetails.Reason)
			originalEnded = true
		case streaming.ResponseCompleted:
			if event.Response.ID == originalID {
				originalEnded = true
			} else {
				require.Equal(t, continuationID, event.Response.ID)
				continuationCompleted = true
			}
		case streaming.ResponseSteerFailed:
			require.FailNow(t, "steering failed", "%+v", event)
		case streaming.ResponseSteerPending:
			require.FailNow(t, "tool-free steering unexpectedly requires input", "%+v", event)
		case streaming.ResponseFailed:
			require.FailNow(t, "steering response failed", "%+v", event.Response.Error)
		}
	}
	require.NoError(t, stream.Err())
	require.True(t, accepted, "expected response.steer.accepted")
	require.True(t, originalEnded, "expected the original terminal response")
	require.True(t, continuationCompleted, "expected the automatic continuation to complete")
	require.NotEmpty(t, continuationText.String())
}

func TestClient_Responses_WebSocket_MultiAgentInjection(t *testing.T) {
	t.Parallel()
	c := NewClient(integrationToken(t))
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	ws, err := c.Responses.WebSocket(ctx, responses.MultiAgentBeta)
	require.NoError(t, err)
	defer ws.Close()
	require.NoError(t, c.Tools().CreateFunction(tools.FunctionCall{
		Name:         "read_probe_value",
		Description:  "Read the test value supplied by the application.",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{},"required":[],"additionalProperties":false}`),
		Strict:       true,
	}))
	// Automatic tool choice lets the response answer after accepting an injection.
	req := &responses.Request{
		Model:           models.Default,
		Input:           "Do not create subagents for this small task. Call read_probe_value exactly once, then report its result in one short sentence.",
		MaxOutputTokens: 768,
		Reasoning:       &responses.ReasoningConfig{Effort: "low"},
		MultiAgent:      &responses.MultiAgentConfig{Enabled: true, MaxConcurrentSubagents: 1},
		Tools:           []string{"read_probe_value"},
	}
	stream, err := ws.Send(ctx, req)
	require.NoError(t, err)
	defer stream.Close()

	var responseID string
	var returnedInput []output.Any
	var finalOutput []output.Any
	calls := make(map[string]struct{})
	acknowledgements := 0
	completed := false
	for stream.Next() {
		switch event := stream.Event().(type) {
		case streaming.ResponseCreated:
			responseID = event.Response.ID
			require.NotEmpty(t, responseID)
		case streaming.ResponseOutputItemDone:
			var item output.Any
			require.NoError(t, json.Unmarshal(event.Item, &item))
			if item.Type != "function_call" {
				continue
			}
			var call output.FunctionCall
			require.NoError(t, item.UnmarshalToTarget(&call))
			require.Equal(t, "read_probe_value", call.Name)
			require.NotEmpty(t, call.CallID)
			require.NotContains(t, calls, call.CallID, "a completed call must not be submitted twice")
			calls[call.CallID] = struct{}{}
			require.NoError(t, ws.Inject(ctx, responseID, []output.FunctionCallOutput{{CallID: call.CallID, Output: `{"value":"violet"}`}}))
		case streaming.ResponseInjectCreated:
			require.Equal(t, responseID, event.ResponseID)
			acknowledgements++
		case streaming.ResponseInjectFailed:
			require.Equal(t, responseID, event.ResponseID)
			require.Equal(t, "response_already_completed", event.Error.Code, event.Error.Message)
			require.NotEmpty(t, event.Input)
			returnedInput = append(returnedInput, event.Input...)
			acknowledgements++
		case streaming.ResponseCompleted:
			require.Equal(t, responseID, event.Response.ID)
			require.NoError(t, json.Unmarshal(event.Response.Output, &finalOutput))
			completed = true
		case streaming.ResponseIncomplete:
			require.FailNow(t, "injection response incomplete", "%+v", event.Response.IncompleteDetails)
		case streaming.ResponseFailed:
			require.FailNow(t, "injection response failed", "%+v", event.Response.Error)
		}
	}
	require.NoError(t, stream.Err())
	require.True(t, completed, "expected the original response to complete")
	require.NotEmpty(t, calls, "expected a function call to inject an output for")
	require.Equal(t, len(calls), acknowledgements, "every injection must receive an acknowledgement")
	require.NotEmpty(t, finalOutput, "expected the original response outputs to remain available")
	if len(returnedInput) == 0 {
		return
	}

	// Completion can win the injection race. Replay the server-returned input
	// using its original call IDs and allow the next response to answer normally.
	req = req.Clone()
	req.PreviousResponseID = responseID
	req.ToolChoice = nil
	req.Input = returnedInput
	followup, err := ws.Send(ctx, req)
	require.NoError(t, err)
	defer followup.Close()
	var recoveredText strings.Builder
	completed = false
	for followup.Next() {
		switch event := followup.Event().(type) {
		case streaming.ResponseOutputTextDelta:
			recoveredText.WriteString(event.Delta)
		case streaming.ResponseCompleted:
			require.NotEqual(t, responseID, event.Response.ID)
			completed = true
		case streaming.ResponseIncomplete:
			require.FailNow(t, "injection recovery incomplete", "%+v", event.Response.IncompleteDetails)
		case streaming.ResponseFailed:
			require.FailNow(t, "injection recovery failed", "%+v", event.Response.Error)
		}
	}
	require.NoError(t, followup.Err())
	require.True(t, completed, "expected the recovered response to complete")
	require.NotEmpty(t, recoveredText.String())
}
