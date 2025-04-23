// Package openai / client.go provides a client for OpenAI APIs.
package openai

import (
	"github.com/unkn0wncode/openai/assistants"
	openai "github.com/unkn0wncode/openai/internal"
	assistantsInternal "github.com/unkn0wncode/openai/internal/assistants"
	"github.com/unkn0wncode/openai/internal/chat"
	"github.com/unkn0wncode/openai/internal/completion"
	"github.com/unkn0wncode/openai/internal/moderation"
	responsesInternal "github.com/unkn0wncode/openai/internal/responses"
	"github.com/unkn0wncode/openai/responses"
)

// Client provides access to OpenAI APIs.
type Client struct {
	*chat.ChatClient
	*moderation.ModerationClient
	*completion.CompletionClient
	assistants.AssistantsService
	responses.ResponsesService

	config *openai.Config
}

// NewClient creates a new OpenAI client with given token and default settings.
// Settings can be changed by accessing exported fields.
func NewClient(token string) *Client {
	c := &Client{config: openai.NewConfig(token)}
	c.ChatClient = &chat.ChatClient{Config: c.config}
	c.ModerationClient = &moderation.ModerationClient{Config: c.config}
	c.CompletionClient = &completion.CompletionClient{Config: c.config}
	c.AssistantsService = &assistantsInternal.AssistantsClient{Config: c.config}
	c.ResponsesService = &responsesInternal.ResponsesClient{Config: c.config}
	return c
}

func (c *Client) Config() *openai.Config {
	return c.config
}
