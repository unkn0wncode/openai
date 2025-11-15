// Package openai / internal / config.go provides configuration for OpenAI API clients.
package openai

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/unkn0wncode/openai/tools"
)

// Config contains configuration options for the OpenAI client.
type Config struct {
	// mutex is actually not used but protects other fields from copying by value
	_ sync.Mutex

	// BaseAPI is the base URL for OpenAI API endpoints
	BaseAPI    string
	Token      string
	HTTPClient *HTTPClient
	Log        *slog.Logger
	Tools      *tools.Registry
}

// NewConfig creates a default configuration with the provided token.
func NewConfig(token string) *Config {
	c := &Config{
		BaseAPI:    DefaultBaseAPI,
		Token:      token,
		HTTPClient: NewHTTPClient(),
		Log:        slog.Default(),
		Tools: &tools.Registry{
			FunctionCalls: make(map[string]tools.FunctionCall),
			Tools:         make(map[string]tools.Tool),
		},
	}

	c.HTTPClient.Transport.(*LoggingTransport).Log = c.Log

	return c
}

// AddHeaders adds the basic required headers to given API request.
// Includes Authorization and Content-Type.
func (c *Config) AddHeaders(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+c.Token)
	req.Header.Add("Content-Type", "application/json")
}

// EnableLogTripper turns on debug logging of HTTP requests and responses
// with slog instance from Config.
// Returns error if the expectation that HTTPClient has LoggingTransport is not met.
func (c *Config) EnableLogTripper() error {
	if c.HTTPClient == nil {
		return fmt.Errorf("HTTPClient is nil")
	}

	if c.HTTPClient.Transport == nil {
		return fmt.Errorf("HTTPClient.Transport is nil")
	}

	transport, ok := c.HTTPClient.Transport.(*LoggingTransport)
	if !ok {
		return fmt.Errorf(
			"transport is not a LoggingTransport but %T",
			c.HTTPClient.Transport,
		)
	}

	transport.EnableLog = true
	if c.Log != nil {
		transport.Log = c.Log
	}

	return nil
}

// DisableLogTripper turns off debug logging of HTTP requests and responses.
// Returns error if the expectation that HTTPClient has LoggingTransport is not met.
func (c *Config) DisableLogTripper() error {
	if c.HTTPClient == nil {
		return fmt.Errorf("HTTPClient is nil")
	}

	if c.HTTPClient.Transport == nil {
		return fmt.Errorf("HTTPClient.Transport is nil")
	}

	transport, ok := c.HTTPClient.Transport.(*LoggingTransport)
	if !ok {
		return fmt.Errorf(
			"transport is not a LoggingTransport but %T",
			c.HTTPClient.Transport,
		)
	}

	transport.EnableLog = false
	return nil
}
