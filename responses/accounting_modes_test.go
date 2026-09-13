// Package responses_test / accounting_modes_test.go tests pricing evidence for execution modes and hosted outputs.
package responses_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func TestEstimateCostDoesNotPriceAggregateWorkAsOneContext(t *testing.T) {
	// Select a supported context-pricing fixture without pinning its ID or rates.
	var model string
	var pricing models.Pricing
	for name, candidate := range models.Data {
		if candidate.LongContextThreshold == 0 {
			continue
		}
		if _, err := candidate.Cost(&models.Usage{InputTokens: candidate.LongContextThreshold + 1, OutputTokens: 1}); err == nil {
			model, pricing = name, candidate
			break
		}
	}
	if model == "" {
		t.Skip("catalog has no supported long-context pricing")
	}
	usage := &responses.Usage{InputTokens: pricing.LongContextThreshold + 1, OutputTokens: 10}
	base := responses.Response{
		Model: model, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global", Usage: usage,
		Tools:   []tools.Tool{{Type: "web_search"}},
		Outputs: pricingOutputs(t, `[{"type":"web_search_call","id":"search","action":{"type":"search"}}]`),
	}
	toolOnly := base
	toolOnly.Usage = &responses.Usage{}
	toolCost, err := (*responses.Request)(nil).EstimateCost(&toolOnly)
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		request   *responses.Request
		reasoning *responses.ReasoningConfig
		multi     *responses.MultiAgentConfig
	}{
		{name: "returned pro", reasoning: &responses.ReasoningConfig{Mode: "pro"}},
		{name: "requested pro with missing mode", request: &responses.Request{Reasoning: &responses.ReasoningConfig{Mode: "pro"}}, reasoning: &responses.ReasoningConfig{Effort: "low"}},
		{name: "returned multi-agent", multi: &responses.MultiAgentConfig{Enabled: true}},
		{name: "requested multi-agent", request: &responses.Request{MultiAgent: &responses.MultiAgentConfig{Enabled: true}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := base
			resp.Reasoning, resp.MultiAgent = tc.reasoning, tc.multi
			cost, err := tc.request.EstimateCost(&resp)
			require.ErrorContains(t, err, "context sizes")
			require.InDelta(t, toolCost, cost, 1e-12, "known tool charges survive unpriceable aggregate token work")
			got, usageErr := resp.TotalUsage()
			require.NoError(t, usageErr)
			require.Equal(t, *usage, got, "pricing uncertainty must not erase observed usage")
		})
	}

	// Effective standard execution overrides requested Pro mode.
	actual := base
	actual.Reasoning = &responses.ReasoningConfig{Mode: "standard"}
	cost, err := (&responses.Request{Reasoning: &responses.ReasoningConfig{Mode: "pro"}}).EstimateCost(&actual)
	require.NoError(t, err)
	tokenCost, err := pricing.Cost(usage)
	require.NoError(t, err)
	require.InDelta(t, tokenCost+toolCost, cost, 1e-12)
}
