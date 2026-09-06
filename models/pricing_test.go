// Package models / pricing_test.go tests cost calculations with synthetic rates and checks catalog validity.
package models

import (
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func testPricing() Pricing {
	return Pricing{
		LongContextThreshold: 100,
		standard: &tierRates{
			short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 3, output: 7},
			long:  &tokenRates{input: 11, cachedInput: 5, cacheWrite: 13, output: 17},
		},
		flex: &tierRates{
			short: tokenRates{input: 1, cachedInput: 0.25, cacheWrite: 2, output: 3},
			long:  &tokenRates{input: 4, cachedInput: 1, cacheWrite: 6, output: 8},
		},
		fast: &tierRates{
			short: tokenRates{input: 6, cachedInput: 2, cacheWrite: 10, output: 19},
			long:  &tokenRates{input: 20, cachedInput: 7, cacheWrite: 30, output: 50},
		},
	}
}

func TestTokenCostComponents(t *testing.T) {
	for _, tt := range []struct {
		name                           string
		input, cached, written, output int
		want                           float64
	}{
		{"uncached input", 10, 0, 0, 0, 0.000020},
		{"cached input", 10, 10, 0, 0, 0.000005},
		{"cache writes", 10, 0, 10, 0, 0.000030},
		{"output", 0, 0, 0, 10, 0.000070},
		{"mixed", 80, 20, 10, 5, 0.000175},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u := &Usage{InputTokens: tt.input, OutputTokens: tt.output, TotalTokens: tt.input + tt.output}
			u.InputTokensDetails.CachedTokens = tt.cached
			u.InputTokensDetails.CacheWriteTokens = tt.written
			u.OutputTokensDetails.ReasoningTokens = tt.output
			got, err := testPricing().Cost(u)
			require.NoError(t, err)
			require.InDelta(t, tt.want, got, 1e-12)
		})
	}
}

func TestServiceTierCost(t *testing.T) {
	for _, tt := range []struct {
		tier                string
		shortWant, longWant float64
	}{
		{"default", 0.000175, 0.001305},
		{"flex", 0.000090, 0.000480},
		{"fast", 0.000535, 0.002490},
	} {
		t.Run(tt.tier, func(t *testing.T) {
			p, err := testPricing().ForTier(tt.tier)
			require.NoError(t, err)
			u := &Usage{InputTokens: 80, OutputTokens: 5}
			u.InputTokensDetails.CachedTokens = 20
			u.InputTokensDetails.CacheWriteTokens = 10
			got, err := p.Cost(u)
			require.NoError(t, err)
			require.InDelta(t, tt.shortWant, got, 1e-12)
			u.InputTokens = 120
			got, err = p.Cost(u)
			require.NoError(t, err)
			require.InDelta(t, tt.longWant, got, 1e-12)
		})
	}
}

func TestLongContextBoundary(t *testing.T) {
	for _, tt := range []struct {
		tier  string
		input int
		want  float64
	}{
		{"default", 100, 0.000215},
		{"default", 101, 0.001096},
		{"flex", 100, 0.000110},
		{"flex", 101, 0.000404},
		{"fast", 100, 0.000655},
		{"fast", 101, 0.002110},
	} {
		t.Run(tt.tier+"/"+strconv.Itoa(tt.input), func(t *testing.T) {
			p, err := testPricing().ForTier(tt.tier)
			require.NoError(t, err)
			u := &Usage{InputTokens: tt.input, OutputTokens: 5}
			u.InputTokensDetails.CachedTokens = 20
			u.InputTokensDetails.CacheWriteTokens = 10
			got, err := p.Cost(u)
			require.NoError(t, err)
			require.InDelta(t, tt.want, got, 1e-12)
		})
	}
	// Cached tokens still count toward the context threshold.
	u := &Usage{InputTokens: 101}
	u.InputTokensDetails.CachedTokens = 101
	got, err := testPricing().Cost(u)
	require.NoError(t, err)
	require.InDelta(t, 0.000505, got, 1e-12)
}

