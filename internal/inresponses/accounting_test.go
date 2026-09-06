// Package inresponses / accounting_test.go tests usage accounting and cost logging across response transports.
package inresponses

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/tools"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

const accountingFunctionCall = `{"type":"function_call","id":"fc_lookup","call_id":"call_lookup","name":"lookup","arguments":"{}","status":"completed"}`

func newAccountingClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	client := newWSTestClient(t, handler)
	target, err := url.Parse(client.BaseAPI)
	require.NoError(t, err)
	client.BaseAPI = "http://api.openai.com/"
	base := client.HTTPClient.Transport
	client.HTTPClient.Transport = accountingRoundTripper(func(req *http.Request) (*http.Response, error) {
		local := req.Clone(req.Context())
		local.URL.Scheme, local.URL.Host = target.Scheme, target.Host
		return base.RoundTrip(local)
	})
	client.WebSocketDialer = &websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, target.Host)
		},
	}
	return client
}

func accountingResponse(id, status, tier string, inputTokens, outputTokens int, outputs ...json.RawMessage) map[string]any {
	return map[string]any{
		"id": id, "object": "response", "model": "gpt-5.4",
		"status": status, "service_tier": tier, "output": outputs,
		"usage": map[string]any{
			"input_tokens": inputTokens, "output_tokens": outputTokens,
			"total_tokens": inputTokens + outputTokens,
		},
	}
}

func accountingMessage(text string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"type":"message","role":"assistant","status":"completed","content":[{"type":"output_text","text":%q}]}`, text))
}

func registerAccountingFunction(t *testing.T, client *Client, execute func(json.RawMessage) (string, error)) {
	t.Helper()
	require.NoError(t, client.Tools.CreateFunction(tools.FunctionCall{
		Name: "lookup", Description: "Look up a value", ParamsSchema: tools.EmptyParamsSchema, F: execute,
	}))
}

func TestSendPreservesEachCallBeforeCombiningOutputs(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PreviousResponseID string          `json:"previous_response_id"`
			Input              json.RawMessage `json:"input"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		var body map[string]any
		switch requests.Add(1) {
		case 1:
			require.Empty(t, req.PreviousResponseID)
			body = accountingResponse("resp_first", "completed", "default", 150000, 100,
				accountingMessage("Looking it up"), json.RawMessage(accountingFunctionCall))
		case 2:
			require.Equal(t, "resp_first", req.PreviousResponseID)
			require.JSONEq(t, `[{"type":"function_call_output","call_id":"call_lookup","output":"found"}]`, string(req.Input))
			body = accountingResponse("resp_final", "completed", "flex", 150000, 100, accountingMessage("The answer"))
		default:
			t.Error("unexpected additional request")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) { return "found", nil })
	req := &responses.Request{Model: "gpt-5.4", Input: "Look up a value", Tools: []string{"lookup"}, ServiceTier: "default"}

	resp, err := client.Send(req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, "resp_final", resp.ID)
	require.Equal(t, "gpt-5.4", resp.Model)
	require.Equal(t, "flex", resp.ServiceTier)
	require.Equal(t, "completed", resp.Status)
	require.Equal(t, "global", resp.ProcessingRegion)
	require.Equal(t, []string{"Looking it up", "The answer"}, resp.Texts())
	require.NotNil(t, resp.Usage)
	require.Equal(t, 150000, resp.Usage.InputTokens)
	require.False(t, resp.BillingIncomplete)
	require.Len(t, resp.Calls, 2)
	require.Equal(t, "resp_first", resp.Calls[0].ID)
	require.Equal(t, "default", resp.Calls[0].ServiceTier)
	require.Len(t, resp.Calls[0].Outputs, 2)
	require.Equal(t, "function_call", resp.Calls[0].Outputs[1].Type)
	require.Equal(t, "resp_final", resp.Calls[1].ID)
	require.Equal(t, "flex", resp.Calls[1].ServiceTier)
	require.Len(t, resp.Calls[1].Outputs, 1)
	for _, call := range resp.Calls {
		require.NotNil(t, call.Usage)
		require.Equal(t, 150000, call.Usage.InputTokens)
		require.Equal(t, "global", call.ProcessingRegion)
		require.Empty(t, call.Calls)
		require.Empty(t, call.ParsedOutputs)
		require.Len(t, call.Tools, 1)
		require.Equal(t, "lookup", call.Tools[0].Name)
	}

	// Each request stays below the context threshold; the second used Flex.
	cost, err := req.EstimateCost(resp)
	require.NoError(t, err)
	require.InDelta(t, 0.56475, cost, 1e-12)
	require.InDelta(t, 0.3765, resp.Calls[0].EstimatedCost, 1e-12)
	require.InDelta(t, 0.18825, resp.Calls[1].EstimatedCost, 1e-12)
}

