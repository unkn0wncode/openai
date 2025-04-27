// Package openai / internal / openai.go provides common functions, constants and types for OpenAI API wrappers.
package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/unkn0wncode/openai/util"

	"github.com/pkoukk/tiktoken-go"
)

// DefaultBaseAPI is the default base URL for OpenAI API endpoints.
const DefaultBaseAPI = "https://api.openai.com/"

// SupportedImageTypes is a list of supported image file extensions.
var SupportedImageTypes = []string{"png", "jpeg", "jpg", "gif", "webp"}

// LoggingTransport is a custom HTTP transport that logs request and response dumps.
type LoggingTransport struct {
	Log       *slog.Logger
	EnableLog bool
}

// HTTPClient is a wrapper for http.Client with OpenAI-specific behaviors.
type HTTPClient struct {
	*http.Client

	// Number of attempts to make the request. 2 means one attempt and one retry.
	RequestAttempts int

	// Interval to wait before each retry.
	RetryInterval time.Duration

	// If true, LogTripper is enabled on errors and disabled on successes.
	AutoLogTripper bool
}

// NewHTTPClient creates a new HTTPClient with default settings.
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		Client: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &LoggingTransport{},
		},
		RequestAttempts: 3,
		RetryInterval:   3 * time.Second,
	}
}

// RoundTrip logs the request and response while performing round trip, if logger is set.
func (lt *LoggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	log := lt.Log
	if log == nil || !lt.EnableLog {
		return http.DefaultTransport.RoundTrip(r)
	}

	reqBytes, dumpErr := util.Dump(r)
	if dumpErr != nil {
		return nil, fmt.Errorf("failed to dump request: %w", dumpErr)
	}

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, dumpErr := util.Dump(resp)
	if dumpErr != nil {
		return nil, fmt.Errorf("failed to dump response: %w", dumpErr)
	}

	log.Debug("request:\n%s\nresponse:\n%s", string(reqBytes), string(respBytes))

	return resp, err
}

// Do performs the HTTP request but makes a copy of body beforehand and sets it back afterwards
// to allow retrying the same request multiple times with no data loss.
func (c *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		return c.Client.Do(req)
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	resp, err := c.Client.Do(req)

	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return resp, err
}

// WithRetry performs the HTTP request and retries it if it fails (err or not 200),
// according to the settings.
func (c *HTTPClient) WithRetry(req *http.Request) (*http.Response, error) {
	restoreBody := func() {}
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		restoreBody = func() {
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}
	defer restoreBody()

	var resp *http.Response
	var err error
	err = util.Retry(func() error {
		defer restoreBody()

		before := time.Now()
		resp, err = c.Do(req)
		duration := time.Since(before)
		if err != nil {

			if c.AutoLogTripper {
				if lt, ok := c.Transport.(*LoggingTransport); ok && lt != nil {
					lt.EnableLog = true
				}
			}

			return fmt.Errorf("request failed in %v: %w", duration, err)
		}

		if resp.StatusCode != http.StatusOK {
			respBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf(
				"request failed in %v with status: %d, response body: %s",
				duration, resp.StatusCode, string(respBytes),
			)
		}

		if c.AutoLogTripper {
			if lt, ok := c.Transport.(*LoggingTransport); ok && lt != nil {
				lt.EnableLog = false
			}
		}

		return nil
	}, c.RequestAttempts, c.RetryInterval)

	return resp, err
}

// Encoders for counting tokens.
var (
	encodersMux            sync.Mutex
	TokenEncoderChat       *tiktoken.Tiktoken
	TokenEncoderCompletion *tiktoken.Tiktoken
)

// LoadTokenEncoders loads token encoders for chat and completion models.
// Uses cache directory.
// Can be called repeatedly but will not reload encoders if they are already successfully loaded.
func LoadTokenEncoders() error {
	encodersMux.Lock()
	defer encodersMux.Unlock()

	if TokenEncoderChat != nil && TokenEncoderCompletion != nil {
		return nil
	}

	err := os.Setenv("TIKTOKEN_CACHE_DIR", "./tiktoken")
	if err != nil {
		return fmt.Errorf("failed to set TIKTOK_CACHE_DIR env variable: %w", err)
	}

	TokenEncoderChat, err = tiktoken.GetEncoding(tiktoken.MODEL_CL100K_BASE)
	if err != nil {
		return fmt.Errorf("failed to get chat token encoder: %w", err)
	}
	TokenEncoderCompletion, err = tiktoken.GetEncoding(tiktoken.MODEL_P50K_BASE)
	if err != nil {
		return fmt.Errorf("failed to get completion token encoder: %w", err)
	}

	return nil
}

// Marshal marshals the given value to JSON.
// HTML escaping is disabled.
func Marshal(v any) ([]byte, error) {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}
