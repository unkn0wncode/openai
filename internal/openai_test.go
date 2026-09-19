// Package openai / openai_test.go tests automatic logging for HTTP requests and retries.
package openai

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDoAutoLogTripperKeepsSSEStreaming(t *testing.T) {
	for _, level := range []slog.Level{slog.LevelInfo, slog.LevelDebug} {
		for _, contentType := range []string{"text/event-stream", "text/event-stream; charset=utf-8"} {
			t.Run(level.String()+"/"+contentType, func(t *testing.T) {
				const event = "data: first event\n\n"
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/failure" {
						w.WriteHeader(http.StatusTooManyRequests)
						return
					}
					w.Header().Set("Content-Type", contentType)
					if _, err := io.WriteString(w, event); err != nil {
						return
					}
					w.(http.Flusher).Flush()
					// Keep the body open until the client has consumed the event and closed it.
					<-r.Context().Done()
				}))
				defer server.Close()
				var logs bytes.Buffer
				lt := &LoggingTransport{Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: level}))}
				client := NewHTTPClient()
				client.Transport = lt
				client.AutoLogTripper = true
				req, err := http.NewRequest(http.MethodGet, server.URL+"/failure", nil)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.True(t, lt.Enabled())

				ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
				defer cancel()
				req, err = http.NewRequestWithContext(ctx, http.MethodGet, server.URL+"/stream", nil)
				require.NoError(t, err)
				resp, err = client.Do(req)
				require.NoError(t, err, "Do must return headers before the event stream ends")
				defer resp.Body.Close()
				got := make([]byte, len(event))
				_, err = io.ReadFull(resp.Body, got)
				require.NoError(t, err)
				require.Equal(t, event, string(got))
				if level == slog.LevelDebug {
					require.Contains(t, logs.String(), "200 OK")
					require.Contains(t, logs.String(), contentType)
					require.NotContains(t, logs.String(), "first event")
				}
			})
		}
	}
}

func TestWithRetryUsesDoAutoLogging(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil || string(body) != "retry body" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()
	var logs bytes.Buffer
	lt := &LoggingTransport{Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	client := NewHTTPClient()
	client.Transport = lt
	client.AutoLogTripper = true
	client.RequestAttempts = 2
	client.RetryInterval = 0
	req, err := http.NewRequest(http.MethodPost, server.URL, strings.NewReader("retry body"))
	require.NoError(t, err)
	resp, err := client.WithRetry(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.EqualValues(t, 2, calls.Load())
}
