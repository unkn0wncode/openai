// Package inresponses / ws_workflow_test.go tests steering and tool injection across WebSocket responses.
package inresponses

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/responses/streaming"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func wsWorkflowClient(t *testing.T, serve func(*websocket.Conn)) *Client {
	t.Helper()
	return newWSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
		serve(conn)
		_, _, _ = conn.ReadMessage()
	})
}

func wsWorkflowRead(t *testing.T, conn *websocket.Conn, eventType string) map[string]json.RawMessage {
	t.Helper()
	var event map[string]json.RawMessage
	require.NoError(t, conn.ReadJSON(&event))
	require.JSONEq(t, `"`+eventType+`"`, string(event["type"]))
	return event
}

func wsWorkflowResponse(t *testing.T, conn *websocket.Conn, id, status string, items ...json.RawMessage) {
	t.Helper()
	body := map[string]any{
		"id": id, "object": "response", "model": "test-model", "status": status,
		"service_tier": "default", "output": items,
		"usage": map[string]int{"input_tokens": 20, "output_tokens": 10, "total_tokens": 30},
	}
	if status == "incomplete" {
		body["incomplete_details"] = map[string]string{"reason": "steered"}
	}
	if status == "created" {
		body["status"] = "in_progress"
		body["usage"] = nil
	}
	require.NoError(t, conn.WriteJSON(map[string]any{"type": "response." + status, "response": body}))
}

func wsWorkflowAccepted(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	require.NoError(t, conn.WriteJSON(map[string]any{
		"type":  "response.steer.accepted",
		"steer": map[string]string{"id": "steer_update", "previous_response_id": "resp_original"},
	}))
}

func TestWebSocketSteeringRetainsAutomaticContinuation(t *testing.T) {
	t.Parallel()

	for _, originalStatus := range []string{"incomplete", "completed"} {
		t.Run(originalStatus, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				steer := wsWorkflowRead(t, conn, "response.steer")
				require.JSONEq(t, `"resp_original"`, string(steer["previous_response_id"]))
				wsWorkflowAccepted(t, conn)
				wsWorkflowResponse(t, conn, "resp_original", originalStatus)
				wsWorkflowResponse(t, conn, "resp_continuation", "created")
				wsWorkflowResponse(t, conn, "resp_continuation", "completed")
			})
			var logs accountingLogBuffer
			client.Log = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			stream, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, stream.Next())
			require.Equal(t, "resp_original", stream.Event().(streaming.ResponseCreated).Response.ID)
			require.NoError(t, ws.Steer(ctx, "resp_original", "Use the updated requirement"))

			events, err := stream.All()
			require.NoError(t, err)
			require.Len(t, events, 4)
			require.IsType(t, streaming.ResponseSteerAccepted{}, events[0])
			if originalStatus == "incomplete" {
				require.IsType(t, streaming.ResponseIncomplete{}, events[1])
			} else {
				require.IsType(t, streaming.ResponseCompleted{}, events[1])
			}
			require.Equal(t, "resp_continuation", events[2].(streaming.ResponseCreated).Response.ID)
			require.Equal(t, "resp_continuation", events[3].(streaming.ResponseCompleted).Response.ID)

			// Each terminal response supplies its own billable usage, even though
			// both responses belong to the same iterator.
			counts := map[string]int{}
			decoder := json.NewDecoder(strings.NewReader(logs.String()))
			for {
				var record map[string]any
				err := decoder.Decode(&record)
				if errors.Is(err, io.EOF) {
					break
				}
				require.NoError(t, err)
				if id, ok := record["responseID"].(string); ok {
					counts[id]++
				}
			}
			require.Equal(t, map[string]int{"resp_original": 1, "resp_continuation": 1}, counts)
		})
	}
}

