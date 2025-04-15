// Package assistants provides functions for managing and interacting with OpenAI Assistants API.
// It uses chat models and some other chat elements as messaages and tools, so chats package has to be imported.
package assistants

import (
	"bytes"
	"fmt"
	"io"
	"macbot/openai/chat"
	openai "macbot/openai/internal"
	"macbot/util"
	"net/http"
	"regexp"
	"time"
)

const (
	assistantsURL = openai.BaseAPI + "v1/assistants"
	threadsURL    = openai.BaseAPI + "v1/threads"
	filesURL      = openai.BaseAPI + "v1/files"
)

// addHeaders adds the basic required headers to given API request.
// Includes Authorization, Content-Type and Beta features headers.
func addHeaders(req *http.Request) {
	openai.AddHeaders(req)
	req.Header.Add("OpenAI-Beta", "assistants=v1")
}

// objectFields is a set of fields that are common to all objects in the OpenAI Assistants API.
type objectFields struct {
	// The identifier, which can be referenced in API endpoints.
	ID string `json:"id"`

	// The object type.
	Object string `json:"object"`

	// The Unix timestamp (in seconds) for when the object was created.
	CreatedAt int `json:"created_at"`

	// Set of 16 key-value pairs that can be attached to an object.
	// This can be useful for storing additional information about the object in a structured format.
	// Keys can be a maximum of 64 characters long and values can be a maxium of 512 characters long.
	Metadata Metadata `json:"metadata"`
}

// Metadata is a user-defined map for storing additional information about an object.
// 16 key-value pairs at most.
// Keys can be a maximum of 64 characters long and values can be a maximum of 512 characters long.
type Metadata map[string]string

// Validate checks if metadata contents are within allowed limits.
func (m Metadata) Validate() error {
	if len(m) > 16 {
		return fmt.Errorf("metadata can have at most 16 key-value pairs, but has %d", len(m))
	}
	for k, v := range m {
		if len(k) > 64 {
			return fmt.Errorf("metadata key %q is too long (max 64 characters permitted, but have %d)", k, len(k))
		}
		if len(v) > 512 {
			return fmt.Errorf("metadata value for key %q is too long (max 512 characters permitted, but have %d)", k, len(v))
		}
	}
	return nil
}

// Tool is an instrument that an assistant can use during a run.
type Tool struct {
	chat.FunctionCall

	// Currently supported: code_interpreter, retrieval, function.
	Type string `json:"type"`
}

// ToolOutput is the output of a tool call to be submitted to continue the run.
type ToolOutput struct {
	// The ID of the tool call in the required_action object within the run object the output is being submitted for.
	ToolCallID string `json:"tool_call_id"`

	// The output of the tool call to be submitted to continue the run.
	Output string `json:"output"`
}

// executeRequest sends the given request and returns the response.
// Handles request headers, retries, timeouts, status errors.
func executeRequest(req *http.Request) (*http.Response, error) {
	// add headers in case if not already added
	addHeaders(req)

	// clear body from empty/null objects/arrays
	if req.Body != nil {
		var b bytes.Buffer
		_, err := io.Copy(&b, req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()

		repl := regexp.MustCompile(`,?"[^"]+":(\[\]|\{\}|null)`)
		b = *bytes.NewBuffer(repl.ReplaceAll(b.Bytes(), nil))
		req.Body = io.NopCloser(&b)
		req.ContentLength = int64(b.Len())
	}

	// send request
	var resp *http.Response
	err := util.Retry(func() error {
		var err error
		resp, err = openai.Cli.Do(req)
		return err
	}, 3, time.Second*3)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("request failed with status: %s, response body: %s", resp.Status, string(body))
	}

	return resp, nil
}
