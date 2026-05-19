package inresponses

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"

	"github.com/stretchr/testify/require"
)

type closeRecorderTransport struct {
	base   http.RoundTripper
	closed chan struct{}
	once   sync.Once
}

func (t *closeRecorderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	resp.Body = &closeRecorderBody{
		ReadCloser: resp.Body,
		close: func() {
			t.once.Do(func() {
				close(t.closed)
			})
		},
	}
	return resp, nil
}

type closeRecorderBody struct {
	io.ReadCloser
	close func()
}

func (b *closeRecorderBody) Close() error {
	err := b.ReadCloser.Close()
	b.close()
	return err
}

func newSSETestClient(t *testing.T, handler http.HandlerFunc) (*Client, <-chan struct{}) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	httpClient := server.Client()
	base := httpClient.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	bodyClosed := make(chan struct{})
	httpClient.Transport = &closeRecorderTransport{
		base:   base,
		closed: bodyClosed,
	}

	cfg := openai.NewConfig("test-token")
	cfg.BaseAPI = server.URL + "/"
	cfg.HTTPClient = &openai.HTTPClient{Client: httpClient}
	return NewClient(cfg), bodyClosed
}

func requireChannelClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("channel did not close")
	}
}

func streamRequest() *responses.Request {
	return &responses.Request{
		Model:  "test-model",
		Input:  "hi",
		Stream: true,
	}
}

func writeSSE(w http.ResponseWriter, event string, data string) {
	_, _ = io.WriteString(w, "event: "+event+"\n")
	_, _ = io.WriteString(w, "data: "+data+"\n\n")
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func TestSSEStreamHTTPErrorReturnsBeforeIteration(t *testing.T) {
	t.Parallel()

	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"bad token"}}`)
	})

	stream, err := client.Stream(t.Context(), streamRequest())

	require.Nil(t, stream)
	require.ErrorContains(t, err, "stream request failed with status 401 Unauthorized")
	require.ErrorContains(t, err, "bad token")
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamSetsStreamParameter(t *testing.T) {
	t.Parallel()

	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
		var reqBody struct {
			Stream bool `json:"stream"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&reqBody))
		require.True(t, reqBody.Stream)

		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, "response.completed", `{"type":"response.completed","response":{"id":"resp_test","object":"response","status":"completed"}}`)
	})

	req := &responses.Request{
		Model: "test-model",
		Input: "hi",
	}
	stream, err := client.Stream(t.Context(), req)
	require.NoError(t, err)
	require.False(t, req.Stream)

	require.True(t, stream.Next())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamCompletedClosesBody(t *testing.T) {
	t.Parallel()

	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, "response.completed", `{"type":"response.completed","response":{"id":"resp_test","object":"response","status":"completed"}}`)
	})

	stream, err := client.Stream(t.Context(), streamRequest())
	require.NoError(t, err)

	require.True(t, stream.Next())
	require.IsType(t, streaming.ResponseCompleted{}, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamAcceptsCRLFSeparators(t *testing.T) {
	t.Parallel()

	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: response.completed\r\n")
		_, _ = io.WriteString(w, `data: {"type":"response.completed","response":{"id":"resp_test","object":"response","status":"completed"}}`+"\r\n")
		_, _ = io.WriteString(w, "\r\n")
	})

	stream, err := client.Stream(t.Context(), streamRequest())
	require.NoError(t, err)

	require.True(t, stream.Next())
	require.IsType(t, streaming.ResponseCompleted{}, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamMalformedDataReturnsErr(t *testing.T) {
	t.Parallel()

	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, "response.completed", `{"type":`)
	})

	stream, err := client.Stream(t.Context(), streamRequest())
	require.NoError(t, err)

	require.False(t, stream.Next())
	require.ErrorContains(t, stream.Err(), "failed to unmarshal event data")
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamContextCancellationWhileReaderBlocks(t *testing.T) {
	t.Parallel()

	requestDone := make(chan struct{})
	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-r.Context().Done()
		close(requestDone)
	})

	ctx, cancel := context.WithCancel(t.Context())
	stream, err := client.Stream(ctx, streamRequest())
	require.NoError(t, err)

	cancel()

	require.False(t, stream.Next())
	require.ErrorIs(t, stream.Err(), context.Canceled)
	requireChannelClosed(t, requestDone)
	requireChannelClosed(t, bodyClosed)
}

func TestSSEStreamSeqEarlyBreakClosesBody(t *testing.T) {
	t.Parallel()

	requestDone := make(chan struct{})
	client, bodyClosed := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		writeSSE(w, "response.output_text.delta", `{"type":"response.output_text.delta","delta":"hi"}`)
		<-r.Context().Done()
		close(requestDone)
	})

	stream, err := client.Stream(t.Context(), streamRequest())
	require.NoError(t, err)

	var text strings.Builder
	for event, err := range stream.Seq() {
		require.NoError(t, err)
		if delta, ok := event.(streaming.ResponseOutputTextDelta); ok {
			text.WriteString(delta.Delta)
		}
		break
	}

	require.Equal(t, "hi", text.String())
	requireChannelClosed(t, requestDone)
	requireChannelClosed(t, bodyClosed)
}