func TestWebSocketSteeringPendingAllowsToolResultContinuation(t *testing.T) {
	t.Parallel()

	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.steer")
		wsWorkflowAccepted(t, conn)
		wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":           "response.steer.pending",
			"steer":          map[string]string{"id": "steer_update", "previous_response_id": "resp_original"},
			"reason":         "waiting_for_required_input",
			"required_input": []map[string]string{{"type": "function_call_output", "call_id": "call_lookup", "name": "lookup"}},
		}))
		followup := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"resp_original"`, string(followup["previous_response_id"]))
		require.JSONEq(t, `[{"type":"function_call_output","call_id":"call_lookup","output":"found"}]`, string(followup["input"]))
		wsWorkflowResponse(t, conn, "resp_followup", "completed")
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	stream, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, stream.Next())
	require.NoError(t, ws.Steer(ctx, "resp_original", "Also include the lookup result"))
	events, err := stream.All()
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.IsType(t, streaming.ResponseSteerAccepted{}, events[0])
	require.IsType(t, streaming.ResponseCompleted{}, events[1])
	require.IsType(t, streaming.ResponseSteerPending{}, events[2])
	pending := events[2].(streaming.ResponseSteerPending)
	require.Equal(t, "steer_update", pending.Steer.ID)
	require.Equal(t, "resp_original", pending.Steer.PreviousResponseID)
	require.Len(t, pending.RequiredInput, 1)
	require.Equal(t, "call_lookup", pending.RequiredInput[0].CallID)

	req := wsTestRequest()
	req.PreviousResponseID = pending.Steer.PreviousResponseID
	req.Input = []output.FunctionCallOutput{{CallID: pending.RequiredInput[0].CallID, Output: "found"}}
	followup, err := ws.Send(ctx, req)
	require.NoError(t, err)
	events, err = followup.All()
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, "resp_followup", events[0].(streaming.ResponseCompleted).Response.ID)
}

func TestWebSocketSteeringQueuesToolResultsUntilPendingConsumed(t *testing.T) {
	t.Parallel()

	queued := make(chan struct{})
	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.steer")
		wsWorkflowAccepted(t, conn)
		wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
		select {
		case <-queued:
		case <-t.Context().Done():
			return
		}
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":           "response.steer.pending",
			"steer":          map[string]string{"id": "steer_update", "previous_response_id": "resp_original"},
			"reason":         "waiting_for_required_input",
			"required_input": []map[string]string{{"type": "function_call_output", "call_id": "call_lookup", "name": "lookup"}},
		}))
		followup := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"resp_original"`, string(followup["previous_response_id"]))
		require.JSONEq(t, `"followup-model"`, string(followup["model"]))
		// The queued request has its own settings and iterator after the
		// original iterator delivers the pending tool-result notification.
		wsWorkflowResponse(t, conn, "resp_followup", "created")
		wsWorkflowResponse(t, conn, "resp_followup", "completed")
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	original, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, original.Next())
	require.NoError(t, ws.Steer(ctx, "resp_original", "Use the lookup result"))
	require.True(t, original.Next())
	require.IsType(t, streaming.ResponseSteerAccepted{}, original.Event())
	require.True(t, original.Next())
	require.IsType(t, streaming.ResponseCompleted{}, original.Event())
	req := wsTestRequest()
	req.Model = "followup-model"
	req.PreviousResponseID = "resp_original"
	req.Input = []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}
	followup, err := ws.Send(ctx, req)
	require.NoError(t, err)
	close(queued)
	require.True(t, original.Next())
	require.IsType(t, streaming.ResponseSteerPending{}, original.Event())
	require.False(t, original.Next())
	require.NoError(t, original.Err())
	events, err := followup.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_followup", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_followup", events[1].(streaming.ResponseCompleted).Response.ID)
}