func TestSendLocalToolFailurePreservesObservedResponse(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		require.NoError(t, json.NewEncoder(w).Encode(accountingResponse("resp_observed", "completed", "default", 1000, 100,
			json.RawMessage(accountingFunctionCall))))
	})
	toolErr := errors.New("lookup failed locally")
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) { return "", toolErr })

	resp, err := client.Send(&responses.Request{Model: "gpt-5.4", Input: "Look up a value", Tools: []string{"lookup"}})

	require.ErrorIs(t, err, toolErr)
	require.NotNil(t, resp)
	require.EqualValues(t, 1, requests.Load())
	require.Equal(t, "resp_observed", resp.ID)
	require.NotNil(t, resp.Usage)
	require.Equal(t, 1000, resp.Usage.InputTokens)
	require.False(t, resp.BillingIncomplete)
	require.Len(t, resp.Calls, 1)
	require.Equal(t, "function_call", resp.Calls[0].Outputs[0].Type)
}

func TestSendKeepsDistinctEstimationErrorsSeparateFromExecution(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		var body map[string]any
		if requests.Add(1) == 1 {
			body = accountingResponse("resp_first", "completed", "future_first_tier", 1000, 100,
				json.RawMessage(accountingFunctionCall))
		} else {
			body = accountingResponse("resp_final", "completed", "future_second_tier", 1000, 100,
				accountingMessage("The answer"))
		}
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) { return "found", nil })
	req := &responses.Request{Model: "gpt-5.4", Input: "Look up a value", Tools: []string{"lookup"}}

	resp, executionErr := client.Send(req)

	require.NoError(t, executionErr)
	require.Equal(t, "The answer", resp.FirstText())
	require.Len(t, resp.Calls, 2)
	require.ErrorContains(t, resp.Calls[0].CostError, "future_first_tier")
	require.ErrorContains(t, resp.Calls[1].CostError, "future_second_tier")
	amount, costErr := req.EstimateCost(resp)
	require.Zero(t, amount)
	require.ErrorIs(t, costErr, resp.Calls[0].CostError)
	require.ErrorIs(t, costErr, resp.Calls[1].CostError)
	require.ErrorContains(t, costErr, "resp_first")
	require.ErrorContains(t, costErr, "resp_final")
	require.False(t, resp.BillingIncomplete)
}

type accountingRoundTripper func(*http.Request) (*http.Response, error)

func (f accountingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSendFollowUpTransportFailurePreservesPriorAccounting(t *testing.T) {
	t.Parallel()

	client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(accountingResponse("resp_observed", "completed", "default", 1000, 100,
			accountingMessage("Looking it up"), json.RawMessage(accountingFunctionCall))))
	})
	registerAccountingFunction(t, client, func(json.RawMessage) (string, error) { return "found", nil })
	transportErr := errors.New("connection lost after sending follow-up")
	base := client.HTTPClient.Transport
	var requests atomic.Int32
	client.HTTPClient.Transport = accountingRoundTripper(func(req *http.Request) (*http.Response, error) {
		if requests.Add(1) == 2 {
			return nil, transportErr
		}
		return base.RoundTrip(req)
	})
	req := &responses.Request{Model: "gpt-5.4", Input: "Look up a value", Tools: []string{"lookup"}}

	resp, err := client.Send(req)

	require.ErrorIs(t, err, transportErr)
	require.NotNil(t, resp)
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, "resp_observed", resp.ID)
	require.True(t, resp.BillingIncomplete)
	require.Len(t, resp.Calls, 1)
	require.Equal(t, "resp_observed", resp.Calls[0].ID)
	require.NotNil(t, resp.Calls[0].Usage)
	require.Equal(t, 1000, resp.Calls[0].Usage.InputTokens)
	cost, costErr := req.EstimateCost(resp)
	require.Error(t, costErr)
	require.InDelta(t, 0.004, cost, 1e-12)
	require.NotErrorIs(t, costErr, transportErr)
}

func TestSendLocalValidationFailureHasNoResponse(t *testing.T) {
	t.Parallel()

	client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid request reached the server")
		w.WriteHeader(http.StatusInternalServerError)
	})
	for _, req := range []*responses.Request{nil, {Model: "gpt-5.4"}, {Model: "gpt-5.4", Input: "hi", Tools: []string{"unregistered"}}} {
		resp, err := client.Send(req)
		require.Error(t, err)
		require.Nil(t, resp)
	}
}

