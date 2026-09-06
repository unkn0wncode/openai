// Package responses_test / pricing_test.go tests request previews, per-call estimates, and billing limitations.
package responses_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

func pricingOutputs(t *testing.T, data string) []output.Any {
	t.Helper()
	var items []output.Any
	require.NoError(t, json.Unmarshal([]byte(data), &items))
	return items
}

func TestPreviewCostAssumptions(t *testing.T) {
	req := &responses.Request{
		Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault,
		MaxOutputTokens: 10000,
	}
	usage := &responses.Usage{InputTokens: 10000, OutputTokens: 1000}
	usage.InputTokensDetails.CachedTokens = 6000
	usage.InputTokensDetails.CacheWriteTokens = 2000
	usage.OutputTokensDetails.ReasoningTokens = 500

	// 2K input at $10/M, 6K cached at $1/M, 2K writes at $12.50/M,
	// and the assumed 1K output at $50/M, including reasoning tokens.
	got, err := req.PreviewCost(usage)
	require.NoError(t, err)
	require.InDelta(t, 0.101, got, 1e-12)

	usage.OutputTokens = 100
	usage.OutputTokensDetails.ReasoningTokens = 50
	got, err = req.PreviewCost(usage)
	require.NoError(t, err)
	require.InDelta(t, 0.056, got, 1e-12)

	req.ServiceTier = responses.ServiceTierFlex
	got, err = req.PreviewCost(usage)
	require.NoError(t, err)
	require.InDelta(t, 0.028, got, 1e-12)
}

func TestPreviewCostRequiresPricingEvidence(t *testing.T) {
	for _, tier := range []string{"", responses.ServiceTierAuto, "future-tier"} {
		t.Run("tier/"+tier, func(t *testing.T) {
			req := &responses.Request{Model: models.GPT6Astra, ServiceTier: tier}
			got, err := req.PreviewCost(&responses.Usage{InputTokens: 1000})
			require.Error(t, err)
			require.Zero(t, got)
		})
	}
	req := &responses.Request{Model: "future-model", ServiceTier: responses.ServiceTierDefault}
	got, err := req.PreviewCost(&responses.Usage{InputTokens: 1000})
	require.ErrorContains(t, err, "future-model")
	require.Zero(t, got)

	req.Model = models.GPT6Astra
	got, err = req.PreviewCost(nil)
	require.ErrorContains(t, err, "usage")
	require.Zero(t, got)
}

func TestEstimateCostUsesReturnedModelAndTier(t *testing.T) {
	req := &responses.Request{Model: models.GPT6Astra, ServiceTier: responses.ServiceTierFast}
	resp := &responses.Response{
		Model: models.GPT56Sol, ServiceTier: responses.ServiceTierFlex,
		ProcessingRegion: "global", Usage: &responses.Usage{InputTokens: 1000, OutputTokens: 100},
	}
	got, err := req.EstimateCost(resp)
	require.NoError(t, err)
	require.InDelta(t, 0.003, got, 1e-12)
	require.Equal(t, got, resp.EstimatedCost)
	require.NoError(t, resp.CostError)

	for _, field := range []string{"model", "tier"} {
		t.Run("missing/"+field, func(t *testing.T) {
			missing := *resp
			if field == "model" {
				missing.Model = ""
			} else {
				missing.ServiceTier = ""
			}
			got, err := req.EstimateCost(&missing)
			require.Error(t, err)
			require.Zero(t, got)
		})
	}
}

func TestEstimateCostPricesCallsIndividually(t *testing.T) {
	resp := &responses.Response{
		Model: models.GPT6Astra, ServiceTier: responses.ServiceTierFast,
		ProcessingRegion: "global", Usage: &responses.Usage{InputTokens: 999999},
		Outputs: []output.Any{{Type: "file_search_call"}},
		Calls: []responses.Response{
			{ID: "first", Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault,
				ProcessingRegion: "global", Usage: &responses.Usage{InputTokens: 150000, OutputTokens: 1000, TotalTokens: 151000}},
			{ID: "second", Model: models.GPT6Astra, ServiceTier: responses.ServiceTierFlex,
				ProcessingRegion: "global", Usage: &responses.Usage{InputTokens: 150000, OutputTokens: 1000, TotalTokens: 151000}},
		},
	}
	got, err := (&responses.Request{}).EstimateCost(resp)
	require.NoError(t, err)
	// Both calls use short-context rates, despite their combined 300K input.
	require.InDelta(t, 2.325, got, 1e-12)
	require.InDelta(t, 1.55, resp.Calls[0].EstimatedCost, 1e-12)
	require.InDelta(t, 0.775, resp.Calls[1].EstimatedCost, 1e-12)
	require.NoError(t, resp.Calls[0].CostError)
	require.NoError(t, resp.Calls[1].CostError)
	usage, err := resp.TotalUsage()
	require.NoError(t, err)
	require.Equal(t, 300000, usage.InputTokens)
	require.Equal(t, 2000, usage.OutputTokens)
	require.Equal(t, 302000, usage.TotalTokens)
}

