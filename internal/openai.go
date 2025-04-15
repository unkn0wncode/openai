// Package openai / internal / openai.go provides common functions, constants and types for OpenAI API wrappers.
package openai

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"macbot/framework"
	"macbot/util"
	"macbot/util/servmon"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pkoukk/tiktoken-go"
)

var (
	logFileOpenai, _ = os.OpenFile("./openai/openai.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	logMultOpenaiStd = io.MultiWriter(logFileOpenai, os.Stdout)
	// Log is a logger for OpenAI system related events
	Log = log.New(logFileOpenai, "", log.Ldate|log.Ltime|log.Lshortfile)
	// LogStd is a logger for OpenAI-related events that also duplicates to console
	LogStd = log.New(logMultOpenaiStd, "", log.Ldate|log.Ltime|log.Lshortfile)

	Mon = servmon.New("OpenAI", 30*time.Minute, 0)
)

const (
	BaseAPI = "https://api.openai.com/"

	FinishReasonStop         = "stop"
	FinishReasonLength       = "length"
	FinishReasonFilter       = "content_filter"
	FinishReasonFunctionCall = "function_call"
	FinishReasonToolCalls    = "tool_calls"
	FinishReasonNull         = "null"

	// Deprecated: tiktoken is used for precise token counting instead of rough estimation based in characters count.
	// charPerToken = 4
)

// loggingTransport is a custom HTTP transport that logs request and response dumps.
type loggingTransport struct {
	Log *log.Logger
}

// LogTripper is a transport instance for logging HTTP requests and responses.
var LogTripper = &loggingTransport{
	// Log: LogStd,
}

// openAIClient is a wrapper for http.Client with OpenAI-specific behaviors.
type openAIClient struct {
	*http.Client
}

// Cli is an HTTP client to be used for OpenAI API requests.
var Cli = &openAIClient{
	Client: &http.Client{
		Timeout:   30 * time.Second,
		Transport: LogTripper,
	},
}

// RoundTrip logs the request and response while performing round trip, if logger is set.
func (lt *loggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	log := lt.Log
	if log == nil {
		return http.DefaultTransport.RoundTrip(r)
	}

	logBytes, dumpErr := util.Dump(r)
	if dumpErr != nil {
		return nil, fmt.Errorf("failed to dump request: %w", dumpErr)
	}

	resp, err := http.DefaultTransport.RoundTrip(r)
	// err is returned after dumping the response

	respBytes, dumpErr := util.Dump(resp)
	if dumpErr != nil {
		return nil, fmt.Errorf("failed to dump response: %w", dumpErr)
	}
	logBytes = append(logBytes, respBytes...)

	log.Printf("%s\n", logBytes)

	return resp, err
}

// Do performs the HTTP request but makes a copy of body beforehand and sets it back afterwards
// to allow retrying the same request multiple times with no data loss.
func (c *openAIClient) Do(req *http.Request) (*http.Response, error) {
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

	err := os.Setenv("TIKTOKEN_CACHE_DIR", "./openai/tiktoken")
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

// AddHeaders adds the basic required headers to given API request.
// Includes Authorization and Content-Type.
func AddHeaders(req *http.Request) {
	req.Header.Add("Authorization", "Bearer "+framework.Conf.OpenAIToken)
	req.Header.Add("Content-Type", "application/json")
}
