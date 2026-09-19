// Package models / tts.go contains list and properties of OpenAI text-to-speech models.
package models

const (
	TTS11106             = "tts-1-1106"
	TTS1HD               = "tts-1-hd"
	TTS1HD1106           = "tts-1-hd-1106"
	GPT4oMiniTTS20250320 = "gpt-4o-mini-tts-2025-03-20"
	GPT4oMiniTTS20251215 = "gpt-4o-mini-tts-2025-12-15"
)

// TTSPricing captures pricing and limits specific to text-to-speech models.
// PricePerCharacter is USD per character; token prices are USD per million tokens.
// A rate of -1 means unavailable.
// ApproxUSDPerMinute is an estimate, not a duration-based billing rate.
type TTSPricing struct {
	PricePerCharacter  float64
	PriceInText        float64
	PriceOutAudio      float64
	ApproxUSDPerMinute float64
	LimitCharacters    int
	LimitInputTokens   int
}

// DataTTS lists pricing information for text-to-speech models.
var DataTTS = map[string]TTSPricing{
	TTS11106: {
		PricePerCharacter:  0.000015,
		PriceInText:        unavailableRate,
		PriceOutAudio:      unavailableRate,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
	},
	TTS1HD: {
		PricePerCharacter:  0.000030,
		PriceInText:        unavailableRate,
		PriceOutAudio:      unavailableRate,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
	},
	TTS1HD1106: {
		PricePerCharacter:  0.000030,
		PriceInText:        unavailableRate,
		PriceOutAudio:      unavailableRate,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
	},
	GPT4oMiniTTS: {
		PricePerCharacter:  unavailableRate,
		PriceInText:        0.6,
		PriceOutAudio:      12,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
		LimitInputTokens:   2000,
	},
	GPT4oMiniTTS20250320: {
		PricePerCharacter:  unavailableRate,
		PriceInText:        0.6,
		PriceOutAudio:      12,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
		LimitInputTokens:   2000,
	},
	GPT4oMiniTTS20251215: {
		PricePerCharacter:  unavailableRate,
		PriceInText:        0.6,
		PriceOutAudio:      12,
		ApproxUSDPerMinute: unavailableRate,
		LimitCharacters:    4096,
		LimitInputTokens:   2000,
	},
}
