// Package completion provides a wrapper for the OpenAI Completion API.
package completion

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"openai/completion"
	openai "openai/internal"
)

const (
	apiURL = openai.BaseAPI + "v1/completions"

	maxTokens = 2048
)

// CompletionClient is a client for the OpenAI Completion API.
type CompletionClient struct {
	*openai.Config
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
func (c *CompletionClient) countTokens(data completion.Request) int {
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
func (c *CompletionClient) execute(data completion.Request) (*response, error) {
	if tokens := c.countTokens(data); tokens > maxTokens {
		return nil, fmt.Errorf("prompt is likely too long: total ~%d tokens, max %d tokens", tokens, maxTokens)
	}

	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	var req *http.Request
	req, err = http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"request failed with status: %s, body: %s",
			resp.Status, string(body),
		)
	}

	var res response
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	c.Log.Info(fmt.Sprintf(
		"Consumed OpenAI tokens: %d + %d = %d",
		res.Usage.Prompt, res.Usage.Completion, res.Usage.Total,
	))

	return &res, nil
}

// checkFirst checks if API response is valid, returns raw content and error.
func (c *CompletionClient) checkFirst(resp *response) (string, error) {
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
	c.Log.Info(fmt.Sprintf("OpenAI response: %s", content))

	return content, nil
}