func TestWebSocketRejectedSteeringPreservesOriginalResponse(t *testing.T) {
	t.Parallel()

	for _, timing := range []string{"immediate", "accepted_then_completed"} {
		t.Run(timing, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				steer := wsWorkflowRead(t, conn, "response.steer")
				if timing == "accepted_then_completed" {
					wsWorkflowAccepted(t, conn)
					wsWorkflowResponse(t, conn, "resp_original", "completed")
				}
				require.NoError(t, conn.WriteJSON(map[string]any{
					"type":  "response.steer.failed",
					"steer": map[string]any{"id": "steer_update", "previous_response_id": "resp_original", "input": steer["input"]},
					"error": map[string]string{"code": "steering_not_supported", "message": "Steering is unavailable for this request."},
				}))
				if timing == "immediate" {
					wsWorkflowResponse(t, conn, "resp_original", "completed")
				}
			})
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			stream, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, stream.Next())
			require.NoError(t, ws.Steer(ctx, "resp_original", "Remember this update"))
			events, err := stream.All()
			require.NoError(t, err)
			failureIndex := 0
			if timing == "accepted_then_completed" {
				require.Len(t, events, 3)
				require.IsType(t, streaming.ResponseSteerAccepted{}, events[0])
				require.Equal(t, "steer_update", events[0].(streaming.ResponseSteerAccepted).Steer.ID)
				failureIndex = 2
			} else {
				require.Len(t, events, 2)
			}
			require.IsType(t, streaming.ResponseSteerFailed{}, events[failureIndex])
			failure := events[failureIndex].(streaming.ResponseSteerFailed)
			require.Equal(t, "steer_update", failure.Steer.ID)
			require.Equal(t, "resp_original", failure.Steer.PreviousResponseID)
			require.Equal(t, "steering_not_supported", failure.Error.Code)
			require.JSONEq(t, `"Remember this update"`, string(failure.Steer.Input))
			require.Equal(t, "resp_original", events[1].(streaming.ResponseCompleted).Response.ID)
		})
	}
}

func TestWebSocketInjectionAcknowledgementSurvivesResponseCompletion(t *testing.T) {
	t.Parallel()

	for _, outcome := range []string{"created", "failed"} {
		t.Run(outcome, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				injection := wsWorkflowRead(t, conn, "response.inject")
				require.JSONEq(t, `"resp_original"`, string(injection["response_id"]))
				wsWorkflowResponse(t, conn, "resp_original", "completed")
				ack := map[string]any{"type": "response.inject." + outcome, "response_id": "resp_original"}
				if outcome == "failed" {
					ack["input"] = injection["input"]
					ack["error"] = map[string]string{"code": "response_already_completed", "message": "The response completed before injection."}
				}
				require.NoError(t, conn.WriteJSON(ack))
				if outcome == "failed" {
					followup := wsWorkflowRead(t, conn, "response.create")
					require.JSONEq(t, string(injection["input"]), string(followup["input"]))
					require.JSONEq(t, `"resp_original"`, string(followup["previous_response_id"]))
					wsWorkflowResponse(t, conn, "resp_recovered", "completed")
				}
			})
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			stream, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, stream.Next())
			require.NoError(t, ws.Inject(ctx, "resp_original", []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}))
			events, err := stream.All()
			require.NoError(t, err)
			require.Len(t, events, 2)
			require.IsType(t, streaming.ResponseCompleted{}, events[0])
			if outcome == "created" {
				require.IsType(t, streaming.ResponseInjectCreated{}, events[1])
				require.Equal(t, "resp_original", events[1].(streaming.ResponseInjectCreated).ResponseID)
				return
			}
			require.IsType(t, streaming.ResponseInjectFailed{}, events[1])
			failure := events[1].(streaming.ResponseInjectFailed)
			require.Equal(t, "response_already_completed", failure.Error.Code)
			req := wsTestRequest()
			req.PreviousResponseID = failure.ResponseID
			req.Input = failure.Input
			followup, err := ws.Send(ctx, req)
			require.NoError(t, err)
			events, err = followup.All()
			require.NoError(t, err)
			require.Len(t, events, 1)
			require.Equal(t, "resp_recovered", events[0].(streaming.ResponseCompleted).Response.ID)
		})
	}
}

