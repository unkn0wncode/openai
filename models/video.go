// Package models / video.go contains pricing information for OpenAI video models.
package models

const (
	Sora2    = "sora-2"
	Sora2Pro = "sora-2-pro"
)

// PricePerSecond stores pricing data in USD per generated second indexed by the
// output resolution.
type PricePerSecond map[string]float64

// VideoData contains pricing of video generation models per rendered second.
var VideoData = map[string]PricePerSecond{
	Sora2: {
		"720x1280": 0.10,
		"1280x720": 0.10,
	},
	Sora2Pro: {
		"720x1280":  0.30,
		"1280x720":  0.30,
		"1024x1792": 0.50,
		"1792x1024": 0.50,
	},
}
