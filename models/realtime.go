// Package models / realtime.go contains pricing for OpenAI realtime models billed by duration.
package models

const (
	GPTTranscribe        = "gpt-transcribe"
	GPTLiveTranscribe    = "gpt-live-transcribe"
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
	// Official docs do not provide context or output limits for these models.
	GPTTranscribe:        {0.00450, 0, 0},
	GPTLiveTranscribe:    {0.01700, 0, 0},
	GPTRealtimeTranslate: {0.03400, 16000, 2000},
	GPTRealtimeWhisper:   {0.01700, 16000, 2000},
}