func TestWebSocketInjectionCanFollowBufferedResponseCompletion(t *testing.T) {
	t.Parallel()

	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "response.output_item.done", "output_index": 0,
			"item": json.RawMessage(accountingFunctionCall),
		}))
		wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
		injection := wsWorkflowRead(t, conn, "response.inject")
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "response.inject.failed", "response_id": "resp_original", "input": injection["input"],
			"error": map[string]string{"code": "response_already_completed", "message": "The response completed before injection."},
		}))
	})
	var logs accountingLogBuffer
	client.Log = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	stream, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, stream.Next())
	require.IsType(t, streaming.ResponseCreated{}, stream.Event())
	require.True(t, stream.Next())
	require.IsType(t, streaming.ResponseOutputItemDone{}, stream.Event())
	var call output.FunctionCall
	require.NoError(t, json.Unmarshal(stream.Event().(streaming.ResponseOutputItemDone).Item, &call))
	// Usage logging confirms that the reader has processed completion while
	// the application is still answering the preceding function call.
	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), `"responseID":"resp_original"`)
	}, time.Second, time.Millisecond)
	require.NoError(t, ws.Inject(ctx, "resp_original", []output.FunctionCallOutput{{CallID: call.CallID, Output: "found"}}))

	events, err := stream.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_original", events[0].(streaming.ResponseCompleted).Response.ID)
	require.IsType(t, streaming.ResponseInjectFailed{}, events[1])
	failure := events[1].(streaming.ResponseInjectFailed)
	require.Equal(t, "response_already_completed", failure.Error.Code)
	require.Equal(t, "resp_original", failure.ResponseID)
	require.Len(t, failure.Input, 1)
	var recovered output.FunctionCallOutput
	require.NoError(t, failure.Input[0].UnmarshalToTarget(&recovered))
	require.Equal(t, call.CallID, recovered.CallID)
	require.Equal(t, "found", recovered.Output)
}

func TestWebSocketLateInjectionAcknowledgementStaysWithOriginalResponse(t *testing.T) {
	t.Parallel()

	queued := make(chan struct{})
	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.inject")
		wsWorkflowResponse(t, conn, "resp_original", "completed")
		select {
		case <-queued:
		case <-t.Context().Done():
			return
		}
		nextRequest := make(chan struct{})
		go func() {
			defer close(nextRequest)
			wsWorkflowRead(t, conn, "response.create")
		}()
		// Allow an already-sent request to reach the server. If it arrives
		// before the acknowledgement, its events reproduce the backpressure
		// that must not block the original iterator waiting for that reply.
		nextStarted := false
		select {
		case <-nextRequest:
			nextStarted = true
			wsWorkflowResponse(t, conn, "resp_next", "created")
			wsWorkflowResponse(t, conn, "resp_next", "in_progress")
		case <-time.After(50 * time.Millisecond):
		}
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "response.inject.created", "response_id": "resp_original",
		}))
		if !nextStarted {
			<-nextRequest
			wsWorkflowResponse(t, conn, "resp_next", "created")
			wsWorkflowResponse(t, conn, "resp_next", "in_progress")
		}
		wsWorkflowResponse(t, conn, "resp_next", "completed")
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	original, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, original.Next())
	require.Equal(t, "resp_original", original.Event().(streaming.ResponseCreated).Response.ID)
	require.NoError(t, ws.Inject(ctx, "resp_original", []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}))
	next, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	close(queued)

	events, err := original.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_original", events[0].(streaming.ResponseCompleted).Response.ID)
	require.Equal(t, "resp_original", events[1].(streaming.ResponseInjectCreated).ResponseID)
	events, err = next.All()
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, "resp_next", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_next", events[1].(streaming.ResponseInProgress).Response.ID)
	require.Equal(t, "resp_next", events[2].(streaming.ResponseCompleted).Response.ID)
}

