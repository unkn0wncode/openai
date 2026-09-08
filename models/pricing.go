// Package models / pricing.go provides shared token usage and service-tier cost calculations.
package models

import (
	"errors"
	"fmt"
)

// Usage contains the token counters shared by API resources.
type Usage struct {
	InputTokens        int `json:"input_tokens"`
	InputTokensDetails struct {
		CachedTokens     int `json:"cached_tokens"`
		CacheWriteTokens int `json:"cache_write_tokens"`
	} `json:"input_tokens_details"`
	OutputTokens        int `json:"output_tokens"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
	TotalTokens int `json:"total_tokens"`
}

// Pricing contains token rates and limits for one model. Its zero value has no
// pricing. Cost uses Standard rates unless ForTier selects another service tier.
type Pricing struct {
	LimitContext         int
	LimitOutput          int
	LongContextThreshold int
	// RegionalUplift is the additional fraction charged for eligible regional processing.
	RegionalUplift float64

	standard *tierRates
	flex     *tierRates
	fast     *tierRates
	tier     string
}

type tierRates struct {
	short tokenRates
	long  *tokenRates
}

// Rates are USD per million tokens. An unavailable rate is distinct from free.
type tokenRates struct {
	input, cachedInput, cacheWrite, output float64
}

const unavailableRate = -1

// ForTier returns pricing for the requested service tier. Empty and "default"
// select Standard pricing; "priority" is the former name of "fast". "auto"
// cannot be priced without knowing which tier actually served the request.
func (p Pricing) ForTier(tier string) (Pricing, error) {
	if p.standard == nil {
		return Pricing{}, errors.New("model pricing unavailable")
	}
	switch tier {
	case "", "default":
		p.tier = ""
	case "flex":
		if p.flex == nil {
			return Pricing{}, errors.New("flex pricing unavailable for model")
		}
		p.tier = "flex"
	case "fast", "priority":
		if p.fast == nil {
			return Pricing{}, errors.New("fast pricing unavailable for model")
		}
		p.tier = "fast"
	default:
		return Pricing{}, fmt.Errorf("pricing unavailable for service tier %q", tier)
	}
	return p, nil
}

// Cost delegates to the usage type so API resources can supply their own cost
// calculation. Unknown pricing always returns an error, including missing Data entries.
func (p Pricing) Cost(usage interface {
	Cost(Pricing) (float64, error)
},
) (float64, error) {
	if p.standard == nil {
		return 0, errors.New("model pricing unavailable")
	}
	if usage == nil {
		return 0, errors.New("token usage unavailable")
	}
	return usage.Cost(p)
}

// Cost returns the token-cost estimate in USD. A nonnil error means the returned
// amount is only the subtotal of priced tokens.
func (u *Usage) Cost(p Pricing) (float64, error) {
	if u == nil {
		return 0, errors.New("token usage unavailable")
	}
	if p.standard == nil {
		return 0, errors.New("model pricing unavailable")
	}
	if u.InputTokens < 0 ||
		u.OutputTokens < 0 ||
		u.InputTokensDetails.CachedTokens < 0 ||
		u.InputTokensDetails.CacheWriteTokens < 0 {
		return 0, errors.New("invalid token counts")
	}
	tier := p.standard
	switch p.tier {
	case "flex":
		tier = p.flex
	case "fast":
		tier = p.fast
	}
	rates := &tier.short
	if p.LongContextThreshold != 0 && u.InputTokens > p.LongContextThreshold {
		if tier.long == nil {
			return 0, errors.New("long-context pricing unavailable for service tier")
		}
		rates = tier.long
	}
	uncachedInput := u.InputTokens - u.InputTokensDetails.CachedTokens - u.InputTokensDetails.CacheWriteTokens
	var cost float64
	var errs []error
	if uncachedInput < 0 {
		uncachedInput = 0
		errs = append(errs, errors.New("input token details exceed input tokens; uncached input was clamped to zero"))
	}
	for _, part := range []struct {
		name   string
		tokens int
		rate   float64
	}{
		{"input", uncachedInput, rates.input},
		{"cached input", u.InputTokensDetails.CachedTokens, rates.cachedInput},
		{"cache write", u.InputTokensDetails.CacheWriteTokens, rates.cacheWrite},
		{"output", u.OutputTokens, rates.output},
	} {
		if part.tokens == 0 {
			continue
		}
		if part.rate == unavailableRate {
			errs = append(errs, fmt.Errorf("%s token pricing unavailable", part.name))
			continue
		}
		cost += float64(part.tokens) * part.rate / 1_000_000
	}
	return cost, errors.Join(errs...)
}
