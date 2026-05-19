// Package models / realtime.go contains pricing for OpenAI realtime models billed by duration.
package models

const (
	GPTRealtimeTranslate = "gpt-realtime-translate"
	GPTRealtimeWhisper   = "gpt-realtime-whisper"
)

// RealtimeDurationPricing captures pricing and limits for realtime models billed by audio duration.
type RealtimeDurationPricing struct {
	// PricePerMinute is in USD.
	PricePerMinute float64
	LimitContext   int
	LimitOutput    int
}

// DataRealtimeDuration lists pricing for realtime models billed by audio duration instead of tokens.
var DataRealtimeDuration = map[string]RealtimeDurationPricing{
	GPTRealtimeTranslate: {0.03400, 16000, 2000},
	GPTRealtimeWhisper:   {0.01700, 16000, 2000},
}
