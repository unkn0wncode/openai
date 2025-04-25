// Package completion provides a wrapper for the OpenAI Completion API.
package completion

// Service defines methods to operate on completion API.
type Service interface {
	Completion(req Request) (string, error)
}

// Request is the request body for the Completions API
type Request struct {
	// required

	Model  string `json:"model"`  // model name: "text-curie-001"
	Prompt string `json:"prompt"` // text to be completed

	// optional

	// What sampling temperature to use, between 0 and 2.
	// Higher values like 0.8 will make the output more random, while lower values like 0.2 will make it more focused and deterministic.
	// We generally recommend altering this or top_p but not both.
	Temperature float64 `json:"temperature,omitempty"` // default 1

	// The maximum number of tokens allowed for the generated answer.
	// By default, the number of tokens the model can return will be (4096 - prompt tokens).
	MaxTokens int `json:"max_tokens,omitempty"` // default 4096 - prompt tokens

	// An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens with top_p probability mass.
	// So 0.1 means only the tokens comprising the top 10% probability mass are considered.
	// We generally recommend altering this or temperature but not both.
	TopP float64 `json:"top_p,omitempty"` // default 1

	// Number between -2.0 and 2.0.
	// Positive values penalize new tokens based on whether they appear in the text so far, increasing the model's likelihood to talk about new topics.
	PresencePenalty float64 `json:"presence_penalty,omitempty"` // default 0

	// Number between -2.0 and 2.0.
	// Positive values penalize new tokens based on their existing frequency in the text so far, decreasing the model's likelihood to repeat the same line verbatim.
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"` // default 0

	// Up to 4 sequences where the API will stop generating further tokens.
	Stop []string `json:"stop,omitempty"` // default []

	// A unique identifier representing your end-user, which can help OpenAI to monitor and detect abuse.
	User string `json:"user,omitempty"` // default ""

	BestOf int `json:"best_of,omitempty"` // default 1
}