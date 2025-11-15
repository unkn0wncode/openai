// Package models / text.go contains list and properties of OpenAI text generation models.
package models

const (
	Default     = Latest
	DefaultMini = GPT5Mini
	DefaultNano = GPT5Nano
	Latest      = GPT51

	// GPT35Turbo-3.5 family
	GPT35Turbo = "gpt-3.5-turbo"
	GPTSnap    = "gpt-3.5-turbo-0125"
	GPT31106   = "gpt-3.5-turbo-1106"

	// GPT-4 family
	GPT4Turbo0125     = "gpt-4-0125-preview"
	GPT4TurboPreview  = "gpt-4-turbo-preview"
	GPT4Turbo         = "gpt-4-turbo"
	GPT4Turbo20240409 = "gpt-4-turbo-2024-04-09"

	GPT4Quasar             = "gpt-4.1"
	GPT4Quasar20250414     = "gpt-4.1-2025-04-14"
	GPT4QuasarMini         = "gpt-4.1-mini"
	GPT4QuasarMini20250414 = "gpt-4.1-mini-2025-04-14"
	GPT4QuasarNano         = "gpt-4.1-nano"
	GPT4QuasarNano20250414 = "gpt-4.1-nano-2025-04-14"

	// GPT-4o family
	GPT4Omni          = "gpt-4o"
	GPT4oLatest       = "chatgpt-4o-latest"
	GPT4o20240513     = "gpt-4o-2024-05-13"
	GPT4o20240806     = "gpt-4o-2024-08-06"
	GPT4o20241120     = "gpt-4o-2024-11-20"
	GPT4oMini         = "gpt-4o-mini"
	GPT4oMini20240718 = "gpt-4o-mini-2024-07-18"

	GPT4oSearchPreview               = "gpt-4o-search-preview"
	GPT4oSearchPreview20250311       = "gpt-4o-search-preview-2025-03-11"
	GPT4oMiniSearchPreview           = "gpt-4o-mini-search-preview"
	GPT4oMiniSearchPreview20250311   = "gpt-4o-mini-search-preview-2025-03-11"
	GPT4oRealtimePreview             = "gpt-4o-realtime-preview"
	GPT4oRealtimePreview20241001     = "gpt-4o-realtime-preview-2024-10-01"
	GPT4oRealtimePreview20241217     = "gpt-4o-realtime-preview-2024-12-17"
	GPT4oRealtimePreview20250603     = "gpt-4o-realtime-preview-2025-06-03"
	GPT4oMiniRealtimePreview         = "gpt-4o-mini-realtime-preview"
	GPT4oMiniRealtimePreview20241217 = "gpt-4o-mini-realtime-preview-2024-12-17"
	GPT4oAudioPreview                = "gpt-4o-audio-preview"
	GPT4oAudioPreview20241001        = "gpt-4o-audio-preview-2024-10-01"
	GPT4oAudioPreview20241217        = "gpt-4o-audio-preview-2024-12-17"
	GPT4oAudioPreview20250603        = "gpt-4o-audio-preview-2025-06-03"
	GPT4oMiniAudioPreview            = "gpt-4o-mini-audio-preview"
	GPT4oMiniAudioPreview20241217    = "gpt-4o-mini-audio-preview-2024-12-17"
	GPT4oTranscribe                  = "gpt-4o-transcribe"
	GPT4oTranscribeDiarize           = "gpt-4o-transcribe-diarize"
	GPT4oMiniTranscribe              = "gpt-4o-mini-transcribe"
	GPT4oMiniTTS                     = "gpt-4o-mini-tts"

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
	GPT51CodexMini        = "gpt-5.1-codex-mini"

	// Multimodal realtime & audio
	GPTRealtime             = "gpt-realtime"
	GPTRealtime20250828     = "gpt-realtime-2025-08-28"
	GPTRealtimeMini         = "gpt-realtime-mini"
	GPTRealtimeMini20251006 = "gpt-realtime-mini-2025-10-06"
	GPTAudio                = "gpt-audio"
	GPTAudio20250828        = "gpt-audio-2025-08-28"
	GPTAudioMini            = "gpt-audio-mini"
	GPTAudioMini20251006    = "gpt-audio-mini-2025-10-06"

	// O-series
	GPTO1                         = "o1"
	GPTO120241217                 = "o1-2024-12-17"
	GPTO1Mini                     = "o1-mini"
	GPTO1Mini20240912             = "o1-mini-2024-09-12"
	GPTO1Pro                      = "o1-pro"
	GPTO1Pro20250319              = "o1-pro-2025-03-19"
	GPTO3                         = "o3"
	GPTO320250416                 = "o3-2025-04-16"
	GPTO3Mini                     = "o3-mini"
	GPTO3Mini20250131             = "o3-mini-2025-01-31"
	GPTO3Pro                      = "o3-pro"
	GPTO3Pro20250610              = "o3-pro-2025-06-10"
	GPTO3DeepResearch             = "o3-deep-research"
	GPTO3DeepResearch20250626     = "o3-deep-research-2025-06-26"
	GPTO4Mini                     = "o4-mini"
	GPTO4Mini20250416             = "o4-mini-2025-04-16"
	GPTO4MiniDeepResearch         = "o4-mini-deep-research"
	GPTO4MiniDeepResearch20250626 = "o4-mini-deep-research-2025-06-26"

	// Tooling / moderation
	ComputerUsePreview         = "computer-use-preview"
	ComputerUsePreview20250311 = "computer-use-preview-2025-03-11"
	CodexMiniLatest            = "codex-mini-latest"

	// Completion models
	Curie           = "text-curie-001"
	Davinci3        = "davinci-002"
	Davinci         = "davinci"
	DavinciInstruct = "davinci-instruct-beta"
	GPTInstruct     = "gpt-3.5-turbo-instruct"
	GPTInstruct0914 = "gpt-3.5-turbo-instruct-0914"
	Babbage002      = "babbage-002"

	// Moderation defaults
	DefaultModeration = OmniMod
	OmniMod           = "omni-moderation-latest"
	OmniMod20240926   = "omni-moderation-2024-09-26"
	ModTextLatest     = "text-moderation-latest"
	ModTextStable     = "text-moderation-stable"
)

