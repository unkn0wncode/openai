// Package openai / client.go provides a client for OpenAI APIs.
package openai

import (
	openai "openai/internal"
	"openai/internal/chat"
)

// Client provides access to OpenAI APIs.
type Client struct {
	*chat.Client
	config *openai.Config
}

// NewClient creates a new OpenAI client with given token and default settings.
// Settings can be changed by accessing exported fields.
func NewClient(token string) *Client {
	return &Client{
		config: openai.NewConfig(token),
	}
}
