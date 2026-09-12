// Package inresponses / tokens_test.go tests input token counting requests and responses.
package inresponses

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
)

func TestCountInputTokensMissingZeroAndHTTPError(t *testing.T) {
	for _, tc := range []struct {
		name, body    string
		status, count int
		wantError     string
	}{
		{"zero", `{"input_tokens":0}`, 200, 0, ""},
		{"missing", `{}`, 200, 0, "missing"},
		{"null", `{"input_tokens":null}`, 200, 0, "missing"},
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

// TestCountInputTokensAPI checks the live token-counting API contract.
func TestCountInputTokensAPI(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test disabled in short mode")
	}
	token := os.Getenv("OPENAI_API_KEY")
	if token == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	client := NewClient(openai.NewConfig(token))
	count, err := client.CountInputTokens(t.Context(), &responses.Request{
		Model: models.Default,
		Input: "Hello, world!",
	})
	require.NoError(t, err)
	require.Positive(t, count)
}
