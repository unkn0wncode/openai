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
type TTSPricing struct {
	PricePerCharacter  float64
	ApproxUSDPerMinute float64
	LimitCharacters    int
}

// DataTTS lists pricing information for text-to-speech models.
var DataTTS = map[string]TTSPricing{
	TTS11106: {
		PricePerCharacter:  0.00001500,
		ApproxUSDPerMinute: 0.01500,
		LimitCharacters:    16384,
	},
	TTS1HD: {
		PricePerCharacter:  0.00003000,
		ApproxUSDPerMinute: 0.03000,
		LimitCharacters:    16384,
	},
	TTS1HD1106: {
		PricePerCharacter:  0.00003000,
		ApproxUSDPerMinute: 0.03000,
		LimitCharacters:    16384,
	},
	GPT4oMiniTTS20250320: {
		PricePerCharacter:  0.00001500,
		ApproxUSDPerMinute: 0.01500,
		LimitCharacters:    16384,
	},
	GPT4oMiniTTS20251215: {
		PricePerCharacter:  0.00001500,
		ApproxUSDPerMinute: 0.01500,
		LimitCharacters:    16384,
	},
}
