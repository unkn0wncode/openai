package assistants

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"macbot/openai/chat"
	"net/http"
	"slices"
	"time"
)

// RunRefreshInterval is the interval between requests when waiting for a run to resolve.
// Can be modified externally, 3 sec default.
var RunRefreshInterval = 3 * time.Second

var (
	RunStatusQueued         = "queued"
	RunStatusInProgress     = "in_progress"
	RunStatusRequiresAction = "requires_action"
	RunStatusCancelling     = "cancelling"
	RunStatusCancelled      = "cancelled"
	RunStatusFailed         = "failed"
	RunStatusCompleted      = "completed"
	RunStatusExpired        = "expired"
)

// Run is an object that stores the state of a single execution of an assistant on a thread.
type Run struct {
	objectFields

	// The ID of the thread that was executed on as a part of this run.
	ThreadID string `json:"thread_id"`

	// The ID of the assistant used for execution of this run.
	AssistantID string `json:"assistant_id"`

	// The status of the run, which can be either queued, in_progress, requires_action, cancelling, cancelled, failed, completed, or expired.
	Status string `json:"status"`

	// Details on the action required to continue the run. Will be null if no action is required.
	RequiredAction struct {
		// For now, this is always submit_tool_outputs.
		Type string `json:"type"`

		// Details on the tool outputs needed for this run to continue.
		SubmitToolOutputs struct {
			// A list of the relevant tool calls.
			ToolCalls []chat.ToolCallData `json:"submit_tool_outputs"`
		} `json:"submit_tool_outputs"`
	} `json:"required_action"`

	// The last error associated with this run. Will be null if there are no errors.
	LastError struct {
		// One of server_error or rate_limit_exceeded.
		Code string `json:"code"`

		// A human-readable description of the error.
		Message string `json:"message"`
	} `json:"last_error"`

	// The Unix timestamp (in seconds) for when the run will expire.
	ExpiresAt int `json:"expires_at"`

	// The Unix timestamp (in seconds) for when the run was started.
	StartedAt int `json:"started_at"`

	// The Unix timestamp (in seconds) for when the run was cancelled.
	CancelledAt int `json:"cancelled_at"`

	// The Unix timestamp (in seconds) for when the run failed.
	FailedAt int `json:"failed_at"`

	// The Unix timestamp (in seconds) for when the run was completed.
	CompletedAt int `json:"completed_at"`

	// The model that the assistant used for this run.
	Model string `json:"model"`

	// The instructions that the assistant used for this run.
	Instructions string `json:"instructions"`

	// The list of tools that the assistant used for this run.
	Tools []Tool `json:"tools"`

	// The list of File IDs the assistant used for this run.
	FileIDs []string `json:"file_ids"`

	// Usage statistics related to the run.
	// This value will be null if the run is not in a terminal state (i.e. in_progress, queued, etc.).
	Usage struct {
		// Number of completion tokens used over the course of the run.
		CompletionTokens int `json:"completion_tokens"`

		// Number of prompt tokens used over the course of the run.
		PromptTokens int `json:"prompt_tokens"`

		// Total number of tokens used (prompt + completion).
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// RunOptions is a set of options that can be used to modify the behavior of a run on creation.
type RunOptions struct {
	// The ID of the assistant to use to execute this run.
	AssistantID string `json:"assistant_id"`

	// The ID of the Model to be used to execute this run.
	// If a value is provided here, it will override the model associated with the assistant.
	// If not, the model associated with the assistant will be used.
	Model string `json:"model"`

	// Overrides the instructions of the assistant.
	// This is useful for modifying the behavior on a per-run basis.
	Instructions string `json:"instructions"`

	// Appends additional instructions at the end of the instructions for the run.
	// This is useful for modifying the behavior on a per-run basis without overriding other instructions.
	AdditionalInstructions string `json:"additional_instructions"`

	// Override the tools the assistant can use for this run.
	// This is useful for modifying the behavior on a per-run basis.
	Tools []Tool `json:"tools"`

	Metadata Metadata `json:"metadata"`
}

// IsPending checks if the run status is pending, meaning that it is expected to change.
func (r Run) IsPending() bool {
	return slices.Contains(
		[]string{RunStatusQueued, RunStatusInProgress, RunStatusCancelling},
		r.Status,
	)
}

// IsExpectingToolOutputs checks if the run is expecting tool outputs to be submitted.
func (r Run) IsExpectingToolOutputs() bool {
	return r.Status == RunStatusRequiresAction && r.RequiredAction.Type == "submit_tool_outputs"
}

// SubmitToolOutputs sends results from tools to a halted run that is expecting them.
// Modifies the run with returned data.
func (r *Run) SubmitToolOutputs(outputs ...ToolOutput) error {
	// validate input
	if len(outputs) == 0 {
		return fmt.Errorf("at least one output is required")
	}
	for i, o := range outputs {
		if o.ToolCallID == "" {
			return fmt.Errorf("tool call ID is required in output %d", i)
		}
	}

	// create request
	data := struct {
		ThreadID    string       `json:"thread_id"`
		RunID       string       `json:"run_id"`
		ToolOutputs []ToolOutput `json:"tool_outputs"`
	}{
		ThreadID:    r.ThreadID,
		RunID:       r.ID,
		ToolOutputs: outputs,
	}
	b, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal tool outputs: %w", err)
	}
	url := fmt.Sprintf("%s/%s/runs/%s/submit_tool_outputs", threadsURL, r.ThreadID, r.ID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// execute request
	resp, err := executeRequest(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// unmarshal response
	err = json.NewDecoder(resp.Body).Decode(r)
	if err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// Await fetches run repeatedly until it is no longer pending, or until provided context expires.
// Refreshes run at least once to confirm its status.
func (r *Run) Await(ctx context.Context) error {
	// check if ctx is already expired
	if err := ctx.Err(); err != nil {
		return err
	}

	// start re-fetching the run
	for {
		// if thread is now pending and was set to such state less than refresh interval ago, wait for next refresh
		if r.IsPending() {
			updateTime := max(r.CreatedAt, r.StartedAt, r.CancelledAt, r.FailedAt, r.CompletedAt)
			if time.Since(time.Unix(int64(updateTime), 0)) < RunRefreshInterval {
				select {
				case <-time.After(RunRefreshInterval):
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}

		// refresh the run
		url := fmt.Sprintf("%s/%s/runs/%s", threadsURL, r.ThreadID, r.ID)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := executeRequest(req)
		if err != nil {
			return fmt.Errorf("failed to execute request: %w", err)
		}
		err = json.NewDecoder(resp.Body).Decode(r)
		if err != nil {
			resp.Body.Close()
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
		resp.Body.Close()

		// check if the run is still pending
		if !r.IsPending() {
			return nil
		}

		// wait until next refresh, or ctx expiry
		select {
		case <-time.After(RunRefreshInterval):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
