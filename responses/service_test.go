package responses

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/content/output"
)

// TestResponseHelpers verifies all response helper methods.
func TestResponseHelpers(t *testing.T) {
	t.Run("Reasonings", func(t *testing.T) {
		r := &Response{ParsedOutputs: []any{
			output.Reasoning{Summary: []output.ReasoningSummary{{Text: "first"}, {Text: "second"}}},
			"unrelated",
			output.Reasoning{Summary: []output.ReasoningSummary{{Text: "third"}}},
		}}
		rs := r.Reasonings()
		require.Len(t, rs, 2)
		require.Equal(t, []string{"first", "second", "third"}, r.ReasoningSummaries())
		require.Equal(t, "first\nsecond\nthird", r.JoinedReasoningSummaries())
	})

	t.Run("Text", func(t *testing.T) {
		r := &Response{ParsedOutputs: []any{
			output.Message{Content: "hello"},
			output.Message{Content: []any{output.OutputText{Text: "world", Annotations: []output.AnyAnnotation{}}}},
		}}
		require.Equal(t, []string{"hello", "world"}, r.Texts())
		require.Equal(t, "hello\nworld", r.JoinedTexts())
		require.Equal(t, "hello", r.FirstText())
		require.Equal(t, "world", r.LastText())
	})

	t.Run("FunctionCallsAndRefusals", func(t *testing.T) {
		r := &Response{ParsedOutputs: []any{
			output.FunctionCall{Name: "fn", ID: "1", Arguments: "{}"},
			output.Message{Content: []any{output.Refusal{Refusal: "no!"}}},
		}}
		fc := r.FunctionCalls()
		require.Len(t, fc, 1)
		require.Equal(t, "fn", fc[0].Name)
		refs := r.Refusals()
		require.Len(t, refs, 1)
		require.Equal(t, "no!", refs[0])
	})
}
