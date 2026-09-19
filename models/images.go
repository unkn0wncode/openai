// Package models / images.go contains list and properties of OpenAI image generation models.
package models

const (
	DefaultImage               = GPTImage2
	GPTImage1                  = "gpt-image-1"
	GPTImage15                 = "gpt-image-1.5"
	GPTImage1Mini              = "gpt-image-1-mini"
	GPTImage2                  = "gpt-image-2"
	GPTImage25Flare            = "gpt-image-2.5-flare"
	GPTImage25Flare20260908    = "gpt-image-2.5-flare-2026-09-08"
	GPTImage25Sunburst         = "gpt-image-2.5-sunburst"
	GPTImage25Sunburst20260908 = "gpt-image-2.5-sunburst-2026-09-08"
	GPTImage220260421          = "gpt-image-2-2026-04-21"
	ChatGPTImageLatest         = "chatgpt-image-latest"
)

// ImageData contains pricing and limits for image generation models.
// Token prices are USD per million tokens; -1 means unavailable and zero means free.
// PriceOut is the image output rate, while PriceOutText is the text output rate.
// Prompt size limit is in characters here, not in tokens.
// A nil PricePerImage means no per-image estimate is available.
var ImageData = map[string]struct {
	PriceInText        float64
	PriceInTextCached  float64
	PriceInImage       float64
	PriceInImageCached float64
	PriceOut           float64
	PriceOutText       float64
	PricePerImage      PricePerImage
	LimitPrompt        int
	LimitInImages      int
	LimitInImageSize   int // in bytes
	LimitOutImages     int
}{
	GPTImage1: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       10,
		PriceInImageCached: 2.5,
		PriceOut:           40,
		PriceOutText:       unavailableRate,
		PricePerImage:      PricePerImageData[GPTImage1],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage15: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           32,
		PriceOutText:       10,
		PricePerImage:      PricePerImageData[GPTImage15],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage1Mini: {
		PriceInText:        2,
		PriceInTextCached:  0.2,
		PriceInImage:       2.5,
		PriceInImageCached: 0.25,
		PriceOut:           8,
		PriceOutText:       unavailableRate,
		PricePerImage:      PricePerImageData[GPTImage1Mini],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage2: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		PricePerImage:      PricePerImageData[GPTImage2],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage220260421: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		PricePerImage:      PricePerImageData[GPTImage220260421],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage25Flare: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 50 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage25Flare20260908: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 50 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage25Sunburst: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 50 * 1024 * 1024, LimitOutImages: 10,
	},
	GPTImage25Sunburst20260908: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           30,
		PriceOutText:       unavailableRate,
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 50 * 1024 * 1024, LimitOutImages: 10,
	},
	ChatGPTImageLatest: {
		PriceInText:        5,
		PriceInTextCached:  1.25,
		PriceInImage:       8,
		PriceInImageCached: 2,
		PriceOut:           32,
		PriceOutText:       10,
		PricePerImage:      PricePerImageData[ChatGPTImageLatest],
		LimitPrompt:        32000, LimitInImages: 16, LimitInImageSize: 25 * 1024 * 1024, LimitOutImages: 10,
	},
}

// PricePerImage maps quality and size to estimated price per generated image in USD.
// A missing quality or size has no estimate, rather than a zero price.
type PricePerImage map[string]map[string]float64

// PricePerImageData contains model-specific estimates for generated images.
// Entries do not describe every supported quality or size; use ImageData for model inventory.
// GPT Image 2 per-image prices are calculator-derived estimates as of 2026-05-18;
// token prices in ImageData are the authoritative billing rates.
var PricePerImageData = map[string]PricePerImage{
	ChatGPTImageLatest: {
		"low": {
			"1024x1024": 0.009,
			"1024x1536": 0.013,
			"1536x1024": 0.013,
		},
		"medium": {
			"1024x1024": 0.034,
			"1024x1536": 0.05,
			"1536x1024": 0.05,
		},
		"high": {
			"1024x1024": 0.133,
			"1024x1536": 0.2,
			"1536x1024": 0.2,
		},
	},
	GPTImage1: {
		"low": {
			"1024x1024": 0.011,
			"1024x1536": 0.016,
			"1536x1024": 0.016,
		},
		"medium": {
			"1024x1024": 0.042,
			"1024x1536": 0.063,
			"1536x1024": 0.063,
		},
		"high": {
			"1024x1024": 0.167,
			"1024x1536": 0.25,
			"1536x1024": 0.25,
		},
	},
	GPTImage15: {
		"low": {
			"1024x1024": 0.009,
			"1024x1536": 0.013,
			"1536x1024": 0.013,
		},
		"medium": {
			"1024x1024": 0.034,
			"1024x1536": 0.05,
			"1536x1024": 0.05,
		},
		"high": {
			"1024x1024": 0.133,
			"1024x1536": 0.2,
			"1536x1024": 0.2,
		},
	},
	GPTImage1Mini: {
		"low": {
			"1024x1024": 0.005,
			"1024x1536": 0.006,
			"1536x1024": 0.006,
		},
		"medium": {
			"1024x1024": 0.011,
			"1024x1536": 0.015,
			"1536x1024": 0.015,
		},
		"high": {
			"1024x1024": 0.036,
			"1024x1536": 0.052,
			"1536x1024": 0.052,
		},
	},
	GPTImage2: {
		"low": {
			"1024x1024": 0.006,
			"1024x1536": 0.005,
			"1536x1024": 0.005,
		},
		"medium": {
			"1024x1024": 0.053,
			"1024x1536": 0.041,
			"1536x1024": 0.041,
		},
		"high": {
			"1024x1024": 0.211,
			"1024x1536": 0.165,
			"1536x1024": 0.165,
		},
	},
	GPTImage220260421: {
		"low": {
			"1024x1024": 0.006,
			"1024x1536": 0.005,
			"1536x1024": 0.005,
		},
		"medium": {
			"1024x1024": 0.053,
			"1024x1536": 0.041,
			"1536x1024": 0.041,
		},
		"high": {
			"1024x1024": 0.211,
			"1024x1536": 0.165,
			"1536x1024": 0.165,
		},
	},
}