func TestUnavailablePricing(t *testing.T) {
	catalog := map[string]Pricing{"test-unpriced": {}}
	for _, model := range []string{"test-missing", "test-unpriced"} {
		got, err := catalog[model].Cost(&Usage{InputTokens: 10})
		require.Error(t, err, "model %q", model)
		require.Zero(t, got, "model %q", model)
	}
	for _, tier := range []string{"auto", "test-unknown-tier"} {
		_, err := testPricing().ForTier(tier)
		require.Error(t, err, "tier %q", tier)
	}
	p := testPricing()
	p.flex, p.fast = nil, nil
	for _, tier := range []string{"flex", "fast", "priority"} {
		_, err := p.ForTier(tier)
		require.Error(t, err, "tier %q", tier)
	}
	p = testPricing()
	p.fast.long = nil
	p, err := p.ForTier("fast")
	require.NoError(t, err)
	_, err = p.Cost(&Usage{InputTokens: 100})
	require.NoError(t, err)
	got, err := p.Cost(&Usage{InputTokens: 101})
	require.Error(t, err)
	require.Zero(t, got)
}

func TestMissingRatesRetainSubtotal(t *testing.T) {
	p := testPricing()
	p.standard.short.cachedInput = unavailableRate
	p.standard.short.cacheWrite = unavailableRate
	u := &Usage{InputTokens: 80, OutputTokens: 5}
	u.InputTokensDetails.CachedTokens = 20
	u.InputTokensDetails.CacheWriteTokens = 10
	got, err := p.Cost(u)
	require.ErrorContains(t, err, "cached input")
	require.ErrorContains(t, err, "cache write")
	require.InDelta(t, 0.000135, got, 1e-12)
}

// There is no actual free model but we want to test the package API contract that distinguishes
// between nil and zero cost, preventing user from mistaking a pricing error for a real zero cost.
func TestNilVsFreeUsage(t *testing.T) {
	var usage *Usage
	_, err := testPricing().Cost(usage)
	require.Error(t, err)
	_, err = testPricing().Cost(nil)
	require.Error(t, err)
	got, err := testPricing().Cost(&Usage{})
	require.NoError(t, err)
	require.Zero(t, got)
	free := Pricing{standard: &tierRates{}}
	got, err = free.Cost(&Usage{InputTokens: 10, OutputTokens: 5})
	require.NoError(t, err)
	require.Zero(t, got)
}

func TestTierSelectionRetainsPricing(t *testing.T) {
	original := testPricing()
	u := &Usage{InputTokens: 10}
	p, err := original.ForTier("priority")
	require.NoError(t, err)
	got, err := p.Cost(u)
	require.NoError(t, err)
	require.InDelta(t, 0.000060, got, 1e-12)
	p, err = p.ForTier("")
	require.NoError(t, err)
	got, err = p.Cost(u)
	require.NoError(t, err)
	require.InDelta(t, 0.000020, got, 1e-12)
	got, err = original.Cost(u)
	require.NoError(t, err)
	require.InDelta(t, 0.000020, got, 1e-12)
}

type resourceUsage struct{ err error }

func (u resourceUsage) Cost(Pricing) (float64, error) { return 7, u.err }

func TestCostDispatch(t *testing.T) {
	expectedErr := errors.New("resource-specific component unavailable")
	u := resourceUsage{err: expectedErr}
	got, err := testPricing().Cost(u)
	require.Equal(t, 7.0, got)
	require.ErrorIs(t, err, expectedErr)
	got, err = (Pricing{}).Cost(u)
	require.Zero(t, got)
	require.Error(t, err)
	require.NotErrorIs(t, err, expectedErr)
}

func TestCatalogValidity(t *testing.T) {
	require.NotEmpty(t, Data)
	for _, id := range []string{Default, Latest, DefaultMini, DefaultNano} {
		require.Contains(t, Data, id)
	}
	for model, p := range Data {
		t.Run(model, func(t *testing.T) {
			require.GreaterOrEqual(t, p.LimitContext, 0)
			require.GreaterOrEqual(t, p.LimitOutput, 0)
			require.GreaterOrEqual(t, p.LongContextThreshold, 0)
			require.False(t, math.IsNaN(p.RegionalUplift) || math.IsInf(p.RegionalUplift, 0))
			require.GreaterOrEqual(t, p.RegionalUplift, 0.0)
			for _, tier := range []*tierRates{p.standard, p.flex, p.fast} {
				if tier == nil {
					continue
				}
				if tier.long != nil {
					require.Positive(t, p.LongContextThreshold)
				}
				for _, rates := range []*tokenRates{&tier.short, tier.long} {
					if rates == nil {
						continue
					}
					for _, rate := range []float64{rates.input, rates.cachedInput, rates.cacheWrite, rates.output} {
						require.False(t, math.IsNaN(rate) || math.IsInf(rate, 0))
						require.True(t, rate >= 0 || rate == unavailableRate, "invalid rate %v", rate)
					}
				}
			}
		})
	}
}
