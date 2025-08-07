// Package models / text.go contains list and properties of OpenAI text generation models.
package models

const (
	Default                = Latest
	DefaultMini            = GPT5Mini
	DefaultNano            = GPT5Nano
	Latest                 = GPT5
	GPT                    = "gpt-3.5-turbo"
	GPTSnap                = "gpt-3.5-turbo-0125"
	GPT4Vision             = "gpt-4-vision-preview"
	GPT4VisionLatest       = "gpt-4-1106-vision-preview"
	GPT4Turbo0125          = "gpt-4-0125-preview"
	GPT4TurboPreview       = "gpt-4-turbo-preview"
	GPT4Turbo              = "gpt-4-turbo"
	GPT4Turbo20240409      = "gpt-4-turbo-2024-04-09"
	GPT4Omni               = "gpt-4o"
	GPT4oLatest            = "chatgpt-4o-latest"
	GPT4Omni20240513       = "gpt-4o-2024-05-13"
	GPT4Omni20240806       = "gpt-4o-2024-08-06"
	GPT4Omni20241120       = "gpt-4o-2024-11-20"
	GPT4Quasar             = "gpt-4.1"
	GPT4Quasar20250414     = "gpt-4.1-2025-04-14"
	GPT4QuasarMini         = "gpt-4.1-mini"
	GPT4QuasarMini20250414 = "gpt-4.1-mini-2025-04-14"
	GPT4QuasarNano         = "gpt-4.1-nano"
	GPT4QuasarNano20250414 = "gpt-4.1-nano-2025-04-14"
	GPT4oMini              = "gpt-4o-mini"
	GPT4oMini20240718      = "gpt-4o-mini-2024-07-18"
	GPTO1                  = "o1"
	GPTO120241217          = "o1-2024-12-17"
	GPTO1Mini              = "o1-mini"
	GPTO1Mini20240912      = "o1-mini-2024-09-12"
	GPTO1Preview           = "o1-preview"
	GPTO1Preview20240912   = "o1-preview-2024-09-12"
	GPTO3Mini              = "o3-mini"
	GPTO3Mini20250131      = "o3-mini-2025-01-31"
	GPTO3                  = "o3"
	GPT5                   = "gpt-5"
	GPT5Mini               = "gpt-5-mini"
	GPT5Nano               = "gpt-5-nano"
	GPT5Chat               = "gpt-5-chat"
	GPT520250807           = "gpt-5-2025-08-07"
	GPT5Mini20250807       = "gpt-5-mini-2025-08-07"
	GPT5Nano20250807       = "gpt-5-nano-2025-08-07"
	GPT5Chat20250807       = "gpt-5-chat-2025-08-07"

	// Deprecated or unused models
	GPT45    = "gpt-4.5"
	GPT4564k = "gpt-4.5-64k"
	GPT4     = "gpt-4"
	GPT432k  = "gpt-4-32k"

	// Completion models
	Curie           = "text-curie-001"
	Davinci3        = "davinci-002"
	Davinci         = "davinci"
	DavinciInstruct = "davinci-instruct-beta"
	GPTInstruct     = "gpt-3.5-turbo-instruct"

	// Moderation models
	DefaultModeration = OmniMod
	ModTextLatest     = "text-moderation-latest"
	ModTextStable     = "text-moderation-stable"
	OmniMod           = "omni-moderation-latest"
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
	"":                     {0.00000000, 0.00000000, 0.00000000, 4096, 4096},
	GPT:                    {0.00000050, 0.00000050, 0.00000150, 16348, 4096},
	GPTSnap:                {0.00000050, 0.00000050, 0.00000150, 16348, 4096},
	GPT4Vision:             {0.00003000, 0.00003000, 0.00006000, 128000, 4096},
	GPT4VisionLatest:       {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4Turbo0125:          {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4TurboPreview:       {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4Turbo:              {0.00001000, 0.00000000, 0.00003000, 128000, 4096},
	GPT4Turbo20240409:      {0.00001000, 0.00001000, 0.00003000, 128000, 4096},
	GPT4Omni:               {0.00000250, 0.00000125, 0.00001000, 128000, 16384},
	GPT4oLatest:            {0.00000500, 0.00000500, 0.00001500, 128000, 4096},
	GPT4Omni20240513:       {0.00000500, 0.00000500, 0.00001500, 128000, 4096},
	GPT4Omni20240806:       {0.00000250, 0.00000125, 0.00001500, 128000, 16348},
	GPT4Omni20241120:       {0.00000250, 0.00000125, 0.00001500, 128000, 16348},
	GPT4Quasar:             {0.00000200, 0.00000050, 0.00000800, 1047576, 32768},
	GPT4Quasar20250414:     {0.00000200, 0.00000050, 0.00000800, 1000000, 32768},
	GPT4QuasarMini:         {0.00000040, 0.00000010, 0.00000160, 1047576, 32768},
	GPT4QuasarMini20250414: {0.00000040, 0.00000010, 0.00000160, 1000000, 32768},
	GPT4QuasarNano:         {0.00000010, 0.00000003, 0.00000040, 1047576, 32768},
	GPT4QuasarNano20250414: {0.00000010, 0.00000003, 0.00000040, 1000000, 32768},
	GPT4oMini:              {0.00000015, 0.00000008, 0.00000060, 128000, 16384},
	GPT4oMini20240718:      {0.00000015, 0.00000008, 0.00000060, 128000, 16348},
	GPTO1:                  {0.00001500, 0.00000750, 0.00006000, 200000, 100000},
	GPTO120241217:          {0.00001500, 0.00000750, 0.00006000, 200000, 100000},
	GPTO1Mini:              {0.00000110, 0.00000055, 0.00000440, 128000, 65536},
	GPTO1Mini20240912:      {0.00000110, 0.00000055, 0.00000440, 128000, 65536},
	GPTO1Preview:           {0.00001500, 0.00000750, 0.00006000, 128000, 32768},
	GPTO1Preview20240912:   {0.00001500, 0.00000750, 0.00006000, 128000, 32768},
	GPTO3Mini:              {0.00000110, 0.00000055, 0.00000440, 200000, 100000},
	GPTO3Mini20250131:      {0.00000110, 0.00000055, 0.00000440, 200000, 100000},
	GPTO3:                  {0.00000200, 0.00000050, 0.00000800, 200000, 100000},
	GPT5:                   {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5Mini:               {0.00000025, 0.00000003, 0.00000200, 400000, 128000},
	GPT5Nano:               {0.00000005, 0.00000001, 0.00000040, 400000, 128000},
	GPT5Chat:               {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT520250807:           {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT5Mini20250807:       {0.00000025, 0.00000003, 0.00000200, 400000, 128000},
	GPT5Nano20250807:       {0.00000005, 0.00000001, 0.00000040, 400000, 128000},
	GPT5Chat20250807:       {0.00000125, 0.00000013, 0.00001000, 400000, 128000},
	GPT45:                  {0.00006000, 0.00006000, 0.00018000, 8192, 4096},
	GPT4564k:               {0.00012000, 0.00012000, 0.00036000, 64000, 4096},
	GPT4:                   {0.00001000, 0.00000000, 0.00003000, 8192, 8192},
	GPT432k:                {0.00006000, 0.00006000, 0.00012000, 32000, 32768},
	"codex-mini-latest":    {0.00000150, 0.00000038, 0.00000600, 200000, 100000},
	"o3-pro":               {0.00002000, 0.00000000, 0.00008000, 200000, 100000},
	"o4-mini":              {0.00000110, 0.00000028, 0.00000440, 200000, 100000},
	"gpt-4.5-preview":      {0.00007500, 0.00003750, 0.00015000, 128000, 16384},
	"o1-pro":               {0.00015000, 0.00000000, 0.00060000, 200000, 100000},

	// Deprecated or unused models
	"gpt-4-0613":             {0.00003000, 0.00003000, 0.00006000, 8192, 8192},
	"gpt-4-0613-32k":         {0.00006000, 0.00006000, 0.00012000, 32000, 32768},
	"gpt-3.5-turbo-0613":     {0.00000150, 0.00000150, 0.00000200, 4096, 4096},
	"gpt-3.5-turbo-1106":     {0.00000100, 0.00000100, 0.00000200, 16348, 4096},
	"gpt-3.5-turbo-16k":      {0.00000300, 0.00000300, 0.00000400, 16348, 4096},
	"gpt-3.5-turbo-16k-0613": {0.00000300, 0.00000300, 0.00000400, 16348, 4096},
}
