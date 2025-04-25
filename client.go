// Package openai / client.go provides a client for OpenAI APIs.
package openai

import (
	"github.com/unkn0wncode/openai/assistants"
	"github.com/unkn0wncode/openai/chat"
	"github.com/unkn0wncode/openai/completion"
	"github.com/unkn0wncode/openai/embedding"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/internal/inassistants"
	"github.com/unkn0wncode/openai/internal/inchat"
	"github.com/unkn0wncode/openai/internal/incompletion"
	"github.com/unkn0wncode/openai/internal/inembedding"
	"github.com/unkn0wncode/openai/internal/inmoderation"
	"github.com/unkn0wncode/openai/internal/inresponses"
	"github.com/unkn0wncode/openai/moderation"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/tools"
)

// Client provides access to OpenAI APIs.
type Client struct {
	Chat       chat.Service
	Moderation moderation.Service
	Completion completion.Service
	Assistants assistants.Service
	Responses  responses.Service
	Embedding  embedding.Service

	config *openai.Config
}

// NewClient creates a new OpenAI client with given token and default settings.
// Settings can be changed by accessing exported fields.
func NewClient(token string) *Client {
	c := &Client{config: openai.NewConfig(token)}
	c.Chat = inchat.NewClient(c.config)
	c.Moderation = inmoderation.NewClient(c.config)
	c.Completion = incompletion.NewClient(c.config)
	c.Assistants = inassistants.NewClient(c.config)
	c.Responses = inresponses.NewClient(c.config)
	c.Embedding = inembedding.NewClient(c.config)
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
