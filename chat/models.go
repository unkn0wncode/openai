// Package chat / models.go contains list and properties of chat models.
package chat

const (
	DefaultModel              = LatestModel
	DefaultMiniModel          = ModelGPT4QuasarMini
	DefaultNanoModel          = ModelGPT4QuasarNano
	LatestModel               = ModelGPT4Quasar
	ModelGPT                  = "gpt-3.5-turbo"
	ModelGPTSnap              = "gpt-3.5-turbo-0125"
	ModelGPT4Vision           = "gpt-4-vision-preview"
	ModelGPT4VisionLatest     = "gpt-4-1106-vision-preview"
	ModelGPT4Turbo0125        = "gpt-4-0125-preview"
	ModelGPT4TurboPreview     = "gpt-4-turbo-preview"
	ModelGPT4Turbo            = "gpt-4-turbo"
	ModelGPT4Turbo20240409    = "gpt-4-turbo-2024-04-09"
	ModelGPT4Omni             = "gpt-4o"
	ModelGPT4oLatest          = "chatgpt-4o-latest"
	ModelGPT4Omni20240513     = "gpt-4o-2024-05-13"
	ModelGPT4Omni20240806     = "gpt-4o-2024-08-06"
	ModelGPT4Omni20241120     = "gpt-4o-2024-11-20"
	ModelGPT4Quasar           = "gpt-4.1-2025-04-14"
	ModelGPT4QuasarMini       = "gpt-4.1-mini-2025-04-14"
	ModelGPT4QuasarNano       = "gpt-4.1-nano-2025-04-14"
	ModelGPT4oMini            = "gpt-4o-mini"
	ModelGPT4oMini20240718    = "gpt-4o-mini-2024-07-18"
	ModelGPTO1                = "o1"
	ModelGPTO120241217        = "o1-2024-12-17"
	ModelGPTO1Mini            = "o1-mini"
	ModelGPTO1Mini20240912    = "o1-mini-2024-09-12"
	ModelGPTO1Preview         = "o1-preview"
	ModelGPTO1Preview20240912 = "o1-preview-2024-09-12"
	ModelGPTO3Mini            = "o3-mini"
	ModelGPTO3Mini20250131    = "o3-mini-2025-01-31"

	// Deprecated or unused models
	ModelGPT45    = "gpt-4.5"
	ModelGPT4564k = "gpt-4.5-64k"
	ModelGPT4     = "gpt-4"
	ModelGPT432k  = "gpt-4-32k"
)

// ModelsData contains price per 1 token for each model, separately for input and output, and token limits.
// Note that pricing page https://openai.com/pricing lists price per 1k tokens and here it's per 1 token.
// The "" denotes default values.
var ModelsData = map[string]struct {
	PriceIn      float64
	PriceOut     float64
	LimitContext int
	LimitOutput  int
}{
	// Zeroes in the end of prices are added to align it and make it easier to read.
	// Can be read as "0.00000450 = 4.5 micro dollars per token = $4.50 per 1M tokens".
	"":                          {0.00000000, 0.00000000, 4096, 4096},
	"gpt-3.5-turbo-0125":        {0.00000050, 0.00000150, 16348, 4096},
	ModelGPT4:                   {0.00003000, 0.00006000, 8192, 4096},
	"gpt-4-0613":                {0.00003000, 0.00006000, 8192, 8192},
	ModelGPT4Vision:             {0.00003000, 0.00006000, 128000, 4096},
	"gpt-4-1106-vision-preview": {0.00001000, 0.00003000, 128000, 4096},
	ModelGPT4TurboPreview:       {0.00001000, 0.00003000, 128000, 4096},
	ModelGPT4Turbo:              {0.00001000, 0.00003000, 128000, 4096},
	ModelGPT4Turbo0125:          {0.00001000, 0.00003000, 128000, 4096},
	ModelGPT4Turbo20240409:      {0.00001000, 0.00003000, 128000, 4096},
	ModelGPT432k:                {0.00006000, 0.00012000, 32000, 32768},
	ModelGPT4Omni:               {0.00000500, 0.00001500, 128000, 4096},
	ModelGPT4Omni20240513:       {0.00000500, 0.00001500, 128000, 4096},
	ModelGPT4Omni20240806:       {0.00000500, 0.00001500, 128000, 16348},
	ModelGPT4Omni20241120:       {0.00000500, 0.00001500, 128000, 16348},
	ModelGPT4oMini:              {0.00000015, 0.00000060, 128000, 16348},
	ModelGPT4oMini20240718:      {0.00000015, 0.00000060, 128000, 16348},
	ModelGPT4QuasarMini:         {0.00000040, 0.00000160, 1000000, 32768},
	ModelGPT4QuasarNano:         {0.00000010, 0.00000040, 1000000, 32768},
	ModelGPT4Quasar:             {0.00000200, 0.00000800, 1000000, 32768},
	ModelGPT4oLatest:            {0.00000500, 0.00001500, 128000, 4096},
	"gpt-4-0613-32k":            {0.00006000, 0.00012000, 32000, 32768},
	ModelGPT45:                  {0.00006000, 0.00018000, 8192, 4096},  // unconfirmed
	ModelGPT4564k:               {0.00012000, 0.00036000, 64000, 4096}, // unconfirmed
	ModelGPT:                    {0.00000050, 0.00000150, 16348, 4096},
	ModelGPTO1:                  {0.00001500, 0.00006000, 200000, 100000},
	ModelGPTO120241217:          {0.00001500, 0.00006000, 200000, 100000},
	ModelGPTO1Mini:              {0.00000300, 0.00001200, 128000, 65536},
	ModelGPTO1Mini20240912:      {0.00000300, 0.00001200, 128000, 65536},
	ModelGPTO1Preview:           {0.00001500, 0.00006000, 128000, 32768},
	ModelGPTO1Preview20240912:   {0.00001500, 0.00006000, 128000, 32768},
	ModelGPTO3Mini:              {0.00000110, 0.00000440, 200000, 100000},
	ModelGPTO3Mini20250131:      {0.00000110, 0.00000440, 200000, 100000},

	// Deprecated or unused models
	"gpt-3.5-turbo-0613":     {0.0000015, 0.000002, 4096, 4096},
	"gpt-3.5-turbo-1106":     {0.0000010, 0.000002, 16348, 4096},
	"gpt-3.5-turbo-16k":      {0.0000030, 0.000004, 16348, 4096},
	"gpt-3.5-turbo-16k-0613": {0.0000030, 0.000004, 16348, 4096},
}
