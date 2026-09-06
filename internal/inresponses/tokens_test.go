// Package inresponses / tokens_test.go tests input token counting requests and responses.
package inresponses

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func TestCountInputTokensUsesCountSchemaAndResolvedTools(t *testing.T) {
	client, closed := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/responses/input_tokens", r.URL.Path)
		require.Equal(t, "POST", r.Method)
		require.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		var body map[string]json.RawMessage
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.JSONEq(t, `"gpt-6-astra"`, string(body["model"]))
		require.JSONEq(t, `"hi"`, string(body["input"]))
		require.JSONEq(t, `"summarize"`, string(body["instructions"]))
		for _, name := range []string{"service_tier", "max_output_tokens", "background", "stream", "metadata"} {
			require.NotContains(t, body, name)
		}
		var definitions []tools.Tool
		require.NoError(t, json.Unmarshal(body["tools"], &definitions))
		require.Len(t, definitions, 1)
		require.Equal(t, "lookup", definitions[0].Name)
		require.JSONEq(t, `{"type":"object","properties":{"query":{"type":"string"}}}`, string(definitions[0].Parameters))
		_, _ = io.WriteString(w, `{"object":"response.input_tokens","input_tokens":137,"future_field":true}`)
	})
	require.NoError(t, client.Tools.CreateFunction(tools.FunctionCall{
		Name: "lookup", Description: "Look up a query",
		ParamsSchema: json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"}}}`),
	}))
	req := &responses.Request{
		Input: "hi", Instructions: "summarize", Tools: []string{"lookup"},
		ServiceTier: responses.ServiceTierFast, MaxOutputTokens: 500, Background: true, Stream: true,
		Metadata: map[string]string{"purpose": "count"},
	}
	count, err := client.CountInputTokens(t.Context(), req)
	require.NoError(t, err)
	require.Equal(t, 137, count)
	require.Empty(t, req.Model, "counting must not change the request")
	require.True(t, req.Stream)
	requireChannelClosed(t, closed)
}

func TestCountInputTokensMissingZeroAndHTTPError(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		status, count int
		wantError     string
	}{
		{"zero", `{"input_tokens":0}`, 200, 0, ""},
		{"missing", `{}`, 200, 0, "missing"},
		{"null", `{"input_tokens":null}`, 200, 0, "missing"},
		{"http_error", `{"error":{"message":"bad input"}}`, 400, 0, "bad input"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, closed := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			count, err := client.CountInputTokens(t.Context(), &responses.Request{Model: models.GPT6Astra, Input: "hi"})
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.count, count)
			}
			requireChannelClosed(t, closed)
		})
	}
}

func TestCountInputTokensDoesNotOmitUnresolvedPrompt(t *testing.T) {
	client, _ := newSSETestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("must not send a different input to the counter")
	})
	_, err := client.CountInputTokens(t.Context(), &responses.Request{Prompt: &responses.Prompt{ID: "prompt"}})
	require.ErrorContains(t, err, "expanded input")
}