func TestWebSocketRejectedRequestAfterAbandonedInjectionAllowsNextRequest(t *testing.T) {
	t.Parallel()

	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.inject")
		wsWorkflowResponse(t, conn, "resp_original", "completed")
		rejected := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"rejected-model"`, string(rejected["model"]))
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "error", "status": http.StatusBadRequest,
			"error": map[string]string{"type": "invalid_request_error", "code": "model_not_found", "message": "Request rejected"},
		}))
		next := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"next-model"`, string(next["model"]))
		// The abandoned iterator still owns this acknowledgement after the
		// rejected request has ended and the next request has been sent.
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "response.inject.created", "response_id": "resp_original",
		}))
		wsWorkflowResponse(t, conn, "resp_next", "created")
		wsWorkflowResponse(t, conn, "resp_next", "completed")
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	first, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, first.Next())
	require.IsType(t, streaming.ResponseCreated{}, first.Event())
	require.NoError(t, ws.Inject(ctx, "resp_original", []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}))
	require.True(t, first.Next())
	require.IsType(t, streaming.ResponseCompleted{}, first.Event())
	first.Close()

	req := wsTestRequest()
	req.Model = "rejected-model"
	second, err := ws.Send(ctx, req)
	require.NoError(t, err)
	req = wsTestRequest()
	req.Model = "next-model"
	third, err := ws.Send(ctx, req)
	require.NoError(t, err)

	events, err := second.All()
	require.Empty(t, events)
	var streamErr *streaming.StreamError
	require.ErrorAs(t, err, &streamErr)
	require.IsType(t, streaming.WSError{}, streamErr.Event)
	require.Equal(t, "model_not_found", streamErr.Event.(streaming.WSError).Error.Code)
	events, err = third.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_next", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_next", events[1].(streaming.ResponseCompleted).Response.ID)
}

