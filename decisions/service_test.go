// Package decisions / service_test.go tests Decisions input encoding and cost estimates.
package decisions

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/models"
)

// TestMessageMarshal checks that messages keep their boundaries and images are sent as base64 data URLs.
func TestMessageMarshal(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n")
	b, err := json.Marshal([]Message{
		{Text("Photo:"), Image{Data: png, Detail: "low"}},
		{Text("Second note.")},
	})
	require.NoError(t, err)
	require.JSONEq(t, `[
		{"role": "user", "content": [
			{"type": "input_text", "text": "Photo:"},
			{"type": "input_image", "image_url": "data:image/png;base64,iVBORw0KGgo=", "detail": "low"}
		]},
		{"role": "user", "content": [{"type": "input_text", "text": "Second note."}]}
	]`, string(b))

	_, err = json.Marshal(Message{Image{Data: []byte("https://example.com/image.png")}})
	require.ErrorContains(t, err, "non-image content type")

	// Longer than the decoded type-detection prefix.
	large := append(png, make([]byte, 1000)...)
	encoded := base64.StdEncoding.EncodeToString(large)
	fromBytes, err := json.Marshal(Image{Data: large, Detail: "high"})
	require.NoError(t, err)
	fromBase64, err := json.Marshal(Base64Image{Data: encoded, Detail: "high"})
	require.NoError(t, err)
	require.JSONEq(t, string(fromBytes), string(fromBase64))
	require.Contains(t, string(fromBase64), `"data:image/png;base64,`+encoded+`"`)

	fromDataURL, err := json.Marshal(Base64Image{Data: "data:image/jpeg;base64," + encoded, Detail: "high"})
	require.NoError(t, err)
	require.JSONEq(t, string(fromBytes), string(fromDataURL))
	declared, detected, err := Base64Image{Data: "data:image/jpeg;base64," + encoded}.MediaTypes()
	require.NoError(t, err)
	require.Equal(t, []string{"image/jpeg", "image/png"}, []string{declared, detected})
	declared, _, err = Base64Image{Data: encoded}.MediaTypes()
	require.NoError(t, err)
	require.Empty(t, declared)

	_, err = json.Marshal(Base64Image{Data: "data:image/png,%89PNG"})
	require.ErrorContains(t, err, "decode base64 image")
	_, err = json.Marshal(Base64Image{Data: base64.StdEncoding.EncodeToString([]byte("plain text"))})
	require.ErrorContains(t, err, "non-image content type")
}

// TestDecisionEstimateCost checks that only input is charged, with cache tokens at the input rate.
func TestDecisionEstimateCost(t *testing.T) {
	usage := func(input, cached, cacheWrite, output int) *models.Usage {
		u := &models.Usage{InputTokens: input, OutputTokens: output}
		u.InputTokensDetails.CachedTokens = cached
		u.InputTokensDetails.CacheWriteTokens = cacheWrite
		return u
	}
	for _, tc := range []struct {
		name, model, region string
		usage               *models.Usage
		want                float64
		wantErr             string
	}{
		{"cache at input rate, output free", models.GPT6Luna, "global", usage(100_000, 40_000, 10_000, 50), 0.01, ""},
		{"regional uplift", models.GPT6Luna, "eu", usage(100_000, 0, 0, 0), 0.011, ""},
		{"long context", models.GPT6Luna, "global", usage(300_000, 0, 0, 0), 0.06, ""},
		{"unknown region", models.GPT6Luna, "", usage(100_000, 0, 0, 0), 0.01, "region"},
		{"no usage", models.GPT6Luna, "global", nil, 0, "usage unavailable"},
		{"unknown model", models.GPT6Sol, "global", usage(100_000, 0, 0, 0), 0, "no Decisions pricing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := &Decision{Model: tc.model, Usage: tc.usage, ProcessingRegion: tc.region}
			cost, err := d.EstimateCost()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			require.InDelta(t, tc.want, cost, 1e-12)
		})
	}
}
