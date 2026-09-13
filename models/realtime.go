// Package models / realtime.go contains pricing for OpenAI realtime models billed by duration.
package models

const (
	GPTLive1             = "gpt-live-1"
	GPTTranscribe        = "gpt-transcribe"
	GPTLiveTranscribe    = "gpt-live-transcribe"
	GPTRealtimeTranslate = "gpt-realtime-translate"
	GPTRealtimeWhisper   = "gpt-realtime-whisper"
)

// RealtimeDurationPricing captures pricing and limits for models billed by duration.
type RealtimeDurationPricing struct {
	// PricePerMinute is in USD per minute of audio, except for session-priced entries noted below.
	PricePerMinute float64
	LimitContext   int
	LimitOutput    int
}

// DataRealtimeDuration lists models billed by audio or session duration instead of tokens.
// Backend model and tool charges are separate from session-duration charges.
var DataRealtimeDuration = map[string]RealtimeDurationPricing{
	// Zero limits mean the official docs do not provide a value.
	// Live 1 charges session duration per second, without rounding to whole minutes.
	GPTLive1:             {PricePerMinute: 0.05},
	GPTTranscribe:        {PricePerMinute: 0.00450},
	GPTLiveTranscribe:    {PricePerMinute: 0.01700},
	GPTRealtimeTranslate: {PricePerMinute: 0.03400, LimitContext: 16000, LimitOutput: 2000},
	GPTRealtimeWhisper:   {PricePerMinute: 0.01700, LimitContext: 16000, LimitOutput: 2000},
}
