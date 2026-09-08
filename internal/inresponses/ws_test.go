// Package inresponses / ws_test.go tests WebSocket event delivery and connection handling.
package inresponses

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func newWSTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	cfg := openai.NewConfig("test-token")
	cfg.BaseAPI = server.URL + "/"
	return NewClient(cfg)
}

func wsTestRequest() *responses.Request {
	return &responses.Request{
		Model: "test-model",
		Input: "hi",
	}
}

func openTestWebSocket(t *testing.T, client *Client) responses.WSConn {
	t.Helper()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	ws, err := client.WebSocket(ctx)
	require.NoError(t, err)
	return ws
}

func TestWSTurnBuffersFirstEventBeforeConsumer(t *testing.T) {
	t.Parallel()

	turn := newWSTurn()
	delivered := make(chan bool, 1)

	go func() {
		delivered <- turn.deliver("first")
	}()

	select {
	case ok := <-delivered:
		require.True(t, ok)
	case <-time.After(time.Second):
		t.Fatal("first event delivery blocked before consumer started")
	}
}

func TestWSTurnFinishRejectsPendingOrLaterDelivery(t *testing.T) {
	t.Parallel()

	turn := newWSTurn()
	require.True(t, turn.deliver("first"))

	delivered := make(chan bool, 1)
	go func() {
		delivered <- turn.deliver("second")
	}()

	turn.finish(errors.New("closed"))

	select {
	case ok := <-delivered:
		require.False(t, ok)
	case <-time.After(time.Second):
		t.Fatal("finish did not unblock blocked delivery")
	}
}

func TestStreamYieldsBufferedEventBeforeCloseError(t *testing.T) {
	t.Parallel()

	turn := newWSTurn()
	require.True(t, turn.deliver("first"))
	turn.finish(errors.New("websocket connection closed"))

	stream := streaming.NewStream(context.Background(), turn)

	require.True(t, stream.Next())
	require.Equal(t, "first", stream.Event())
	require.False(t, stream.Next())
	require.ErrorContains(t, stream.Err(), "websocket connection closed")
}

func TestWebSocketCloseFinishesPendingTurn(t *testing.T) {
	t.Parallel()

	requestRead := make(chan struct{})
	client := newWSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		close(requestRead)

		_, _, _ = conn.ReadMessage()
	})
	ws := openTestWebSocket(t, client)

	stream, err := ws.Send(t.Context(), wsTestRequest())
	require.NoError(t, err)

	requireChannelClosed(t, requestRead)
	require.NoError(t, ws.Close())
	require.False(t, stream.Next())
	require.ErrorContains(t, stream.Err(), "websocket connection closed")
}

func TestWebSocketWSErrorBecomesStreamError(t *testing.T) {
	t.Parallel()

	client := newWSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{
			"type":   "error",
			"status": http.StatusBadRequest,
			"error": map[string]any{
				"code":    "previous_response_not_found",
				"message": "previous response not found",
			},
		})
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()

	stream, err := ws.Send(t.Context(), wsTestRequest())
	require.NoError(t, err)

	var gotErr error
	for event, err := range stream.Seq() {
		require.Nil(t, event)
		gotErr = err
	}

	var streamErr *streaming.StreamError
	require.ErrorAs(t, gotErr, &streamErr)
	require.EqualError(t, streamErr, "stream error: previous response not found")
}

func TestWebSocketResponseFailedIsDeliveredAsEvent(t *testing.T) {
	t.Parallel()

	client := newWSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{
			"type": "response.failed",
			"response": map[string]any{
				"id":     "resp_failed",
				"object": "response",
				"status": "failed",
				"error": map[string]any{
					"code":    "server_error",
					"message": "failed",
				},
			},
		})
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()

	stream, err := ws.Send(t.Context(), wsTestRequest())
	require.NoError(t, err)

	require.True(t, stream.Next())
	require.IsType(t, streaming.ResponseFailed{}, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
}

func TestWebSocketEarlyCloseOnHeadTurnAllowsNextTurn(t *testing.T) {
	t.Parallel()

	client := newWSTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{
			"type":  "response.output_text.delta",
			"delta": "first",
		})

		_, _, err = conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":     "resp_first",
				"object": "response",
				"status": "completed",
			},
		})
		_ = conn.WriteJSON(map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":     "resp_second",
				"object": "response",
				"status": "completed",
			},
		})
	})
	ws := openTestWebSocket(t, client)
	defer ws.Close()

	first, err := ws.Send(t.Context(), wsTestRequest())
	require.NoError(t, err)
	require.True(t, first.Next())
	require.IsType(t, streaming.ResponseOutputTextDelta{}, first.Event())
	first.Close()

	second, err := ws.Send(t.Context(), wsTestRequest())
	require.NoError(t, err)
	require.True(t, second.Next())
	require.IsType(t, streaming.ResponseCompleted{}, second.Event())
	require.False(t, second.Next())
	require.NoError(t, second.Err())
}