func TestEstimateCostJoinsIndividualErrorsAndSubtotals(t *testing.T) {
	search := pricingOutputs(t, `[{"type":"web_search_call","id":"search","action":{"type":"search"}}]`)
	resp := &responses.Response{Calls: []responses.Response{
		{ID: "missing-model", Model: "future-model", ServiceTier: responses.ServiceTierDefault,
			Outputs: []output.Any{{Type: "file_search_call"}}},
		{ID: "missing-tier", Model: models.GPT6Astra, ServiceTier: "future-tier",
			Tools: []tools.Tool{{Type: "web_search"}}, Outputs: search},
		{ID: "missing-usage", Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global"},
		{ID: "known", Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global",
			Usage: &responses.Usage{InputTokens: 1000, OutputTokens: 1000}},
	}}
	got, err := (&responses.Request{}).EstimateCost(resp)
	require.Error(t, err)
	require.InDelta(t, 0.0725, got, 1e-12)
	require.InDelta(t, 0.0025, resp.Calls[0].EstimatedCost, 1e-12)
	require.InDelta(t, 0.01, resp.Calls[1].EstimatedCost, 1e-12)
	require.Zero(t, resp.Calls[2].EstimatedCost)
	for i := range 3 {
		call := &resp.Calls[i]
		require.Error(t, call.CostError)
		require.True(t, errors.Is(err, call.CostError), "aggregate lost call %d's error", i+1)
		require.Contains(t, err.Error(), call.ID)
	}
	require.NoError(t, resp.Calls[3].CostError)
	require.Equal(t, got, resp.EstimatedCost)
	require.True(t, errors.Is(resp.CostError, err))
}

func TestEstimateCostMissingUsageDiffersFromZero(t *testing.T) {
	resp := &responses.Response{
		Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global",
	}
	req := &responses.Request{}
	got, err := req.EstimateCost(resp)
	require.Zero(t, got)
	require.ErrorContains(t, err, "usage")
	_, err = resp.TotalUsage()
	require.ErrorContains(t, err, "usage")

	resp.Usage = &responses.Usage{}
	got, err = req.EstimateCost(resp)
	require.Zero(t, got)
	require.NoError(t, err)
	usage, err := resp.TotalUsage()
	require.NoError(t, err)
	require.Equal(t, responses.Usage{}, usage)
}

func TestEstimateCostIncompleteBillingKeepsObservedUsage(t *testing.T) {
	usage := &responses.Usage{InputTokens: 1000, OutputTokens: 1000, TotalTokens: 2000}
	usage.InputTokensDetails.CachedTokens = 200
	usage.InputTokensDetails.CacheWriteTokens = 100
	usage.OutputTokensDetails.ReasoningTokens = 500
	resp := &responses.Response{BillingIncomplete: true, Calls: []responses.Response{
		{ID: "observed", Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault,
			ProcessingRegion: "global", Usage: usage},
	}}
	got, err := (&responses.Request{}).EstimateCost(resp)
	require.ErrorContains(t, err, "attempted API request")
	require.InDelta(t, 0.05845, got, 1e-12)
	require.NoError(t, resp.Calls[0].CostError)
	aggregate, err := resp.TotalUsage()
	require.ErrorContains(t, err, "attempted API request")
	require.Equal(t, *usage, aggregate)

	resp.BillingIncomplete = false
	resp.Calls = append(resp.Calls, responses.Response{ID: "unobserved"})
	aggregate, err = resp.TotalUsage()
	require.ErrorContains(t, err, "unobserved")
	require.Equal(t, *usage, aggregate)
}

func TestEstimateCostRegionalPricing(t *testing.T) {
	for _, tt := range []struct {
		region string
		want   float64
		err    string
	}{
		{"global", 0.06, ""},
		{"us", 0.066, ""},
		{"eu", 0.066, ""},
		{"", 0.06, "regional"},
		{"unknown", 0.06, "regional"},
	} {
		t.Run(tt.region, func(t *testing.T) {
			resp := &responses.Response{
				Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault,
				ProcessingRegion: tt.region, Usage: &responses.Usage{InputTokens: 1000, OutputTokens: 1000},
			}
			got, err := (&responses.Request{}).EstimateCost(resp)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
			require.InDelta(t, tt.want, got, 1e-12)
		})
	}
	for _, tier := range []string{responses.ServiceTierFast, responses.ServiceTierPriority} {
		t.Run("eu/"+tier, func(t *testing.T) {
			resp := &responses.Response{
				Model: models.GPT6Astra, ServiceTier: tier, ProcessingRegion: "eu",
				Usage:   &responses.Usage{InputTokens: 1000, OutputTokens: 1000},
				Outputs: []output.Any{{Type: "file_search_call"}},
			}
			got, err := (&responses.Request{}).EstimateCost(resp)
			require.ErrorContains(t, err, "EU")
			require.InDelta(t, 0.0025, got, 1e-12)
		})
	}
}

func TestEstimateCostWebSearch(t *testing.T) {
	search := pricingOutputs(t, `[
		{"type":"web_search_call","id":"search","action":{"type":"search"}},
		{"type":"web_search_call","id":"open","action":{"type":"open_page"}},
		{"type":"web_search_call","id":"find","action":{"type":"find_in_page"}}
	]`)
	for _, tt := range []struct {
		name, model, tool string
		want              float64
		err               string
	}{
		{"standard", models.GPT6Astra, "web_search", 0.16, ""},
		{"preview reasoning", models.GPT6Astra, "web_search_preview", 0.16, ""},
		{"unknown variant", models.GPT6Astra, "", 0.05, "variant"},
		{"fixed search input", models.GPT41Mini, "web_search", 0.0116, "fixed search-content"},
		{"free search input", models.GPT41, "web_search_preview", 0.033, "free search-content"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := &responses.Response{
				Model: tt.model, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global",
				Usage: &responses.Usage{InputTokens: 10000, OutputTokens: 1000}, Outputs: search,
			}
			if tt.tool != "" {
				resp.Tools = []tools.Tool{{Type: tt.tool}}
			}
			got, err := (&responses.Request{}).EstimateCost(resp)
			if tt.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.err)
			}
			require.InDelta(t, tt.want, got, 1e-12)
		})
	}
	// An unitemized input charge must not erase the known long-context output rate.
	resp := &responses.Response{
		Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global",
		Usage: &responses.Usage{InputTokens: 300000, OutputTokens: 1000}, Outputs: search,
	}
	got, err := (&responses.Request{}).EstimateCost(resp)
	require.ErrorContains(t, err, "variant")
	require.InDelta(t, 0.075, got, 1e-12)
}

