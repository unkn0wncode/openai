// Package models contains constants and pricing data for all OpenAI models.
package models

// Constant names are derived from the model ID:
//
//	GPT41          = "gpt-4.1"            // collapse dotted versions
//	GPT4o          = "gpt-4o"             // keep literal family tokens
//	O4Mini         = "o4-mini"            // o-series has no GPT prefix
//	GPT4o20241120  = "gpt-4o-2024-11-20"  // dates appended as YYYYMMDD
//	GPT41Mini      = "gpt-4.1-mini"       // suffixes stay PascalCase
//
// No marketing names (Quasar, Omni, etc). If there's no constant, use the
// model ID string directly.
const (
	Latest      = GPT6Astra
	Default     = Latest
	DefaultMini = GPT56Terra
	DefaultNano = GPT56Luna

	// Chat aliases
	ChatLatest = "chat-latest"

	// GPT-3.5 family
	GPT35Turbo             = "gpt-3.5-turbo"
	GPT35Turbo0125         = "gpt-3.5-turbo-0125"
	GPT35Turbo1106         = "gpt-3.5-turbo-1106"
	GPT35TurboInstruct     = "gpt-3.5-turbo-instruct"
	GPT35TurboInstruct0914 = "gpt-3.5-turbo-instruct-0914"

	// GPT-4 family
	GPT4Turbo         = "gpt-4-turbo"
	GPT4Turbo20240409 = "gpt-4-turbo-2024-04-09"

	// GPT-4.1 family
	GPT41             = "gpt-4.1"
	GPT4120250414     = "gpt-4.1-2025-04-14"
	GPT41Mini         = "gpt-4.1-mini"
	GPT41Mini20250414 = "gpt-4.1-mini-2025-04-14"
	GPT41Nano         = "gpt-4.1-nano"
	GPT41Nano20250414 = "gpt-4.1-nano-2025-04-14"

	// GPT-4o family
	GPT4o                          = "gpt-4o"
	GPT4o20240513                  = "gpt-4o-2024-05-13"
	GPT4o20240806                  = "gpt-4o-2024-08-06"
	GPT4o20241120                  = "gpt-4o-2024-11-20"
	GPT4oMini                      = "gpt-4o-mini"
	GPT4oMini20240718              = "gpt-4o-mini-2024-07-18"
	GPT4oSearchPreview             = "gpt-4o-search-preview"
	GPT4oSearchPreview20250311     = "gpt-4o-search-preview-2025-03-11"
	GPT4oMiniSearchPreview         = "gpt-4o-mini-search-preview"
	GPT4oMiniSearchPreview20250311 = "gpt-4o-mini-search-preview-2025-03-11"
	GPT4oTranscribe                = "gpt-4o-transcribe"
	GPT4oTranscribeDiarize         = "gpt-4o-transcribe-diarize"
	GPT4oMiniTranscribe            = "gpt-4o-mini-transcribe"
	GPT4oMiniTranscribe20250320    = "gpt-4o-mini-transcribe-2025-03-20"
	GPT4oMiniTranscribe20251215    = "gpt-4o-mini-transcribe-2025-12-15"
	GPT4oMiniTTS                   = "gpt-4o-mini-tts"

	// GPT-5 family
	GPT5                  = "gpt-5"
	GPT520250807          = "gpt-5-2025-08-07"
	GPT5Mini              = "gpt-5-mini"
	GPT5Mini20250807      = "gpt-5-mini-2025-08-07"
	GPT5Nano              = "gpt-5-nano"
	GPT5Nano20250807      = "gpt-5-nano-2025-08-07"
	GPT5ChatLatest        = "gpt-5-chat-latest"
	GPT5Codex             = "gpt-5-codex"
	GPT5Pro               = "gpt-5-pro"
	GPT5Pro20251006       = "gpt-5-pro-2025-10-06"
	GPT5SearchAPI         = "gpt-5-search-api"
	GPT5SearchAPI20251014 = "gpt-5-search-api-2025-10-14"
	GPT51                 = "gpt-5.1"
	GPT5120251113         = "gpt-5.1-2025-11-13"
	GPT51ChatLatest       = "gpt-5.1-chat-latest"
	GPT51Codex            = "gpt-5.1-codex"
	GPT51CodexMax         = "gpt-5.1-codex-max"
	GPT51CodexMini        = "gpt-5.1-codex-mini"
	GPT52                 = "gpt-5.2"
	GPT5220251211         = "gpt-5.2-2025-12-11"
	GPT52ChatLatest       = "gpt-5.2-chat-latest"
	GPT52Pro              = "gpt-5.2-pro"
	GPT52Pro20251211      = "gpt-5.2-pro-2025-12-11"
	GPT52Codex            = "gpt-5.2-codex"
	GPT53Codex            = "gpt-5.3-codex"
	GPT53ChatLatest       = "gpt-5.3-chat-latest"
	GPT54                 = "gpt-5.4"
	GPT5420260305         = "gpt-5.4-2026-03-05"
	GPT54Mini             = "gpt-5.4-mini"
	GPT54Mini20260317     = "gpt-5.4-mini-2026-03-17"
	GPT54Nano             = "gpt-5.4-nano"
	GPT54Nano20260317     = "gpt-5.4-nano-2026-03-17"
	GPT54Pro              = "gpt-5.4-pro"
	GPT54Pro20260305      = "gpt-5.4-pro-2026-03-05"
	GPT55                 = "gpt-5.5"
	GPT5520260423         = "gpt-5.5-2026-04-23"
	GPT55Pro              = "gpt-5.5-pro"
	GPT55Pro20260423      = "gpt-5.5-pro-2026-04-23"
	GPT56Sol              = "gpt-5.6-sol"
	GPT56Terra            = "gpt-5.6-terra"
	GPT56Luna             = "gpt-5.6-luna"

	// GPT-6 family
	GPT6Astra = "gpt-6-astra"

	// Multimodal realtime & audio
	GPTRealtime             = "gpt-realtime"
	GPTRealtime15           = "gpt-realtime-1.5"
	GPTRealtime2            = "gpt-realtime-2"
	GPTRealtime21           = "gpt-realtime-2.1"
	GPTRealtime21Mini       = "gpt-realtime-2.1-mini"
	GPTRealtime20250828     = "gpt-realtime-2025-08-28"
	GPTRealtimeMini         = "gpt-realtime-mini"
	GPTRealtimeMini20251215 = "gpt-realtime-mini-2025-12-15"
	GPTAudio                = "gpt-audio"
	GPTAudio15              = "gpt-audio-1.5"
	GPTAudio20250828        = "gpt-audio-2025-08-28"
	GPTAudioMini            = "gpt-audio-mini"
	GPTAudioMini20251006    = "gpt-audio-mini-2025-10-06"
	GPTAudioMini20251215    = "gpt-audio-mini-2025-12-15"

	// O-series
	O1                         = "o1"
	O120241217                 = "o1-2024-12-17"
	O1Mini                     = "o1-mini"
	O1Mini20240912             = "o1-mini-2024-09-12"
	O1Pro                      = "o1-pro"
	O1Pro20250319              = "o1-pro-2025-03-19"
	O3                         = "o3"
	O320250416                 = "o3-2025-04-16"
	O3Mini                     = "o3-mini"
	O3Mini20250131             = "o3-mini-2025-01-31"
	O3Pro                      = "o3-pro"
	O3Pro20250610              = "o3-pro-2025-06-10"
	O3DeepResearch             = "o3-deep-research"
	O3DeepResearch20250626     = "o3-deep-research-2025-06-26"
	O4Mini                     = "o4-mini"
	O4Mini20250416             = "o4-mini-2025-04-16"
	O4MiniDeepResearch         = "o4-mini-deep-research"
	O4MiniDeepResearch20250626 = "o4-mini-deep-research-2025-06-26"

	// Tooling
	ComputerUsePreview         = "computer-use-preview"
	ComputerUsePreview20250311 = "computer-use-preview-2025-03-11"

	// Completion models
	TextCurie001        = "text-curie-001"
	Davinci002          = "davinci-002"
	Davinci             = "davinci"
	DavinciInstructBeta = "davinci-instruct-beta"
	Babbage002          = "babbage-002"

	// Moderation
	DefaultModeration      = OmniModeration
	OmniModeration         = "omni-moderation-latest"
	OmniModeration20240926 = "omni-moderation-2024-09-26"
	TextModerationLatest   = "text-moderation-latest"
	TextModerationStable   = "text-moderation-stable"
)