//go:generate go run ../internal/cmd/getmodels

// CODE BELOW THIS LINE IS GENERATED. ONLY EDIT IF YOU KNOW HOW.

// Data contains price per 1 token for each model, separately for input and output, and token limits.
// Note that pricing page https://openai.com/pricing lists price per 1k tokens and here it's per 1 token.
// The "" denotes default values.
var Data = map[string]struct {
	PriceIn       float64
	PriceCachedIn float64
	PriceOut      float64
	LimitContext  int
	LimitOutput   int
}{
	// Zeroes in the end of prices are added to align it and make it easier to read.
	// Can be read as "0.00000450 = 4.5 micro dollars per token = $4.50 per 1M tokens".
	"": {0.00000000, 0.00000000, 0.00000000, 4096, 4096},

	// GPT-3.5 family
	GPT35Turbo:      {0.00000050, 0.00000000, 0.00000150, 16385, 4096},
	GPTSnap:         {0.00000050, 0.00000050, 0.00000150, 16348, 4096},
	GPT31106:        {0.00000100, 0.00000100, 0.00000200, 16348, 4096},
	GPTInstruct:     {0.00000150, 0.00000000, 0.00000200, 16348, 4096},
	GPTInstruct0914: {0.00000150, 0.00000000, 0.00000200, 16348, 4096},

	// GPT-4 family
	"gpt-4":                {0.00003000, 0.00000000, 0.00006000, 8192, 8192},
	GPT4Turbo0125:          {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4TurboPreview:       {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4Turbo:              {0.00001000, 0.00000000, 0.00003000, 128000, 4096},
	GPT4Turbo20240409:      {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	"gpt-4-0613":           {0.00003000, 0.00003000, 0.00006000, 8192, 8192},
	"gpt-4-1106-preview":   {0.00001000, 0.00000000, 0.00003000, 128000, 4096},
	GPT4Quasar:             {0.00000200, 0.00000050, 0.00000800, 1047576, 32768},
	GPT4Quasar20250414:     {0.00000200, 0.00000050, 0.00000800, 1000000, 32768},
	GPT4QuasarMini:         {0.00000040, 0.00000010, 0.00000160, 1047576, 32768},
	GPT4QuasarMini20250414: {0.00000040, 0.00000010, 0.00000160, 1000000, 32768},
	GPT4QuasarNano:         {0.00000010, 0.00000003, 0.00000040, 1047576, 32768},
	GPT4QuasarNano20250414: {0.00000010, 0.00000003, 0.00000040, 1000000, 32768},

	// GPT-4o family
	GPT4Omni:                         {0.00000250, 0.00000125, 0.00001000, 128000, 16384},
	GPT4oLatest:                      {0.00000500, 0.00000000, 0.00001500, 128000, 4096},
	GPT4o20240513:                    {0.00000500, 0.00000000, 0.00001500, 128000, 4096},
	GPT4o20240806:                    {0.00000250, 0.00000125, 0.00001000, 128000, 16384},
	GPT4o20241120:                    {0.00000250, 0.00000125, 0.00001000, 128000, 16384},
	GPT4oMini:                        {0.00000015, 0.00000008, 0.00000060, 128000, 16384},
	GPT4oMini20240718:                {0.00000015, 0.00000008, 0.00000060, 128000, 16348},
	GPT4oSearchPreview:               {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oSearchPreview20250311:       {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oMiniSearchPreview:           {0.00000015, 0.00000000, 0.00000060, 128000, 16384},
	GPT4oMiniSearchPreview20250311:   {0.00000015, 0.00000000, 0.00000060, 128000, 16384},
	GPT4oRealtimePreview:             {0.00000500, 0.00000250, 0.00002000, 128000, 16384},
	GPT4oRealtimePreview20241001:     {0.00000500, 0.00000250, 0.00002000, 128000, 16384},
	GPT4oRealtimePreview20241217:     {0.00000500, 0.00000250, 0.00002000, 128000, 16384},
	GPT4oRealtimePreview20250603:     {0.00000500, 0.00000250, 0.00002000, 128000, 16384},
	GPT4oMiniRealtimePreview:         {0.00000060, 0.00000030, 0.00000240, 128000, 16384},
	GPT4oMiniRealtimePreview20241217: {0.00000060, 0.00000030, 0.00000240, 128000, 16384},
	GPT4oAudioPreview:                {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oAudioPreview20241001:        {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oAudioPreview20241217:        {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oAudioPreview20250603:        {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oMiniAudioPreview:            {0.00000015, 0.00000000, 0.00000060, 128000, 16384},
	GPT4oMiniAudioPreview20241217:    {0.00000015, 0.00000000, 0.00000060, 128000, 16384},
	GPT4oTranscribe:                  {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oTranscribeDiarize:           {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPT4oMiniTranscribe:              {0.00000125, 0.00000000, 0.00000500, 128000, 16384},
	GPT4oMiniTTS:                     {0.00000060, 0.00000000, 0.00001200, 128000, 16384},

	// GPT-5 family
	GPT5:                  {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT520250807:          {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5Mini:              {0.00000025, 0.00000003, 0.00000200, 400000, 128000},
	GPT5Mini20250807:      {0.00000025, 0.00000003, 0.00000200, 400000, 128000},
	GPT5Nano:              {0.00000005, 0.00000001, 0.00000040, 400000, 128000},
	GPT5Nano20250807:      {0.00000005, 0.00000001, 0.00000040, 400000, 128000},
	GPT5ChatLatest:        {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5Codex:             {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5Pro:               {0.00001500, 0.00000000, 0.00012000, 400000, 272000},
	GPT5Pro20251006:       {0.00001500, 0.00000000, 0.00012000, 400000, 272000},
	GPT5SearchAPI:         {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5SearchAPI20251014: {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT51:                 {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5120251113:         {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT51ChatLatest:       {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT51Codex:            {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT51CodexMini:        {0.00000025, 0.00000003, 0.00000200, 400000, 128000},

	// Multimodal realtime & audio
	GPTRealtime:             {0.00000400, 0.00000040, 0.00001600, 128000, 16384},
	GPTRealtime20250828:     {0.00000400, 0.00000040, 0.00001600, 128000, 16384},
	GPTRealtimeMini:         {0.00000060, 0.00000006, 0.00000240, 128000, 16384},
	GPTRealtimeMini20251006: {0.00000060, 0.00000006, 0.00000240, 128000, 16384},
	GPTAudio:                {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPTAudio20250828:        {0.00000250, 0.00000000, 0.00001000, 128000, 16384},
	GPTAudioMini:            {0.00000060, 0.00000000, 0.00000240, 128000, 16384},
	GPTAudioMini20251006:    {0.00000060, 0.00000000, 0.00000240, 128000, 16384},

	// O-series
	GPTO1:                         {0.00001500, 0.00000750, 0.00006000, 200000, 100000},
	GPTO120241217:                 {0.00001500, 0.00000750, 0.00006000, 200000, 100000},
	GPTO1Pro:                      {0.00015000, 0.00000000, 0.00060000, 200000, 100000},
	GPTO1Pro20250319:              {0.00015000, 0.00000000, 0.00060000, 200000, 100000},
	GPTO3:                         {0.00000200, 0.00000050, 0.00000800, 200000, 100000},
	GPTO320250416:                 {0.00000200, 0.00000050, 0.00000800, 200000, 100000},
	GPTO3Mini:                     {0.00000110, 0.00000055, 0.00000440, 200000, 100000},
	GPTO3Mini20250131:             {0.00000110, 0.00000055, 0.00000440, 200000, 100000},
	GPTO3Pro:                      {0.00002000, 0.00000000, 0.00008000, 200000, 100000},
	GPTO3Pro20250610:              {0.00002000, 0.00000000, 0.00008000, 200000, 100000},
	GPTO3DeepResearch:             {0.00001000, 0.00000250, 0.00004000, 200000, 100000},
	GPTO3DeepResearch20250626:     {0.00001000, 0.00000250, 0.00004000, 200000, 100000},
	GPTO4Mini:                     {0.00000110, 0.00000028, 0.00000440, 200000, 100000},
	GPTO4Mini20250416:             {0.00000110, 0.00000028, 0.00000440, 200000, 100000},
	GPTO4MiniDeepResearch:         {0.00000200, 0.00000050, 0.00000800, 200000, 100000},
	GPTO4MiniDeepResearch20250626: {0.00000200, 0.00000050, 0.00000800, 200000, 100000},

	// Tooling & moderation
	ComputerUsePreview:         {0.00000300, 0.00000000, 0.00001200, 128000, 16384},
	ComputerUsePreview20250311: {0.00000300, 0.00000000, 0.00001200, 128000, 16384},
	CodexMiniLatest:            {0.00000150, 0.00000038, 0.00000600, 200000, 100000},
	OmniMod:                    {0.00000000, 0.00000000, 0.00000000, 8192, 4096},
	OmniMod20240926:            {0.00000000, 0.00000000, 0.00000000, 8192, 4096},

	// Completion models
	Davinci3:   {0.00000200, 0.00000000, 0.00000200, 16384, 4096},
	Babbage002: {0.00000040, 0.00000000, 0.00000040, 16384, 4096},

	// Embedding models
	ThreeLarge: {0.00000013, 0.00000000, 0.00000000, 8191, 3072},
	ThreeSmall: {0.00000002, 0.00000000, 0.00000000, 8191, 1536},
}
