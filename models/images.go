// Package models / images.go contains list and properties of OpenAI image generation models.
package models

const (
	DefaultImage = GPTImage1
	GPTImage1    = "gpt-image-1"
	DALLE2       = "dall-e-2"
	DALLE3       = "dall-e-3"
)

// ImageData contains pricing and limits for image generation models.
// Prompt size limit is in characters here, not in tokens.
var ImageData = map[string]struct {
	PriceInText      float64
	PriceInImage     float64
	PriceOut         float64
	PricePerImage    PricePerImage
	LimitPrompt      int
	LimitInImages    int
	LimitInImageSize int // in bytes
	LimitOutImages   int
}{
	GPTImage1: {0.00000500, 0.00001000, 0.00004000, PricePerImageData[GPTImage1], 32000, 16, 25*1024*1024, 10},
	DALLE2:    {0.00000000, 0.00000000, 0.00000000, PricePerImageData[DALLE2], 1000, 1, 4*1024*1024, 1},
	DALLE3:    {0.00000000, 0.00000000, 0.00000000, PricePerImageData[DALLE3], 4000, 1, 4*1024*1024, 1},
}

// PricePerImageData contains pricing in USD per generated image depending on
// (1) quality and (2) size.
type PricePerImage map[string]map[string]float64

// PricePerImageData contains pricing of generated images for image generation models.
// For newer models, same pricing can be calculated based on token usage data.
var PricePerImageData = map[string]PricePerImage{
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
	DALLE2: {
		"standard": {
			"256x256":   0.016,
			"512x512":   0.018,
			"1024x1024": 0.02,
		},
	},
	DALLE3: {
		"standard": {
			"1024x1024": 0.04,
			"1024x1792": 0.08,
			"1792x1024": 0.08,
		},
		"hd": {
			"1024x1024": 0.08,
			"1024x1792": 0.12,
			"1792x1024": 0.12,
		},
	},
}
