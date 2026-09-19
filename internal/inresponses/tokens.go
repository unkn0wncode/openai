// Package inresponses / tokens.go implements input token counting for the Responses API.
package inresponses

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
)

// CountInputTokens uses the API's input counter without generating a response.
func (c *Client) CountInputTokens(ctx context.Context, req *responses.Request) (int, error) {
	if req == nil {
		return 0, errors.New("request is nil")
	}
	// These generation features are absent from the count endpoint's schema;
	// omitting them could count a different input than the one being compared.
	if req.Prompt != nil || len(req.ContextManagement) != 0 {
		return 0, errors.New("input token counting requires expanded input without prompt templates or context management")
	}
	data := req.Clone()
	if data.Model == "" {
		data.Model = models.Default
	}
	body, err := c.marshalRequest(data)
	if err != nil {
		return 0, fmt.Errorf("marshal token-count request: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return 0, err
	}
	for name := range fields {
		switch name {
		case "conversation", "input", "instructions", "model", "parallel_tool_calls", "personality",
			"previous_response_id", "reasoning", "text", "tool_choice", "tools", "truncation":
		default:
			delete(fields, name)
		}
	}
	body, err = json.Marshal(fields)
	if err != nil {
		return 0, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseAPI+"v1/responses/input_tokens", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create token-count request: %w", err)
	}
	c.AddHeaders(httpReq)
	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return 0, fmt.Errorf("count input tokens: %w", err)
	}
	defer httpResp.Body.Close()
	body, err = io.ReadAll(httpResp.Body)
	if err != nil {
		return 0, fmt.Errorf("read token count: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("token-count request failed with status %s, body: %s", httpResp.Status, body)
	}
	var result struct {
		Object      string `json:"object"`
		InputTokens *int   `json:"input_tokens"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("decode token count: %w", err)
	}
	if result.InputTokens == nil {
		return 0, errors.New("input token count is missing from response")
	}
	return *result.InputTokens, nil
}
