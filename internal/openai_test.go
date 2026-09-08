// Package openai / openai_test.go tests automatic logging for HTTP requests and retries.
package openai

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDoAutoLogTripper(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.URL.Path == "/failure" {
					w.WriteHeader(http.StatusTooManyRequests)
				} else {
					w.WriteHeader(http.StatusAccepted)
				}
				_, _ = io.WriteString(w, "response body")
			}))
			defer server.Close()
			var logs bytes.Buffer
			lt := &LoggingTransport{Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
			client := NewHTTPClient()
			client.Transport = lt
			client.AutoLogTripper = true
			for i, path := range []string{"/failure", "/success"} {
				var body io.Reader
				if method == http.MethodPost {
					body = strings.NewReader("request body")
				}
				req, err := http.NewRequest(method, server.URL+path, body)
				require.NoError(t, err)
				resp, err := client.Do(req)
				require.NoError(t, err)
				got, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				require.NoError(t, resp.Body.Close())
				require.Equal(t, "response body", string(got))
				require.EqualValues(t, i+1, calls.Load(), "Do must not retry")
				if req.Body != nil {
					got, err := io.ReadAll(req.Body)
					require.NoError(t, err)
					require.Equal(t, "request body", string(got))
				}
				if path == "/failure" {
					require.True(t, lt.Enabled())
					require.Empty(t, logs.String(), "failure enables logging for subsequent requests")
				} else {
					require.False(t, lt.Enabled(), "202 is a successful response")
					require.Contains(t, logs.String(), "request:")
					require.Contains(t, logs.String(), "202 Accepted")
					require.Contains(t, logs.String(), "response body")
				}
			}
		})
	}
}

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
				require.False(t, lt.Enabled())
				if level == slog.LevelDebug {
					require.Contains(t, logs.String(), "200 OK")
					require.Contains(t, logs.String(), contentType)
					require.NotContains(t, logs.String(), "first event")
				} else {
					require.Empty(t, logs.String())
				}
			})
		}
	}
}

func TestDoKeepsManualLogSettingWhenAutoDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	for _, enabled := range []bool{false, true} {
		lt := &LoggingTransport{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), EnableLog: enabled}
		client := NewHTTPClient()
		client.Transport = lt
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.Equal(t, enabled, lt.Enabled())
	}
}

func TestDoAutoLogTripperOnConsecutiveTransportErrors(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	server.Close()
	var logs bytes.Buffer
	lt := &LoggingTransport{Log: slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug}))}
	client := NewHTTPClient()
	client.Transport = lt
	client.AutoLogTripper = true
	for range 2 {
		req, err := http.NewRequest(http.MethodGet, server.URL, nil)
		require.NoError(t, err)
		resp, err := client.Do(req)
		require.Error(t, err)
		require.Nil(t, resp)
		require.True(t, lt.Enabled())
	}
	require.Contains(t, logs.String(), "request failed:")
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
	require.False(t, lt.Enabled())
	require.Contains(t, logs.String(), "retry body")
	require.Contains(t, logs.String(), "200 OK")
}

func TestLogTripperConcurrentRequestsAndConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failure" {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()
	cfg := NewConfig("test-token")
	cfg.Log = slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg.HTTPClient.AutoLogTripper = true
	require.NoError(t, cfg.EnableLogTripper())
	const workers = 8
	workerErrors := make([]error, workers)
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for attempt := range 8 {
				var err error
				if attempt%2 == 0 {
					err = cfg.EnableLogTripper()
				} else {
					err = cfg.DisableLogTripper()
				}
				if err != nil {
					workerErrors[i] = err
					return
				}
				path := "/success"
				if i%2 == 0 {
					path = "/failure"
				}
				req, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
				if err != nil {
					workerErrors[i] = err
					return
				}
				resp, err := cfg.HTTPClient.Do(req)
				if err != nil {
					workerErrors[i] = err
					return
				}
				if err := resp.Body.Close(); err != nil {
					workerErrors[i] = fmt.Errorf("close response: %w", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	for _, err := range workerErrors {
		require.NoError(t, err)
	}
}
