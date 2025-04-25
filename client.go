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
	"github.com/unkn0wncode/openai/tools"
)

// Client provides access to OpenAI APIs.
type Client struct {
	Chat       *chat.ChatClient
	Moderation *moderation.ModerationClient
	Completion *completion.CompletionClient
	Assistants assistants.AssistantsService
	Responses  responses.ResponsesService

	config *openai.Config
}

// NewClient creates a new OpenAI client with given token and default settings.
// Settings can be changed by accessing exported fields.
func NewClient(token string) *Client {
	c := &Client{config: openai.NewConfig(token)}
	c.Chat = &chat.ChatClient{Config: c.config}
	c.Moderation = &moderation.ModerationClient{Config: c.config}
	c.Completion = &completion.CompletionClient{Config: c.config}
	c.Assistants = &assistantsInternal.AssistantsClient{Config: c.config}
	c.Responses = &responsesInternal.ResponsesClient{Config: c.config}
	return c
}

// Config provides access to the client's configuration.
func (c *Client) Config() *openai.Config {
	return c.config
}

// Tools provides access to the client's tools registry.
func (c *Client) Tools() *tools.Registry {
	return c.config.Tools
}