func TestEstimateCostRetainsPartialTokenAndToolCharges(t *testing.T) {
	usage := &responses.Usage{InputTokens: 1000, OutputTokens: 1000}
	usage.InputTokensDetails.CachedTokens = 500
	resp := &responses.Response{
		Model: models.GPT4o20240513, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global", Usage: usage,
		Outputs: []output.Any{{Type: "file_search_call"}, {Type: "code_interpreter_call"}},
	}
	got, err := (&responses.Request{}).EstimateCost(resp)
	require.ErrorContains(t, err, "cached input")
	require.ErrorContains(t, err, "storage")
	require.ErrorContains(t, err, "container")
	// 500 known input at $5/M + 1K output at $15/M + one $0.0025 file search.
	require.InDelta(t, 0.02, got, 1e-12)
	require.Equal(t, got, resp.EstimatedCost)
}

func TestEstimateCostShellNeedsExecutionEnvironment(t *testing.T) {
	for _, tt := range []struct {
		name  string
		tools []tools.Tool
		local bool
	}{
		{name: "missing definition"},
		{name: "missing environment", tools: []tools.Tool{{Type: "shell"}}},
		{name: "hosted", tools: []tools.Tool{{Type: "shell", Environment: map[string]any{"type": "container_auto"}}}},
		{name: "local", tools: []tools.Tool{{Type: "shell", Environment: map[string]any{"type": "local"}}}, local: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp := &responses.Response{
				Model: models.GPT6Astra, ServiceTier: responses.ServiceTierDefault, ProcessingRegion: "global",
				Usage:   &responses.Usage{InputTokens: 1000, OutputTokens: 1000},
				Outputs: []output.Any{{Type: "shell_call"}}, Tools: tt.tools,
			}
			got, err := (&responses.Request{}).EstimateCost(resp)
			if tt.local {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "container")
			}
			require.InDelta(t, 0.06, got, 1e-12)
		})
	}
}