func TestWebSocketAbandonedRequestRejectionPreservesNextResponse(t *testing.T) {
	t.Parallel()

	firstSent := make(chan struct{})
	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		close(firstSent)
		next := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"next-model"`, string(next["model"]))
		// The first request was abandoned before the server accepted it.
		// Its rejection must not end the already-sent second request.
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "error", "status": http.StatusBadRequest,
			"error": map[string]string{"type": "invalid_request_error", "code": "model_not_found", "message": "First request rejected"},
		}))
		wsWorkflowResponse(t, conn, "resp_next", "created")
		wsWorkflowResponse(t, conn, "resp_next", "completed")
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	first, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	select {
	case <-firstSent:
	case <-ctx.Done():
		t.Fatal("first request was not sent")
	}
	first.Close()
	req := wsTestRequest()
	req.Model = "next-model"
	second, err := ws.Send(ctx, req)
	require.NoError(t, err)
	events, err := second.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_next", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_next", events[1].(streaming.ResponseCompleted).Response.ID)
}

func TestWebSocketAbandonedSteeringDrainsBeforeNextRequest(t *testing.T) {
	t.Parallel()

	for _, abandon := range []string{"close", "cancel"} {
		t.Run(abandon, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				wsWorkflowRead(t, conn, "response.steer")
				wsWorkflowAccepted(t, conn)
				// The next request is queued before the first response and its
				// automatic continuation finish; neither belongs to its iterator.
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "incomplete")
				wsWorkflowResponse(t, conn, "resp_continuation", "created")
				wsWorkflowResponse(t, conn, "resp_continuation", "completed")
				wsWorkflowResponse(t, conn, "resp_next", "created")
				wsWorkflowResponse(t, conn, "resp_next", "completed")
			})
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			firstCtx, cancelFirst := context.WithCancel(ctx)
			defer cancelFirst()
			first, err := ws.Send(firstCtx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, first.Next())
			require.NoError(t, ws.Steer(ctx, "resp_original", "Use the update"))
			require.True(t, first.Next())
			require.IsType(t, streaming.ResponseSteerAccepted{}, first.Event())
			if abandon == "cancel" {
				cancelFirst()
				require.False(t, first.Next())
				require.ErrorIs(t, first.Err(), context.Canceled)
			} else {
				first.Close()
			}
			second, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			events, err := second.All()
			require.NoError(t, err)
			require.Len(t, events, 2)
			require.Equal(t, "resp_next", events[0].(streaming.ResponseCreated).Response.ID)
			require.Equal(t, "resp_next", events[1].(streaming.ResponseCompleted).Response.ID)
		})
	}
}

func TestWebSocketSteeringHandsOffCoalescedInputToExplicitContinuation(t *testing.T) {
	t.Parallel()

	for _, pending := range []bool{true, false} {
		name := "without_pending"
		if pending {
			name = "with_pending"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				for _, id := range []string{"steer_first", "steer_second"} {
					wsWorkflowRead(t, conn, "response.steer")
					require.NoError(t, conn.WriteJSON(map[string]any{
						"type":  "response.steer.accepted",
						"steer": map[string]string{"id": id, "previous_response_id": "resp_original"},
					}))
				}
				wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
				if pending {
					for _, id := range []string{"steer_first", "steer_second"} {
						require.NoError(t, conn.WriteJSON(map[string]any{
							"type":           "response.steer.pending",
							"steer":          map[string]string{"id": id, "previous_response_id": "resp_original"},
							"reason":         "waiting_for_required_input",
							"required_input": []map[string]string{{"type": "function_call_output", "call_id": "call_lookup", "name": "lookup"}},
						}))
					}
				}
				followup := wsWorkflowRead(t, conn, "response.create")
				require.JSONEq(t, `"resp_original"`, string(followup["previous_response_id"]))
				require.JSONEq(t, `"followup-model"`, string(followup["model"]))
				require.JSONEq(t, `[{"type":"function_call_output","call_id":"call_lookup","output":"found"}]`, string(followup["input"]))
				require.NoError(t, conn.WriteJSON(map[string]any{
					"type":     "response.created",
					"response": map[string]string{"id": "resp_followup", "previous_response_id": "resp_original", "status": "in_progress"},
				}))
				wsWorkflowResponse(t, conn, "resp_followup", "completed")
			})
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			original, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, original.Next())
			require.NoError(t, ws.Steer(ctx, "resp_original", "First update"))
			require.NoError(t, ws.Steer(ctx, "resp_original", "Second update"))
			for _, id := range []string{"steer_first", "steer_second"} {
				require.True(t, original.Next())
				require.Equal(t, id, original.Event().(streaming.ResponseSteerAccepted).Steer.ID)
			}
			require.True(t, original.Next())
			require.IsType(t, streaming.ResponseCompleted{}, original.Event())
			if pending {
				events, err := original.All()
				require.NoError(t, err)
				require.Len(t, events, 2)
				require.Equal(t, "steer_first", events[0].(streaming.ResponseSteerPending).Steer.ID)
				require.Equal(t, "steer_second", events[1].(streaming.ResponseSteerPending).Steer.ID)
			} else {
				// Returning tool results before pending is allowed; both accepted
				// submissions are committed by this one explicit continuation.
				original.Close()
			}
			req := wsTestRequest()
			req.Model = "followup-model"
			req.PreviousResponseID = "resp_original"
			req.Input = []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}
			followup, err := ws.Send(ctx, req)
			require.NoError(t, err)
			events, err := followup.All()
			require.NoError(t, err)
			require.Len(t, events, 2)
			require.Equal(t, "resp_followup", events[0].(streaming.ResponseCreated).Response.ID)
			require.Equal(t, "resp_followup", events[1].(streaming.ResponseCompleted).Response.ID)
		})
	}
}

func TestWebSocketFailedExplicitContinuationReturnsPendingSteering(t *testing.T) {
	t.Parallel()

	for _, errorFirst := range []bool{true, false} {
		name := "steering_failure_first"
		if errorFirst {
			name = "request_error_first"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			client := wsWorkflowClient(t, func(conn *websocket.Conn) {
				wsWorkflowRead(t, conn, "response.create")
				wsWorkflowResponse(t, conn, "resp_original", "created")
				wsWorkflowRead(t, conn, "response.steer")
				wsWorkflowAccepted(t, conn)
				wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
				require.NoError(t, conn.WriteJSON(map[string]any{
					"type":           "response.steer.pending",
					"steer":          map[string]string{"id": "steer_update", "previous_response_id": "resp_original"},
					"reason":         "waiting_for_required_input",
					"required_input": []map[string]string{{"type": "function_call_output", "call_id": "call_lookup", "name": "lookup"}},
				}))
				wsWorkflowRead(t, conn, "response.create")
				requestError := map[string]any{
					"type": "error", "status": http.StatusBadRequest,
					"error": map[string]string{"type": "invalid_request_error", "code": "invalid_value", "message": "Cannot create the continuation"},
				}
				if errorFirst {
					require.NoError(t, conn.WriteJSON(requestError))
				}
				require.NoError(t, conn.WriteJSON(map[string]any{
					"type":  "response.steer.failed",
					"steer": map[string]any{"id": "steer_update", "previous_response_id": "resp_original", "input": "Apply my correction"},
					"error": map[string]string{"type": "invalid_request_error", "code": "successor_creation_failed", "message": "Could not create the successor"},
				}))
				if !errorFirst {
					require.NoError(t, conn.WriteJSON(requestError))
				}
			})
			ws := openTestWebSocket(t, client)
			defer ws.Close()
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			original, err := ws.Send(ctx, wsTestRequest())
			require.NoError(t, err)
			require.True(t, original.Next())
			require.NoError(t, ws.Steer(ctx, "resp_original", "Apply my correction"))
			events, err := original.All()
			require.NoError(t, err)
			require.Len(t, events, 3)
			require.IsType(t, streaming.ResponseSteerPending{}, events[2])
			req := wsTestRequest()
			req.PreviousResponseID = "resp_original"
			req.Input = []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}
			followup, err := ws.Send(ctx, req)
			require.NoError(t, err)
			events, err = followup.All()
			require.Len(t, events, 1)
			failure := events[0].(streaming.ResponseSteerFailed)
			require.Equal(t, "steer_update", failure.Steer.ID)
			require.Equal(t, "resp_original", failure.Steer.PreviousResponseID)
			require.JSONEq(t, `"Apply my correction"`, string(failure.Steer.Input))
			var streamErr *streaming.StreamError
			require.ErrorAs(t, err, &streamErr)
			require.Equal(t, "invalid_value", streamErr.Event.(streaming.WSError).Error.Code)
		})
	}
}

func TestWebSocketAutomaticContinuationPreservesUnacknowledgedSteering(t *testing.T) {
	t.Parallel()

	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.steer")
		wsWorkflowAccepted(t, conn)
		wsWorkflowResponse(t, conn, "resp_original", "completed")
		wsWorkflowRead(t, conn, "response.steer")
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":     "response.created",
			"response": map[string]string{"id": "resp_successor", "previous_response_id": "resp_original", "status": "in_progress"},
		}))
		wsWorkflowResponse(t, conn, "resp_successor", "completed")
		// The successor commits the first accepted input. The second submission
		// can still be rejected after that response has completed.
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":  "response.steer.failed",
			"steer": map[string]any{"previous_response_id": "resp_original", "input": "Second correction"},
			"error": map[string]string{"type": "invalid_request_error", "code": "response_not_active", "message": "The original response is no longer accepting steering"},
		}))
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	original, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, original.Next())
	require.NoError(t, ws.Steer(ctx, "resp_original", "First correction"))
	require.True(t, original.Next())
	require.IsType(t, streaming.ResponseSteerAccepted{}, original.Event())
	require.True(t, original.Next())
	require.IsType(t, streaming.ResponseCompleted{}, original.Event())
	require.NoError(t, ws.Steer(ctx, "resp_original", "Second correction"))
	events, err := original.All()
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.Equal(t, "resp_successor", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_successor", events[1].(streaming.ResponseCompleted).Response.ID)
	failure := events[2].(streaming.ResponseSteerFailed)
	require.Empty(t, failure.Steer.ID)
	require.Equal(t, "response_not_active", failure.Error.Code)
	require.JSONEq(t, `"Second correction"`, string(failure.Steer.Input))
}

func TestWebSocketPendingSteeringSurvivesIndependentRequest(t *testing.T) {
	t.Parallel()

	client := wsWorkflowClient(t, func(conn *websocket.Conn) {
		wsWorkflowRead(t, conn, "response.create")
		wsWorkflowResponse(t, conn, "resp_original", "created")
		wsWorkflowRead(t, conn, "response.steer")
		wsWorkflowAccepted(t, conn)
		wsWorkflowResponse(t, conn, "resp_original", "completed", json.RawMessage(accountingFunctionCall))
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":           "response.steer.pending",
			"steer":          map[string]string{"id": "steer_update", "previous_response_id": "resp_original"},
			"reason":         "waiting_for_required_input",
			"required_input": []map[string]string{{"type": "function_call_output", "call_id": "call_lookup", "name": "lookup"}},
		}))
		independent := wsWorkflowRead(t, conn, "response.create")
		require.Empty(t, independent["previous_response_id"])
		wsWorkflowResponse(t, conn, "resp_independent", "created")
		wsWorkflowResponse(t, conn, "resp_independent", "completed")

		followup := wsWorkflowRead(t, conn, "response.create")
		require.JSONEq(t, `"resp_original"`, string(followup["previous_response_id"]))
		require.JSONEq(t, `[{"type":"function_call_output","call_id":"call_lookup","output":"found"}]`, string(followup["input"]))
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type": "error", "status": http.StatusBadRequest,
			"error": map[string]string{"type": "invalid_request_error", "code": "invalid_value", "message": "Cannot create the continuation"},
		}))
		require.NoError(t, conn.WriteJSON(map[string]any{
			"type":  "response.steer.failed",
			"steer": map[string]any{"id": "steer_update", "previous_response_id": "resp_original", "input": "Apply my correction"},
			"error": map[string]string{"type": "invalid_request_error", "code": "successor_creation_failed", "message": "Could not create the successor"},
		}))
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	original, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	require.True(t, original.Next())
	require.NoError(t, ws.Steer(ctx, "resp_original", "Apply my correction"))
	events, err := original.All()
	require.NoError(t, err)
	require.Len(t, events, 3)
	require.IsType(t, streaming.ResponseSteerPending{}, events[2])

	independent, err := ws.Send(ctx, wsTestRequest())
	require.NoError(t, err)
	events, err = independent.All()
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "resp_independent", events[0].(streaming.ResponseCreated).Response.ID)
	require.Equal(t, "resp_independent", events[1].(streaming.ResponseCompleted).Response.ID)

	// The independent response leaves the correction queued for its own parent.
	req := wsTestRequest()
	req.PreviousResponseID = "resp_original"
	req.Input = []output.FunctionCallOutput{{CallID: "call_lookup", Output: "found"}}
	followup, err := ws.Send(ctx, req)
	require.NoError(t, err)
	events, err = followup.All()
	require.Len(t, events, 1)
	failure := events[0].(streaming.ResponseSteerFailed)
	require.Equal(t, "steer_update", failure.Steer.ID)
	require.Equal(t, "resp_original", failure.Steer.PreviousResponseID)
	require.JSONEq(t, `"Apply my correction"`, string(failure.Steer.Input))
	var streamErr *streaming.StreamError
	require.ErrorAs(t, err, &streamErr)
	require.Equal(t, "invalid_value", streamErr.Event.(streaming.WSError).Error.Code)
}