func TestSendDecodeAndParseFailuresPreserveAvailableAccounting(t *testing.T) {
	t.Parallel()

	t.Run("undecodable response", func(t *testing.T) {
		t.Parallel()
		client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, `{"id":"resp_broken","usage":`)
		})
		resp, err := client.Send(&responses.Request{Model: "gpt-5.4", Input: "hi"})
		require.ErrorContains(t, err, "failed to decode response")
		require.NotNil(t, resp)
		require.True(t, resp.BillingIncomplete)
		require.Empty(t, resp.Calls)
		require.Nil(t, resp.Usage)
		require.Error(t, resp.CostError)
	})

	t.Run("unparseable output", func(t *testing.T) {
		t.Parallel()
		client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
			body := accountingResponse("resp_observed", "completed", "default", 1000, 100,
				json.RawMessage(`{"type":"message","role":"assistant","status":"completed","content":42}`))
			require.NoError(t, json.NewEncoder(w).Encode(body))
		})
		resp, err := client.Send(&responses.Request{Model: "gpt-5.4", Input: "hi"})
		require.ErrorContains(t, err, "failed to parse output")
		require.NotNil(t, resp)
		require.False(t, resp.BillingIncomplete)
		require.Equal(t, "resp_observed", resp.ID)
		require.Len(t, resp.Calls, 1)
		require.Len(t, resp.Calls[0].Outputs, 1)
		require.NotNil(t, resp.Calls[0].Usage)
		require.Equal(t, 1100, resp.Calls[0].Usage.TotalTokens)
		require.NoError(t, resp.CostError)
		require.InDelta(t, 0.004, resp.EstimatedCost, 1e-12)
	})
}

func TestBackgroundSendRetainsPendingMetadataWithoutUsage(t *testing.T) {
	t.Parallel()

	for _, statusCode := range []int{http.StatusOK, http.StatusAccepted} {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			t.Parallel()
			client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
				body := accountingResponse("resp_pending", "queued", "flex", 0, 0)
				body["usage"] = nil
				w.WriteHeader(statusCode)
				require.NoError(t, json.NewEncoder(w).Encode(body))
			})
			resp, err := client.Send(&responses.Request{Model: "gpt-5.4", Input: "hi", Background: true})
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, "resp_pending", resp.ID)
			require.Equal(t, "gpt-5.4", resp.Model)
			require.Equal(t, "flex", resp.ServiceTier)
			require.Equal(t, "queued", resp.Status)
			require.Nil(t, resp.Usage)
		})
	}
}

func TestPollWaitsForPendingStatesAndPreservesCompletion(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v1/responses/resp_poll", r.URL.Path)
		statuses := []string{"queued", "in_progress", "completed"}
		call := int(requests.Add(1))
		if call > len(statuses) {
			t.Error("poll continued after completion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		body := accountingResponse("resp_poll", statuses[call-1], "flex", 1000, 100, accountingMessage("Finished"))
		if call < len(statuses) {
			body["output"] = nil
			body["usage"] = nil
		}
		require.NoError(t, json.NewEncoder(w).Encode(body))
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	resp, err := client.Poll(ctx, "resp_poll", 0)

	require.NoError(t, err)
	require.EqualValues(t, 3, requests.Load())
	require.NotNil(t, resp)
	require.Equal(t, "completed", resp.Status)
	require.Equal(t, "gpt-5.4", resp.Model)
	require.Equal(t, "flex", resp.ServiceTier)
	require.Equal(t, "Finished", resp.FirstText())
	require.NotNil(t, resp.Usage)
	require.Equal(t, 1100, resp.Usage.TotalTokens)
}

func TestPollPreservesPendingEstimatesWhenInterrupted(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"queued", "in_progress"} {
		for _, stop := range []string{"cancellation", "fetch error"} {
			t.Run(status+"/"+stop, func(t *testing.T) {
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				stopErr := errors.New("fetch failed")
				if stop == "cancellation" {
					stopErr = context.Canceled
				}
				body := accountingResponse("resp_pending", status, "default", 1000, 100)
				if status == "queued" {
					body["usage"] = nil
				}
				payload, err := json.Marshal(body)
				require.NoError(t, err)
				client := NewClient(openai.NewConfig("test-token"))
				var requests int
				client.HTTPClient.Transport = accountingRoundTripper(func(*http.Request) (*http.Response, error) {
					requests++
					if requests > 1 {
						return nil, stopErr
					}
					responseBody := io.NopCloser(bytes.NewReader(payload))
					if stop == "cancellation" {
						responseBody = &closeRecorderBody{ReadCloser: responseBody, close: cancel}
					}
					return &http.Response{StatusCode: http.StatusOK, Body: responseBody}, nil
				})
				var logs accountingLogBuffer
				client.Log = slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))

				resp, err := client.Poll(ctx, "resp_pending", 0)

				require.ErrorIs(t, err, stopErr)
				require.NotNil(t, resp)
				require.Equal(t, status, resp.Status)
				require.Len(t, resp.Calls, 1)
				require.ErrorContains(t, resp.CostError, status)
				require.ErrorContains(t, resp.Calls[0].CostError, status)
				require.ErrorIs(t, resp.CostError, resp.Calls[0].CostError)
				require.NotErrorIs(t, resp.CostError, stopErr)
				require.Equal(t, resp.Calls[0].EstimatedCost, resp.EstimatedCost)
				if status == "queued" {
					require.ErrorContains(t, resp.CostError, "usage")
					require.Zero(t, resp.EstimatedCost)
				} else {
					require.Positive(t, resp.EstimatedCost)
				}
				require.Empty(t, logs.String(), "pending estimates should not produce terminal cost logs")
			})
		}
	}
}

