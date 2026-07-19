// Package models contains constants and pricing data for all OpenAI models.
package models

import "github.com/unkn0wncode/openai/responses"

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
	Default     = GPT56Sol
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

	// Multimodal realtime & audio
	GPTRealtime             = "gpt-realtime"
	GPTRealtime15           = "gpt-realtime-1.5"
	GPTRealtime2            = "gpt-realtime-2"
	GPTRealtime21           = "gpt-realtime-2.1"
	GPTRealtime21Mini       = "gpt-realtime-2.1-mini"
	GPTRealtime20250828     = "gpt-realtime-2025-08-28"
	GPTRealtimeMini         = "gpt-realtime-mini"
	GPTRealtimeMini20251006 = "gpt-realtime-mini-2025-10-06"
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

// pricing contains token prices and limits for one model.
type pricing struct {
	PriceIn                    float64
	PriceCachedIn              float64
	PriceCacheWrite            float64
	PriceOut                   float64
	LongContextThreshold       int
	LongContextPriceIn         float64
	LongContextPriceCachedIn   float64
	LongContextPriceCacheWrite float64
	LongContextPriceOut        float64
	LimitContext               int
	LimitOutput                int
}

// newModelData returns pricing data whose cache-write price equals its input price.
func newModelData(priceIn, priceCachedIn, priceOut float64, limitContext, limitOutput int) pricing {
	return pricing{
		PriceIn:         priceIn,
		PriceCachedIn:   priceCachedIn,
		PriceCacheWrite: priceIn,
		PriceOut:        priceOut,
		LimitContext:    limitContext,
		LimitOutput:     limitOutput,
	}
}

// newCacheWriteModelData returns pricing data with a distinct cache-write price.
func newCacheWriteModelData(
	priceIn, priceCachedIn, priceCacheWrite, priceOut float64,
	limitContext, limitOutput int,
) pricing {
	data := newModelData(priceIn, priceCachedIn, priceOut, limitContext, limitOutput)
	data.PriceCacheWrite = priceCacheWrite
	return data
}

// newTieredModelData returns pricing data with separate long-context prices.
func newTieredModelData(
	priceIn, priceCachedIn, priceCacheWrite, priceOut float64,
	longContextThreshold int,
	longContextPriceIn, longContextPriceCachedIn, longContextPriceCacheWrite, longContextPriceOut float64,
	limitContext, limitOutput int,
) pricing {
	data := newCacheWriteModelData(priceIn, priceCachedIn, priceCacheWrite, priceOut, limitContext, limitOutput)
	data.LongContextThreshold = longContextThreshold
	data.LongContextPriceIn = longContextPriceIn
	data.LongContextPriceCachedIn = longContextPriceCachedIn
	data.LongContextPriceCacheWrite = longContextPriceCacheWrite
	data.LongContextPriceOut = longContextPriceOut
	return data
}

// Cost returns the Responses API request cost in USD.
func (data pricing) Cost(usage responses.Usage) float64 {
	priceIn := data.PriceIn
	priceCachedIn := data.PriceCachedIn
	priceCacheWrite := data.PriceCacheWrite
	priceOut := data.PriceOut
	if data.LongContextThreshold != 0 && usage.InputTokens > data.LongContextThreshold {
		priceIn = data.LongContextPriceIn
		priceCachedIn = data.LongContextPriceCachedIn
		priceCacheWrite = data.LongContextPriceCacheWrite
		priceOut = data.LongContextPriceOut
	}

	details := usage.InputTokensDetails
	uncachedInput := usage.InputTokens - details.CachedTokens - details.CacheWriteTokens
	return float64(uncachedInput)*priceIn +
		float64(details.CachedTokens)*priceCachedIn +
		float64(details.CacheWriteTokens)*priceCacheWrite +
		float64(usage.OutputTokens)*priceOut
}