// Data contains token prices and limits for each model.
// https://developers.openai.com/api/docs/pricing
// Older models have no additional cache-write charge; their writes use input rates.
// The empty model ID supplies default limits only and has no pricing.
var Data = map[string]Pricing{
	"": {LimitContext: 4096, LimitOutput: 4096},

	// Chat aliases
	ChatLatest: {
		standard:     &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 30}},
		LimitContext: 400000, LimitOutput: 128000,
	},

	// GPT-3.5 family
	GPT35Turbo: {
		standard:     &tierRates{short: tokenRates{input: 0.5, cachedInput: unavailableRate, cacheWrite: 0.5, output: 1.5}},
		LimitContext: 16385, LimitOutput: 4096,
	},
	GPT35Turbo0125: {
		standard:     &tierRates{short: tokenRates{input: 0.5, cachedInput: unavailableRate, cacheWrite: 0.5, output: 1.5}},
		LimitContext: 16348, LimitOutput: 4096,
	},
	GPT35Turbo1106: {
		standard:     &tierRates{short: tokenRates{input: 1, cachedInput: unavailableRate, cacheWrite: 1, output: 2}},
		LimitContext: 16348, LimitOutput: 4096,
	},
	GPT35TurboInstruct: {
		standard:     &tierRates{short: tokenRates{input: 1.5, cachedInput: unavailableRate, cacheWrite: 1.5, output: 2}},
		LimitContext: 16348, LimitOutput: 4096,
	},
	GPT35TurboInstruct0914: {
		standard:     &tierRates{short: tokenRates{input: 1.5, cachedInput: unavailableRate, cacheWrite: 1.5, output: 2}},
		LimitContext: 16348, LimitOutput: 4096,
	},

	// GPT-4 family
	"gpt-4": {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 60}},
		LimitContext: 8192, LimitOutput: 8192,
	},
	GPT4Turbo: {
		standard:     &tierRates{short: tokenRates{input: 10, cachedInput: unavailableRate, cacheWrite: 10, output: 30}},
		LimitContext: 128000, LimitOutput: 4096,
	},
	GPT4Turbo20240409: {
		standard:     &tierRates{short: tokenRates{input: 10, cachedInput: unavailableRate, cacheWrite: 10, output: 30}},
		LimitContext: 128000, LimitOutput: 4096,
	},
	"gpt-4-0613": {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 60}},
		LimitContext: 8192, LimitOutput: 8192,
	},

	// GPT-4.1 family
	GPT41: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.875, cacheWrite: 3.5, output: 14}},
		LimitContext: 1047576, LimitOutput: 32768,
	},
	GPT4120250414: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.875, cacheWrite: 3.5, output: 14}},
		LimitContext: 1000000, LimitOutput: 32768,
	},
	GPT41Mini: {
		standard:     &tierRates{short: tokenRates{input: 0.4, cachedInput: 0.1, cacheWrite: 0.4, output: 1.6}},
		fast:         &tierRates{short: tokenRates{input: 0.7, cachedInput: 0.175, cacheWrite: 0.7, output: 2.8}},
		LimitContext: 1047576, LimitOutput: 32768,
	},
	GPT41Mini20250414: {
		standard:     &tierRates{short: tokenRates{input: 0.4, cachedInput: 0.1, cacheWrite: 0.4, output: 1.6}},
		fast:         &tierRates{short: tokenRates{input: 0.7, cachedInput: 0.175, cacheWrite: 0.7, output: 2.8}},
		LimitContext: 1000000, LimitOutput: 32768,
	},
	GPT41Nano: {
		standard:     &tierRates{short: tokenRates{input: 0.1, cachedInput: 0.025, cacheWrite: 0.1, output: 0.4}},
		fast:         &tierRates{short: tokenRates{input: 0.2, cachedInput: 0.05, cacheWrite: 0.2, output: 0.8}},
		LimitContext: 1047576, LimitOutput: 32768,
	},
	GPT41Nano20250414: {
		standard:     &tierRates{short: tokenRates{input: 0.1, cachedInput: 0.025, cacheWrite: 0.1, output: 0.4}},
		fast:         &tierRates{short: tokenRates{input: 0.2, cachedInput: 0.05, cacheWrite: 0.2, output: 0.8}},
		LimitContext: 1000000, LimitOutput: 32768,
	},

	// GPT-4o family
	GPT4o: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: 1.25, cacheWrite: 2.5, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 4.25, cachedInput: 2.125, cacheWrite: 4.25, output: 17}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4o20240513: {
		standard:     &tierRates{short: tokenRates{input: 5, cachedInput: unavailableRate, cacheWrite: 5, output: 15}},
		fast:         &tierRates{short: tokenRates{input: 8.75, cachedInput: unavailableRate, cacheWrite: 8.75, output: 26.25}},
		LimitContext: 128000, LimitOutput: 4096,
	},
	GPT4o20240806: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: 1.25, cacheWrite: 2.5, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 4.25, cachedInput: 2.125, cacheWrite: 4.25, output: 17}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4o20241120: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: 1.25, cacheWrite: 2.5, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 4.25, cachedInput: 2.125, cacheWrite: 4.25, output: 17}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMini: {
		standard:     &tierRates{short: tokenRates{input: 0.15, cachedInput: 0.075, cacheWrite: 0.15, output: 0.6}},
		fast:         &tierRates{short: tokenRates{input: 0.25, cachedInput: 0.125, cacheWrite: 0.25, output: 1}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMini20240718: {
		standard:     &tierRates{short: tokenRates{input: 0.15, cachedInput: 0.075, cacheWrite: 0.15, output: 0.6}},
		fast:         &tierRates{short: tokenRates{input: 0.25, cachedInput: 0.125, cacheWrite: 0.25, output: 1}},
		LimitContext: 128000, LimitOutput: 16348,
	},
	GPT4oSearchPreview: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oSearchPreview20250311: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniSearchPreview: {
		standard:     &tierRates{short: tokenRates{input: 0.15, cachedInput: unavailableRate, cacheWrite: 0.15, output: 0.6}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniSearchPreview20250311: {
		standard:     &tierRates{short: tokenRates{input: 0.15, cachedInput: unavailableRate, cacheWrite: 0.15, output: 0.6}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oTranscribe: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oTranscribeDiarize: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniTranscribe: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: unavailableRate, cacheWrite: 1.25, output: 5}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniTranscribe20250320: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: unavailableRate, cacheWrite: 1.25, output: 5}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniTranscribe20251215: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: unavailableRate, cacheWrite: 1.25, output: 5}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT4oMiniTTS: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: unavailableRate, cacheWrite: 0.6, output: 12}},
		LimitContext: 128000, LimitOutput: 16384,
	},

	// GPT-5 family
	GPT5: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 20}},
		flex:         &tierRates{short: tokenRates{input: 0.625, cachedInput: 0.0625, cacheWrite: 0.625, output: 5}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT520250807: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 20}},
		flex:         &tierRates{short: tokenRates{input: 0.625, cachedInput: 0.0625, cacheWrite: 0.625, output: 5}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Mini: {
		standard:     &tierRates{short: tokenRates{input: 0.25, cachedInput: 0.025, cacheWrite: 0.25, output: 2}},
		fast:         &tierRates{short: tokenRates{input: 0.45, cachedInput: 0.045, cacheWrite: 0.45, output: 3.6}},
		flex:         &tierRates{short: tokenRates{input: 0.125, cachedInput: 0.0125, cacheWrite: 0.125, output: 1}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Mini20250807: {
		standard:     &tierRates{short: tokenRates{input: 0.25, cachedInput: 0.025, cacheWrite: 0.25, output: 2}},
		fast:         &tierRates{short: tokenRates{input: 0.45, cachedInput: 0.045, cacheWrite: 0.45, output: 3.6}},
		flex:         &tierRates{short: tokenRates{input: 0.125, cachedInput: 0.0125, cacheWrite: 0.125, output: 1}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Nano: {
		standard:     &tierRates{short: tokenRates{input: 0.05, cachedInput: 0.005, cacheWrite: 0.05, output: 0.4}},
		flex:         &tierRates{short: tokenRates{input: 0.025, cachedInput: 0.0025, cacheWrite: 0.025, output: 0.2}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Nano20250807: {
		standard:     &tierRates{short: tokenRates{input: 0.05, cachedInput: 0.005, cacheWrite: 0.05, output: 0.4}},
		flex:         &tierRates{short: tokenRates{input: 0.025, cachedInput: 0.0025, cacheWrite: 0.025, output: 0.2}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5ChatLatest: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Codex: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5Pro: {
		standard:     &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 120}},
		LimitContext: 400000, LimitOutput: 272000,
	},
	GPT5Pro20251006: {
		standard:     &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 120}},
		LimitContext: 400000, LimitOutput: 272000,
	},
	GPT5SearchAPI: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5SearchAPI20251014: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT51: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 20}},
		flex:         &tierRates{short: tokenRates{input: 0.625, cachedInput: 0.0625, cacheWrite: 0.625, output: 5}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5120251113: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		fast:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 20}},
		flex:         &tierRates{short: tokenRates{input: 0.625, cachedInput: 0.0625, cacheWrite: 0.625, output: 5}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT51ChatLatest: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT51Codex: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT51CodexMax: {
		standard:     &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.125, cacheWrite: 1.25, output: 10}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT51CodexMini: {
		standard:     &tierRates{short: tokenRates{input: 0.25, cachedInput: 0.025, cacheWrite: 0.25, output: 2}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT52: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.35, cacheWrite: 3.5, output: 28}},
		flex:         &tierRates{short: tokenRates{input: 0.875, cachedInput: 0.0875, cacheWrite: 0.875, output: 7}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT5220251211: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.35, cacheWrite: 3.5, output: 28}},
		flex:         &tierRates{short: tokenRates{input: 0.875, cachedInput: 0.0875, cacheWrite: 0.875, output: 7}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT52ChatLatest: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT52Pro: {
		standard:     &tierRates{short: tokenRates{input: 21, cachedInput: unavailableRate, cacheWrite: 21, output: 168}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT52Pro20251211: {
		standard:     &tierRates{short: tokenRates{input: 21, cachedInput: unavailableRate, cacheWrite: 21, output: 168}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT52Codex: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT53Codex: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.35, cacheWrite: 3.5, output: 28}},
		LimitContext: 400000, LimitOutput: 128000,
	},
	GPT53ChatLatest: {
		standard:     &tierRates{short: tokenRates{input: 1.75, cachedInput: 0.175, cacheWrite: 1.75, output: 14}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPT54: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 15}, long: &tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 22.5}},
		fast:         &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 30}},
		flex:         &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.13, cacheWrite: 1.25, output: 7.5}, long: &tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 11.25}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT5420260305: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 15}, long: &tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 22.5}},
		fast:         &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 30}},
		flex:         &tierRates{short: tokenRates{input: 1.25, cachedInput: 0.13, cacheWrite: 1.25, output: 7.5}, long: &tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 11.25}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT54Mini: {
		standard:     &tierRates{short: tokenRates{input: 0.75, cachedInput: 0.075, cacheWrite: 0.75, output: 4.5}},
		fast:         &tierRates{short: tokenRates{input: 1.5, cachedInput: 0.15, cacheWrite: 1.5, output: 9}},
		flex:         &tierRates{short: tokenRates{input: 0.375, cachedInput: 0.0375, cacheWrite: 0.375, output: 2.25}},
		LimitContext: 400000, LimitOutput: 128000,
		RegionalUplift: 0.1,
	},
	GPT54Mini20260317: {
		standard:     &tierRates{short: tokenRates{input: 0.75, cachedInput: 0.075, cacheWrite: 0.75, output: 4.5}},
		fast:         &tierRates{short: tokenRates{input: 1.5, cachedInput: 0.15, cacheWrite: 1.5, output: 9}},
		flex:         &tierRates{short: tokenRates{input: 0.375, cachedInput: 0.0375, cacheWrite: 0.375, output: 2.25}},
		LimitContext: 400000, LimitOutput: 128000,
		RegionalUplift: 0.1,
	},
	GPT54Nano: {
		standard:     &tierRates{short: tokenRates{input: 0.2, cachedInput: 0.02, cacheWrite: 0.2, output: 1.25}},
		flex:         &tierRates{short: tokenRates{input: 0.1, cachedInput: 0.01, cacheWrite: 0.1, output: 0.625}},
		LimitContext: 400000, LimitOutput: 128000,
		RegionalUplift: 0.1,
	},
	GPT54Nano20260317: {
		standard:     &tierRates{short: tokenRates{input: 0.2, cachedInput: 0.02, cacheWrite: 0.2, output: 1.25}},
		flex:         &tierRates{short: tokenRates{input: 0.1, cachedInput: 0.01, cacheWrite: 0.1, output: 0.625}},
		LimitContext: 400000, LimitOutput: 128000,
		RegionalUplift: 0.1,
	},
	GPT54Pro: {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 180}, long: &tokenRates{input: 60, cachedInput: unavailableRate, cacheWrite: 60, output: 270}},
		flex:         &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 90}, long: &tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 135}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT54Pro20260305: {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 180}, long: &tokenRates{input: 60, cachedInput: unavailableRate, cacheWrite: 60, output: 270}},
		flex:         &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 90}, long: &tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 135}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT55: {
		standard:     &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 30}, long: &tokenRates{input: 10, cachedInput: 1, cacheWrite: 10, output: 45}},
		fast:         &tierRates{short: tokenRates{input: 12.5, cachedInput: 1.25, cacheWrite: 12.5, output: 75}},
		flex:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 15}, long: &tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 22.5}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT5520260423: {
		standard:     &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 30}, long: &tokenRates{input: 10, cachedInput: 1, cacheWrite: 10, output: 45}},
		fast:         &tierRates{short: tokenRates{input: 12.5, cachedInput: 1.25, cacheWrite: 12.5, output: 75}},
		flex:         &tierRates{short: tokenRates{input: 2.5, cachedInput: 0.25, cacheWrite: 2.5, output: 15}, long: &tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 5, output: 22.5}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT55Pro: {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 180}, long: &tokenRates{input: 60, cachedInput: unavailableRate, cacheWrite: 60, output: 270}},
		flex:         &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 90}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT55Pro20260423: {
		standard:     &tierRates{short: tokenRates{input: 30, cachedInput: unavailableRate, cacheWrite: 30, output: 180}, long: &tokenRates{input: 60, cachedInput: unavailableRate, cacheWrite: 60, output: 270}},
		flex:         &tierRates{short: tokenRates{input: 15, cachedInput: unavailableRate, cacheWrite: 15, output: 90}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT56Sol: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 5, output: 20}, long: &tokenRates{input: 8, cachedInput: 0.8, cacheWrite: 10, output: 30}},
		fast:         &tierRates{short: tokenRates{input: 8, cachedInput: 0.8, cacheWrite: 10, output: 40}, long: &tokenRates{input: 16, cachedInput: 1.6, cacheWrite: 20, output: 60}},
		flex:         &tierRates{short: tokenRates{input: 2, cachedInput: 0.2, cacheWrite: 2.5, output: 10}, long: &tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 5, output: 15}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT56Terra: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.2, cacheWrite: 2.5, output: 12}, long: &tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 5, output: 18}},
		fast:         &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 5, output: 24}, long: &tokenRates{input: 8, cachedInput: 0.8, cacheWrite: 10, output: 36}},
		flex:         &tierRates{short: tokenRates{input: 1, cachedInput: 0.1, cacheWrite: 1.25, output: 6}, long: &tokenRates{input: 2, cachedInput: 0.2, cacheWrite: 2.5, output: 9}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},
	GPT56Luna: {
		standard:     &tierRates{short: tokenRates{input: 0.2, cachedInput: 0.02, cacheWrite: 0.25, output: 1.2}, long: &tokenRates{input: 0.4, cachedInput: 0.04, cacheWrite: 0.5, output: 1.8}},
		fast:         &tierRates{short: tokenRates{input: 0.4, cachedInput: 0.04, cacheWrite: 0.5, output: 2.4}, long: &tokenRates{input: 0.8, cachedInput: 0.08, cacheWrite: 1, output: 3.6}},
		flex:         &tierRates{short: tokenRates{input: 0.1, cachedInput: 0.01, cacheWrite: 0.125, output: 0.6}, long: &tokenRates{input: 0.2, cachedInput: 0.02, cacheWrite: 0.25, output: 0.9}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},

	// GPT-6 family
	GPT6Astra: {
		standard:     &tierRates{short: tokenRates{input: 10, cachedInput: 1, cacheWrite: 12.5, output: 50}, long: &tokenRates{input: 20, cachedInput: 2, cacheWrite: 25, output: 75}},
		fast:         &tierRates{short: tokenRates{input: 20, cachedInput: 2, cacheWrite: 25, output: 100}, long: &tokenRates{input: 40, cachedInput: 4, cacheWrite: 50, output: 150}},
		flex:         &tierRates{short: tokenRates{input: 5, cachedInput: 0.5, cacheWrite: 6.25, output: 25}, long: &tokenRates{input: 10, cachedInput: 1, cacheWrite: 12.5, output: 37.5}},
		LimitContext: 1050000, LimitOutput: 128000,
		LongContextThreshold: 272000,
		RegionalUplift:       0.1,
	},

	// Multimodal realtime & audio
	GPTRealtime: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 4, output: 16}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTRealtime15: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 4, output: 16}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTRealtime2: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 4, output: 24}},
		LimitContext: 128000, LimitOutput: 32000,
	},
	GPTRealtime21: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 4, output: 24}},
		LimitContext: 128000, LimitOutput: 32000,
	},
	GPTRealtime21Mini: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: 0.06, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 0, LimitOutput: 0,
	}, // official docs do not provide context/output limits
	GPTRealtime20250828: {
		standard:     &tierRates{short: tokenRates{input: 4, cachedInput: 0.4, cacheWrite: 4, output: 16}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTRealtimeMini: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: 0.06, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTRealtimeMini20251215: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: 0.06, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudio: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudio15: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudio20250828: {
		standard:     &tierRates{short: tokenRates{input: 2.5, cachedInput: unavailableRate, cacheWrite: 2.5, output: 10}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudioMini: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: unavailableRate, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudioMini20251006: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: unavailableRate, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	GPTAudioMini20251215: {
		standard:     &tierRates{short: tokenRates{input: 0.6, cachedInput: unavailableRate, cacheWrite: 0.6, output: 2.4}},
		LimitContext: 128000, LimitOutput: 16384,
	},

	// O-series
	O1: {
		standard:     &tierRates{short: tokenRates{input: 15, cachedInput: 7.5, cacheWrite: 15, output: 60}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O120241217: {
		standard:     &tierRates{short: tokenRates{input: 15, cachedInput: 7.5, cacheWrite: 15, output: 60}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O1Pro: {
		standard:     &tierRates{short: tokenRates{input: 150, cachedInput: unavailableRate, cacheWrite: 150, output: 600}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O1Pro20250319: {
		standard:     &tierRates{short: tokenRates{input: 150, cachedInput: unavailableRate, cacheWrite: 150, output: 600}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.875, cacheWrite: 3.5, output: 14}},
		flex:         &tierRates{short: tokenRates{input: 1, cachedInput: 0.25, cacheWrite: 1, output: 4}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O320250416: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		fast:         &tierRates{short: tokenRates{input: 3.5, cachedInput: 0.875, cacheWrite: 3.5, output: 14}},
		flex:         &tierRates{short: tokenRates{input: 1, cachedInput: 0.25, cacheWrite: 1, output: 4}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3Mini: {
		standard:     &tierRates{short: tokenRates{input: 1.1, cachedInput: 0.55, cacheWrite: 1.1, output: 4.4}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3Mini20250131: {
		standard:     &tierRates{short: tokenRates{input: 1.1, cachedInput: 0.55, cacheWrite: 1.1, output: 4.4}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3Pro: {
		standard:     &tierRates{short: tokenRates{input: 20, cachedInput: unavailableRate, cacheWrite: 20, output: 80}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3Pro20250610: {
		standard:     &tierRates{short: tokenRates{input: 20, cachedInput: unavailableRate, cacheWrite: 20, output: 80}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3DeepResearch: {
		standard:     &tierRates{short: tokenRates{input: 10, cachedInput: 2.5, cacheWrite: 10, output: 40}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O3DeepResearch20250626: {
		standard:     &tierRates{short: tokenRates{input: 10, cachedInput: 2.5, cacheWrite: 10, output: 40}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O4Mini: {
		standard:     &tierRates{short: tokenRates{input: 1.1, cachedInput: 0.275, cacheWrite: 1.1, output: 4.4}},
		fast:         &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		flex:         &tierRates{short: tokenRates{input: 0.55, cachedInput: 0.138, cacheWrite: 0.55, output: 2.2}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O4Mini20250416: {
		standard:     &tierRates{short: tokenRates{input: 1.1, cachedInput: 0.275, cacheWrite: 1.1, output: 4.4}},
		fast:         &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		flex:         &tierRates{short: tokenRates{input: 0.55, cachedInput: 0.138, cacheWrite: 0.55, output: 2.2}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O4MiniDeepResearch: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		LimitContext: 200000, LimitOutput: 100000,
	},
	O4MiniDeepResearch20250626: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: 0.5, cacheWrite: 2, output: 8}},
		LimitContext: 200000, LimitOutput: 100000,
	},

	// Tooling & moderation
	ComputerUsePreview: {
		standard:     &tierRates{short: tokenRates{input: 3, cachedInput: unavailableRate, cacheWrite: 3, output: 12}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	ComputerUsePreview20250311: {
		standard:     &tierRates{short: tokenRates{input: 3, cachedInput: unavailableRate, cacheWrite: 3, output: 12}},
		LimitContext: 128000, LimitOutput: 16384,
	},
	OmniModeration: {
		standard:     &tierRates{short: tokenRates{input: 0, cachedInput: 0, cacheWrite: 0, output: 0}},
		LimitContext: 8192, LimitOutput: 4096,
	},
	OmniModeration20240926: {
		standard:     &tierRates{short: tokenRates{input: 0, cachedInput: 0, cacheWrite: 0, output: 0}},
		LimitContext: 8192, LimitOutput: 4096,
	},

	// Completion models
	Davinci002: {
		standard:     &tierRates{short: tokenRates{input: 2, cachedInput: unavailableRate, cacheWrite: 2, output: 2}},
		LimitContext: 16384, LimitOutput: 4096,
	},
	Babbage002: {
		standard:     &tierRates{short: tokenRates{input: 0.4, cachedInput: unavailableRate, cacheWrite: 0.4, output: 0.4}},
		LimitContext: 16384, LimitOutput: 4096,
	},

	// Embedding models
	TextEmbedding3Large: {
		standard:     &tierRates{short: tokenRates{input: 0.13, cachedInput: unavailableRate, cacheWrite: 0.13, output: unavailableRate}},
		LimitContext: 8191, LimitOutput: 3072,
	},
	TextEmbedding3Small: {
		standard:     &tierRates{short: tokenRates{input: 0.02, cachedInput: unavailableRate, cacheWrite: 0.02, output: unavailableRate}},
		LimitContext: 8191, LimitOutput: 1536,
	},
}
