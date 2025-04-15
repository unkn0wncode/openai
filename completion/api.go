// Package completion provides a wrapper for the OpenAI Completion API.
package completion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	openai "macbot/openai/internal"
	"macbot/openai/openrouter"
	"net/http"
)

const (
	apiURL = openai.BaseAPI + "v1/completions"

	maxTokens = 2048
)

// Request is the request body for the Completions API
type Request struct {
	// required

	Model  string `json:"model"`  // model name: "text-curie-001"
	Prompt string `json:"prompt"` // text to be completed

	// optional

	// Alternative to Model. If set, the request will be sent to OpenRouter API instead of OpenAI.
	ModelOpenRouter string `json:"-"`

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

// MarshalJSON implements json.Marshaler interface.
// Passes model from OpenRouter field.
func (data Request) MarshalJSON() ([]byte, error) {
	if data.ModelOpenRouter != "" {
		data.Model = data.ModelOpenRouter
	}

	type Alias Request
	return json.Marshal(&struct{ Alias }{Alias: (Alias)(data)})
}

// response is the request body for the Completion API.
type response struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"` // Unix timestamp
	Model   string `json:"model"`
	Usage   struct {
		Prompt     int `json:"prompt_tokens"`
		Completion int `json:"completion_tokens"`
		Total      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Text         string          `json:"text"`
		FinishReason string          `json:"finish_reason"` // stop/length/content_filter/null
		Index        int             `json:"index"`
		Logprobs     json.RawMessage `json:"logprobs"` // actual type is unknown
	} `json:"choices"`
	Error struct {
		Message string          `json:"message"`
		Type    string          `json:"type"`
		Param   json.RawMessage `json:"param"` // actual type is unknown
		Code    json.RawMessage `json:"code"`  // actual type is unknown
	} `json:"error"`
}

// countTokens returns the number of tokens in the request.
func (data Request) countTokens() int {
	b, err := json.Marshal(data)
	if err != nil {
		panic("failed to marshal request body: " + err.Error())
	}

	if err := openai.LoadTokenEncoders(); err != nil {
		panic("failed to load token encoders: " + err.Error())
	}

	return len(openai.TokenEncoderCompletion.Encode(string(b), nil, nil))
}

// execute sends request to the Completion API and returns the response.
func (data Request) execute() (*response, error) {
	if tokens := data.countTokens(); tokens > maxTokens {
		return nil, fmt.Errorf("prompt is likely too long: total ~%d tokens, max %d tokens", tokens, maxTokens)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var req *http.Request
	if data.ModelOpenRouter == "" {
		req, err = http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(b))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		openai.AddHeaders(req)
	} else {
		req, err = openrouter.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(b))
		if err != nil {
			return nil, fmt.Errorf("failed to create openrouter request: %w", err)
		}
	}

	resp, err := openai.Cli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status: %s, body: %s", resp.Status, string(body))
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	openai.Log.Printf("Consumed OpenAI tokens: %d + %d = %d\n", res.Usage.Prompt, res.Usage.Completion, res.Usage.Total)

	return &res, nil
}

// checkFirst checks if API response is valid, returns raw content and error.
func (resp *response) checkFirst() (string, error) {
	if resp == nil {
		return "", fmt.Errorf("response is nil")
	}

	if resp.Error.Message != "" {
		return "", fmt.Errorf("got API error: %s", resp.Error.Message)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	finishReason := resp.Choices[0].FinishReason
	content := resp.Choices[0].Text
	if finishReason != openai.FinishReasonStop && finishReason != "" {
		return content, fmt.Errorf("got unexpected finish reason: %s", finishReason)
	}
	openai.Log.Println("OpenAI response:", content)

	return content, nil
}