// Data contains token prices and limits for each model.
// Note that pricing page https://openai.com/pricing lists price per 1M tokens and here it's per 1 token.
// The "" denotes default values.
var Data = map[string]pricing{
	// Zeroes in the end of prices are added to align it and make it easier to read.
	// Can be read as "0.00000450 = 4.5 micro dollars per token = $4.50 per 1M tokens".
	"": newModelData(0.00000000, 0.00000000, 0.00000000, 4096, 4096),

	// Chat aliases
	ChatLatest: newModelData(0.00000500, 0.00000050, 0.00003000, 400000, 128000),

	// GPT-3.5 family
	GPT35Turbo:             newModelData(0.00000050, 0.00000000, 0.00000150, 16385, 4096),
	GPT35Turbo0125:         newModelData(0.00000050, 0.00000050, 0.00000150, 16348, 4096),
	GPT35Turbo1106:         newModelData(0.00000100, 0.00000100, 0.00000200, 16348, 4096),
	GPT35TurboInstruct:     newModelData(0.00000150, 0.00000000, 0.00000200, 16348, 4096),
	GPT35TurboInstruct0914: newModelData(0.00000150, 0.00000000, 0.00000200, 16348, 4096),

	// GPT-4 family
	"gpt-4":           newModelData(0.00003000, 0.00000000, 0.00006000, 8192, 8192),
	GPT4Turbo:         newModelData(0.00001000, 0.00000000, 0.00003000, 128000, 4096),
	GPT4Turbo20240409: newModelData(0.00001000, 0.00001000, 0.00003000, 128000, 4096),
	"gpt-4-0613":      newModelData(0.00003000, 0.00003000, 0.00006000, 8192, 8192),

	// GPT-4.1 family
	GPT41:             newModelData(0.00000200, 0.00000050, 0.00000800, 1047576, 32768),
	GPT4120250414:     newModelData(0.00000200, 0.00000050, 0.00000800, 1000000, 32768),
	GPT41Mini:         newModelData(0.00000040, 0.00000010, 0.00000160, 1047576, 32768),
	GPT41Mini20250414: newModelData(0.00000040, 0.00000010, 0.00000160, 1000000, 32768),
	GPT41Nano:         newModelData(0.00000010, 0.00000003, 0.00000040, 1047576, 32768),
	GPT41Nano20250414: newModelData(0.00000010, 0.00000003, 0.00000040, 1000000, 32768),

	// GPT-4o family
	GPT4o:                          newModelData(0.00000250, 0.00000125, 0.00001000, 128000, 16384),
	GPT4o20240513:                  newModelData(0.00000500, 0.00000000, 0.00001500, 128000, 4096),
	GPT4o20240806:                  newModelData(0.00000250, 0.00000125, 0.00001000, 128000, 16384),
	GPT4o20241120:                  newModelData(0.00000250, 0.00000125, 0.00001000, 128000, 16384),
	GPT4oMini:                      newModelData(0.00000015, 0.00000008, 0.00000060, 128000, 16384),
	GPT4oMini20240718:              newModelData(0.00000015, 0.00000008, 0.00000060, 128000, 16348),
	GPT4oSearchPreview:             newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPT4oSearchPreview20250311:     newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPT4oMiniSearchPreview:         newModelData(0.00000015, 0.00000000, 0.00000060, 128000, 16384),
	GPT4oMiniSearchPreview20250311: newModelData(0.00000015, 0.00000000, 0.00000060, 128000, 16384),
	GPT4oTranscribe:                newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPT4oTranscribeDiarize:         newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPT4oMiniTranscribe:            newModelData(0.00000125, 0.00000000, 0.00000500, 128000, 16384),
	GPT4oMiniTranscribe20250320:    newModelData(0.00000125, 0.00000000, 0.00000500, 128000, 16384),
	GPT4oMiniTranscribe20251215:    newModelData(0.00000125, 0.00000000, 0.00000500, 128000, 16384),
	GPT4oMiniTTS:                   newModelData(0.00000060, 0.00000000, 0.00001200, 128000, 16384),

	// GPT-5 family
	GPT5:                  newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT520250807:          newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT5Mini:              newModelData(0.00000025, 0.00000003, 0.00000200, 400000, 128000),
	GPT5Mini20250807:      newModelData(0.00000025, 0.00000003, 0.00000200, 400000, 128000),
	GPT5Nano:              newModelData(0.00000005, 0.00000001, 0.00000040, 400000, 128000),
	GPT5Nano20250807:      newModelData(0.00000005, 0.00000001, 0.00000040, 400000, 128000),
	GPT5ChatLatest:        newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT5Codex:             newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT5Pro:               newModelData(0.00001500, 0.00000000, 0.00012000, 400000, 272000),
	GPT5Pro20251006:       newModelData(0.00001500, 0.00000000, 0.00012000, 400000, 272000),
	GPT5SearchAPI:         newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT5SearchAPI20251014: newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT51:                 newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT5120251113:         newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT51ChatLatest:       newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT51Codex:            newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT51CodexMax:         newModelData(0.00000125, 0.00000013, 0.00001000, 400000, 128000),
	GPT51CodexMini:        newModelData(0.00000025, 0.00000003, 0.00000200, 400000, 128000),
	GPT52:                 newModelData(0.00000175, 0.00000018, 0.00001400, 400000, 128000),
	GPT5220251211:         newModelData(0.00000175, 0.00000018, 0.00001400, 400000, 128000),
	GPT52ChatLatest:       newModelData(0.00000175, 0.00000018, 0.00001400, 128000, 16384),
	GPT52Pro:              newModelData(0.00002100, 0.00000000, 0.00016800, 400000, 128000),
	GPT52Pro20251211:      newModelData(0.00002100, 0.00000000, 0.00016800, 400000, 128000),
	GPT52Codex:            newModelData(0.00000175, 0.00000018, 0.00001400, 400000, 128000),
	GPT53Codex:            newModelData(0.00000175, 0.00000018, 0.00001400, 400000, 128000),
	GPT53ChatLatest:       newModelData(0.00000175, 0.00000018, 0.00001400, 128000, 16384),
	GPT54:                 newTieredModelData(0.00000250, 0.00000025, 0.00000250, 0.00001500, 272000, 0.00000500, 0.00000050, 0.00000500, 0.00002250, 1050000, 128000),
	GPT5420260305:         newTieredModelData(0.00000250, 0.00000025, 0.00000250, 0.00001500, 272000, 0.00000500, 0.00000050, 0.00000500, 0.00002250, 1050000, 128000),
	GPT54Mini:             newModelData(0.00000075, 0.00000008, 0.00000450, 400000, 128000),
	GPT54Mini20260317:     newModelData(0.00000075, 0.00000008, 0.00000450, 400000, 128000),
	GPT54Nano:             newModelData(0.00000020, 0.00000002, 0.00000125, 400000, 128000),
	GPT54Nano20260317:     newModelData(0.00000020, 0.00000002, 0.00000125, 400000, 128000),
	GPT54Pro:              newTieredModelData(0.00003000, 0.00000000, 0.00003000, 0.00018000, 272000, 0.00006000, 0.00000000, 0.00006000, 0.00027000, 1050000, 128000),
	GPT54Pro20260305:      newTieredModelData(0.00003000, 0.00000000, 0.00003000, 0.00018000, 272000, 0.00006000, 0.00000000, 0.00006000, 0.00027000, 1050000, 128000),
	GPT55:                 newTieredModelData(0.00000500, 0.00000050, 0.00000500, 0.00003000, 272000, 0.00001000, 0.00000100, 0.00001000, 0.00004500, 1050000, 128000),
	GPT5520260423:         newTieredModelData(0.00000500, 0.00000050, 0.00000500, 0.00003000, 272000, 0.00001000, 0.00000100, 0.00001000, 0.00004500, 1050000, 128000),
	GPT55Pro:              newTieredModelData(0.00003000, 0.00000000, 0.00003000, 0.00018000, 272000, 0.00006000, 0.00000000, 0.00006000, 0.00027000, 1050000, 128000),
	GPT55Pro20260423:      newTieredModelData(0.00003000, 0.00000000, 0.00003000, 0.00018000, 272000, 0.00006000, 0.00000000, 0.00006000, 0.00027000, 1050000, 128000),
	GPT56Sol:              newTieredModelData(0.00000500, 0.00000050, 0.00000625, 0.00003000, 272000, 0.00001000, 0.00000100, 0.00001250, 0.00004500, 1050000, 128000),
	GPT56Terra:            newTieredModelData(0.00000250, 0.00000025, 0.000003125, 0.00001500, 272000, 0.00000500, 0.00000050, 0.00000625, 0.00002250, 1050000, 128000),
	GPT56Luna:             newTieredModelData(0.00000100, 0.00000010, 0.00000125, 0.00000600, 272000, 0.00000200, 0.00000020, 0.00000250, 0.00000900, 1050000, 128000),

	// Multimodal realtime & audio
	GPTRealtime:             newModelData(0.00000400, 0.00000040, 0.00001600, 128000, 16384),
	GPTRealtime15:           newModelData(0.00000400, 0.00000040, 0.00001600, 128000, 16384),
	GPTRealtime2:            newModelData(0.00000400, 0.00000040, 0.00002400, 128000, 32000),
	GPTRealtime21:           newModelData(0.00000400, 0.00000040, 0.00002400, 128000, 32000),
	GPTRealtime21Mini:       newModelData(0.00000060, 0.00000006, 0.00000240, 0, 0), // official docs do not provide context/output limits
	GPTRealtime20250828:     newModelData(0.00000400, 0.00000040, 0.00001600, 128000, 16384),
	GPTRealtimeMini:         newModelData(0.00000060, 0.00000006, 0.00000240, 128000, 16384),
	GPTRealtimeMini20251006: newModelData(0.00000060, 0.00000006, 0.00000240, 128000, 16384),
	GPTRealtimeMini20251215: newModelData(0.00000060, 0.00000006, 0.00000240, 128000, 16384),
	GPTAudio:                newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPTAudio15:              newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPTAudio20250828:        newModelData(0.00000250, 0.00000000, 0.00001000, 128000, 16384),
	GPTAudioMini:            newModelData(0.00000060, 0.00000000, 0.00000240, 128000, 16384),
	GPTAudioMini20251006:    newModelData(0.00000060, 0.00000000, 0.00000240, 128000, 16384),
	GPTAudioMini20251215:    newModelData(0.00000060, 0.00000000, 0.00000240, 128000, 16384),

	// O-series
	O1:                         newModelData(0.00001500, 0.00000750, 0.00006000, 200000, 100000),
	O120241217:                 newModelData(0.00001500, 0.00000750, 0.00006000, 200000, 100000),
	O1Pro:                      newModelData(0.00015000, 0.00000000, 0.00060000, 200000, 100000),
	O1Pro20250319:              newModelData(0.00015000, 0.00000000, 0.00060000, 200000, 100000),
	O3:                         newModelData(0.00000200, 0.00000050, 0.00000800, 200000, 100000),
	O320250416:                 newModelData(0.00000200, 0.00000050, 0.00000800, 200000, 100000),
	O3Mini:                     newModelData(0.00000110, 0.00000055, 0.00000440, 200000, 100000),
	O3Mini20250131:             newModelData(0.00000110, 0.00000055, 0.00000440, 200000, 100000),
	O3Pro:                      newModelData(0.00002000, 0.00000000, 0.00008000, 200000, 100000),
	O3Pro20250610:              newModelData(0.00002000, 0.00000000, 0.00008000, 200000, 100000),
	O3DeepResearch:             newModelData(0.00001000, 0.00000250, 0.00004000, 200000, 100000),
	O3DeepResearch20250626:     newModelData(0.00001000, 0.00000250, 0.00004000, 200000, 100000),
	O4Mini:                     newModelData(0.00000110, 0.00000028, 0.00000440, 200000, 100000),
	O4Mini20250416:             newModelData(0.00000110, 0.00000028, 0.00000440, 200000, 100000),
	O4MiniDeepResearch:         newModelData(0.00000200, 0.00000050, 0.00000800, 200000, 100000),
	O4MiniDeepResearch20250626: newModelData(0.00000200, 0.00000050, 0.00000800, 200000, 100000),

	// Tooling & moderation
	ComputerUsePreview:         newModelData(0.00000300, 0.00000000, 0.00001200, 128000, 16384),
	ComputerUsePreview20250311: newModelData(0.00000300, 0.00000000, 0.00001200, 128000, 16384),
	OmniModeration:             newModelData(0.00000000, 0.00000000, 0.00000000, 8192, 4096),
	OmniModeration20240926:     newModelData(0.00000000, 0.00000000, 0.00000000, 8192, 4096),

	// Completion models
	Davinci002: newModelData(0.00000200, 0.00000000, 0.00000200, 16384, 4096),
	Babbage002: newModelData(0.00000040, 0.00000000, 0.00000040, 16384, 4096),

	// Embedding models
	TextEmbedding3Large: newModelData(0.00000013, 0.00000000, 0.00000000, 8191, 3072),
	TextEmbedding3Small: newModelData(0.00000002, 0.00000000, 0.00000000, 8191, 1536),
}
