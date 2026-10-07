// Package indecisions / client.go implements Decisions API requests.
package indecisions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/unkn0wncode/openai/decisions"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
)

// Client is the client for the Decisions API.
type Client struct {
	*openai.Config
}

// NewClient creates a new Decisions client.
func NewClient(config *openai.Config) *Client {
	return &Client{Config: config}
}

// interface compliance checks
var _ decisions.Service = (*Client)(nil)

// Send evaluates the request questions against its input.
// Refusals are returned as answers, not errors.
func (c *Client) Send(ctx context.Context, req *decisions.Request) (*decisions.Decision, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	data := *req
	if data.Model == "" {
		data.Model = models.DefaultDecision
	}
	for _, message := range data.Input {
		for _, part := range message {
			image, ok := part.(decisions.Base64Image)
			if !ok {
				continue
			}
			// Parse errors are returned by marshaling below.
			if declared, detected, err := image.MediaTypes(); err == nil && declared != "" && !strings.EqualFold(declared, detected) {
				c.Log.Warn(fmt.Sprintf("Decision image data URL declares type %q but contains %q; sending as %q", declared, detected, detected))
			}
		}
	}
	body, err := openai.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseAPI+"v1/decisions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	c.AddHeaders(httpReq)
	httpResp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer httpResp.Body.Close()
	body, err = io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s, body: %s", httpResp.Status, body)
	}
	var decision decisions.Decision
	if err := json.Unmarshal(body, &decision); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	decision.ProcessingRegion = openai.ProcessingRegion(httpReq.URL)
	c.logCost(&decision)
	for _, answer := range decision.Answers {
		switch answer.Type {
		case decisions.QuestionTypePredicate, decisions.QuestionTypeChoice, decisions.QuestionTypeScore:
		case decisions.AnswerTypeRefusal:
			c.Log.Warn(fmt.Sprintf("Decision question %q was refused", answer.Name))
		default:
			c.Log.Warn(fmt.Sprintf("Unexpected decision answer type %q", answer.Type))
		}
	}
	return &decision, nil
}

// logCost records usage and its estimated cost.
func (c *Client) logCost(decision *decisions.Decision) {
	message := "OpenAI Decisions usage unavailable"
	if decision.Usage != nil {
		message = fmt.Sprintf("Consumed OpenAI Decisions input tokens: %d", decision.Usage.InputTokens)
	}
	attrs := []any{slog.String("model", decision.Model), slog.String("region", decision.ProcessingRegion)}
	amount, err := decision.EstimateCost()
	switch {
	case err == nil:
		message += fmt.Sprintf(" (estimated cost $%.9f)", amount)
	case amount != 0:
		message += fmt.Sprintf(" (known cost subtotal $%.9f; estimate incomplete)", amount)
	default:
		message += " (cost estimate unavailable)"
	}
	if err != nil {
		attrs = append(attrs, slog.Any("costError", err))
	}
	c.Log.Debug(message, attrs...)
}
