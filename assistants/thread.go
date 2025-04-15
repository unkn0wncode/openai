package assistants

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"openai/chat"
	openai "openai/internal"
)

// Thread is an object that stores the state of a conversation with a user.
// Contains messages, can be used to create runs.
type Thread struct {
	objectFields
}

// AddMessage adds a message to the thread.
// "role" and "content" are required, fileIDs and meta can be nil.
// Only "user" role is supported, so it is filled automatically.
// Returns the created message.
func (t *Thread) AddMessage(input InputMessage) (*Message, error) {
	// validate input
	if input.Content == "" {
		return nil, fmt.Errorf("content is required")
	}
	if input.Metadata != nil {
		if err := input.Metadata.Validate(); err != nil {
			return nil, fmt.Errorf("invalid metadata: %w", err)
		}
	}
	if len(input.FileIDs) > 10 {
		return nil, fmt.Errorf("maximum of 10 file IDs can be attached to a message, but have %d", len(input.FileIDs))
	}
	input.Role = chat.RoleUser

	// create request
	b, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message data: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, threadsURL+"/"+t.ID+"/messages", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// unmarshal response
	var msg Message
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &msg, nil
}

// Messages fetches messages from the thread.
// limit is from 1 to 100, 20 by default.
// after is the message ID to start after, can be empty.
func (t *Thread) Messages(limit int, after string) (msgs []*Message, hasMore bool, err error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// create request
	url := threadsURL + "/" + t.ID + "/messages?limit=" + fmt.Sprint(limit)
	if after != "" {
		url += "&after=" + after
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	// unmarshal response
	var data struct {
		Data    []*Message `json:"data"`
		HasMore bool       `json:"has_more"`
		// other fields exist but are not used here
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return data.Data, data.HasMore, nil
}

// Run creates a new run on the thread.
// AssistantID is required, other fields can be empty.
func (t *Thread) Run(opts RunOptions) (*Run, error) {
	// validate input
	if opts.Metadata != nil {
		if err := opts.Metadata.Validate(); err != nil {
			return nil, fmt.Errorf("invalid metadata: %w", err)
		}
	}

	// create request
	b, err := json.Marshal(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run options: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, threadsURL+"/"+t.ID+"/runs", bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// unmarshal response
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &run, nil
}

// RunAndFetch adds messages to run (optionally), creates a new run, awaits its completion and fetches a new message from it.
// In case if run resolves into a different status than "completed", returns the run to handle it, and the message is nil.
func (t *Thread) RunAndFetch(ctx context.Context, opts RunOptions, msgs ...InputMessage) (*Run, *Message, error) {
	// add messages to thread
	for _, m := range msgs {
		if _, err := t.AddMessage(m); err != nil {
			return nil, nil, fmt.Errorf("failed to add message: %w", err)
		}
	}

	// create run
	run, err := t.Run(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a run: %w", err)
	}

	// await run
	if err := run.Await(ctx); err != nil {
		return run, nil, fmt.Errorf("failed to await run: %w", err)
	}

	// if functions are called, execute them and submit outputs
	for run.IsExpectingToolOutputs() {
		var outputs []ToolOutput

		// execute functions
		for _, tc := range run.RequiredAction.SubmitToolOutputs.ToolCalls {
			// get the function
			f, ok := chat.GetFunction(tc.Function.Name)
			if !ok {
				return run, nil, fmt.Errorf("called function '%s' which is not registered", tc.Function.Name)
			}
			if f.F == nil {
				return run, nil, fmt.Errorf("called function '%s' that has no implementation", tc.Function.Name)
			}

			// run the function
			fResult, err := f.F([]byte(tc.Function.Arguments))
			switch {
			case err == nil:
			case errors.Is(err, chat.ErrDoNotRespond):
				// in assistants, we can't just drop the run, or thread will stay locked
				outputs = append(outputs, ToolOutput{
					ToolCallID: tc.ID,
					Output:     "Function has executed successfully and indicated that no further messages should be generated at this time.",
				})
				continue
			default:
				return run, nil, fmt.Errorf("failed to execute function '%s': %w", tc.Function.Name, err)
			}
			openai.Log.Printf("Function '%s' returned: %s", tc.Function.Name, fResult)

			// append the output
			outputs = append(outputs, ToolOutput{
				ToolCallID: tc.ID,
				Output:     fResult,
			})
		}

		// submit tool outputs
		if err := run.SubmitToolOutputs(outputs...); err != nil {
			return run, nil, fmt.Errorf("failed to submit tool outputs: %w", err)
		}

		// await run
		if err := run.Await(ctx); err != nil {
			return run, nil, fmt.Errorf("failed to await run: %w", err)
		}
	}

	// check run status
	if run.Status != RunStatusCompleted {
		return run, nil, nil
	}

	// fetch message
	newMsgs, _, err := t.Messages(1, "")
	if err != nil {
		return run, nil, fmt.Errorf("failed to fetch message: %w", err)
	}

	return run, newMsgs[0], nil
}

// NewThread creates a new thread.
// Messages are optional.
// Metadata can be nil.
func NewThread(meta Metadata, messages ...InputMessage) (*Thread, error) {
	// validate input
	if meta != nil {
		if err := meta.Validate(); err != nil {
			return nil, fmt.Errorf("invalid metadata: %w", err)
		}
	}
	for i, m := range messages {
		if m.Content == "" {
			return nil, fmt.Errorf("content is required in messages")
		}
		if m.Metadata != nil {
			if err := m.Metadata.Validate(); err != nil {
				return nil, fmt.Errorf("invalid metadata in message '%s': %w", m.Content, err)
			}
		}
		if len(m.FileIDs) > 10 {
			return nil, fmt.Errorf("maximum of 10 file IDs can be attached to a message, but have %d in message '%s'", len(m.FileIDs), m.Content)
		}
		messages[i].Role = chat.RoleUser
	}

	// create request
	data := struct {
		Messages []InputMessage `json:"messages"`
		Metadata Metadata       `json:"metadata"`
	}{
		Messages: messages,
		Metadata: meta,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal thread data: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, threadsURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// unmarshal response
	var t Thread
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &t, nil
}

// LoadThread fetches a thread by its ID.
func LoadThread(id string) (*Thread, error) {
	// create request
	req, err := http.NewRequest(http.MethodGet, threadsURL+"/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// unmarshal response
	var t Thread
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &t, nil
}