func TestPollOtherTerminalStatesReturnAvailableResponse(t *testing.T) {
	t.Parallel()

	for _, status := range []string{"failed", "incomplete", "cancelled", "future_terminal_status"} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			var requests atomic.Int32
			client := newAccountingClient(t, func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				body := accountingResponse("resp_terminal", status, "flex", 1000, 100)
				if status == "failed" {
					body["error"] = map[string]string{"code": "server_error", "message": "generation failed"}
				}
				require.NoError(t, json.NewEncoder(w).Encode(body))
			})
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()

			resp, err := client.Poll(ctx, "resp_terminal", 0)

			require.ErrorContains(t, err, status)
			require.NotErrorIs(t, err, context.DeadlineExceeded)
			require.EqualValues(t, 1, requests.Load())
			require.NotNil(t, resp)
			require.Equal(t, "resp_terminal", resp.ID)
			require.Equal(t, status, resp.Status)
			require.Equal(t, "gpt-5.4", resp.Model)
			require.Equal(t, "flex", resp.ServiceTier)
			require.NotNil(t, resp.Usage)
			require.Equal(t, 1100, resp.Usage.TotalTokens)
		})
	}
}

type accountingLogBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *accountingLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *accountingLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.String()
}

func TestStreamingTerminalUsageLogsActualTier(t *testing.T) {
	t.Parallel()

	for _, transport := range []string{"SSE", "WebSocket"} {
		for _, status := range []string{"completed", "incomplete", "failed"} {
			t.Run(transport+"/"+status, func(t *testing.T) {
				t.Parallel()
				body := accountingResponse("resp_stream", status, "flex", 1000, 100, accountingMessage("Partial or final answer"))
				if status == "failed" {
					body["error"] = map[string]string{"code": "server_error", "message": "generation failed"}
				}
				event := map[string]any{"type": "response." + status, "response": body}
				client := newAccountingClient(t, func(w http.ResponseWriter, r *http.Request) {
					if transport == "SSE" {
						data, err := json.Marshal(event)
						require.NoError(t, err)
						w.Header().Set("Content-Type", "text/event-stream")
						writeSSE(w, "response."+status, string(data))
						return
					}
					upgrader := websocket.Upgrader{}
					conn, err := upgrader.Upgrade(w, r, nil)
					if err != nil {
						return
					}
					defer conn.Close()
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
					require.NoError(t, conn.WriteJSON(event))
					_, _, _ = conn.ReadMessage()
				})
				var logs accountingLogBuffer
				client.Log = slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
				req := &responses.Request{Model: "gpt-5.4", Input: "hi", ServiceTier: "default"}
				ctx, cancel := context.WithTimeout(t.Context(), time.Second)
				defer cancel()
				var stream *streaming.StreamIterator
				var err error
				if transport == "SSE" {
					stream, err = client.Stream(ctx, req)
				} else {
					ws := openTestWebSocket(t, client)
					defer ws.Close()
					stream, err = ws.Send(ctx, req)
				}
				require.NoError(t, err)
				require.True(t, stream.Next())
				switch status {
				case "completed":
					require.IsType(t, streaming.ResponseCompleted{}, stream.Event())
				case "incomplete":
					require.IsType(t, streaming.ResponseIncomplete{}, stream.Event())
				case "failed":
					require.IsType(t, streaming.ResponseFailed{}, stream.Event())
				}
				require.False(t, stream.Next())
				require.NoError(t, stream.Err())
				decoder := json.NewDecoder(strings.NewReader(logs.String()))
				var costRecords []map[string]any
				for {
					var record map[string]any
					err := decoder.Decode(&record)
					if errors.Is(err, io.EOF) {
						break
					}
					require.NoError(t, err)
					if record["responseID"] == "resp_stream" {
						costRecords = append(costRecords, record)
					}
				}
				require.Len(t, costRecords, 1)
				require.Equal(t, "DEBUG", costRecords[0]["level"])
				require.Equal(t, "gpt-5.4", costRecords[0]["model"])
				require.Equal(t, "flex", costRecords[0]["serviceTier"])
				require.Contains(t, costRecords[0]["msg"], "0.002000000")
			})
		}
	}
}
