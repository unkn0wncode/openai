// Package openai / client.go provides a client for OpenAI APIs.
package openai

import (
	"openai/assistants"
	openai "openai/internal"
	assistantsInternal "openai/internal/assistants"
	"openai/internal/chat"
	"openai/internal/completion"
	"openai/internal/moderation"
)

// Client provides access to OpenAI APIs.
type Client struct {
	*chat.ChatClient
	*moderation.ModerationClient
	*completion.CompletionClient
	assistants.AssistantsService // assistant methods embedded for direct access

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

	return c
}
