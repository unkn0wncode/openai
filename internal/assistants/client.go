package assistants

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"openai/assistants"
	"openai/content/output"
	openai "openai/internal"
	"openai/tools"
	"time"
)

const (
	assistantsURL = openai.BaseAPI + "v1/assistants"
	threadsURL    = openai.BaseAPI + "v1/threads"
	filesURL      = openai.BaseAPI + "v1/files"
)

// NewClient creates a new internal AssistantsClient with given configuration.
func NewClient(cfg *openai.Config) *AssistantsClient {
	return &AssistantsClient{
		Config:             cfg,
		RunRefreshInterval: 3 * time.Second,
	}
}

// AssistantsClient provides methods to manage assistants.
type AssistantsClient struct {
	*openai.Config

	// RunRefreshInterval is the interval between status polls in Await.
	RunRefreshInterval time.Duration
}

// type conformity checks
var (
	_ assistants.AssistantsService = &AssistantsClient{}
	_ assistants.Assistant         = &assistantHandle{}
	_ assistants.Thread            = &threadHandle{}
	_ assistants.Run               = &runHandle{}
)

// assistantDTO maps the JSON for an assistant object.
type assistantDTO struct {
	ID        string              `json:"id"`
	Object    string              `json:"object"`
	CreatedAt int                 `json:"created_at"`
	Metadata  assistants.Metadata `json:"metadata"`

	Model        string   `json:"model"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Instructions string   `json:"instructions"`
	FileIDs      []string `json:"file_ids"`
}

// assistantHandle implements the public Assistant interface.
type assistantHandle struct {
	client *AssistantsClient
	dto    assistantDTO
}

// addHeaders adds the necessary headers to the request.
func (c *AssistantsClient) addHeaders(req *http.Request) {
	c.AddHeaders(req)
	req.Header.Add("OpenAI-Beta", "assistants=v2")
}

// AssistantsRunRefreshInterval returns the interval between status polls in Await.
func (c *AssistantsClient) AssistantsRunRefreshInterval() time.Duration {
	return c.RunRefreshInterval
}

// SetAssistantsRunRefreshInterval sets the interval between status polls in Await.
func (c *AssistantsClient) SetAssistantsRunRefreshInterval(interval time.Duration) {
	c.RunRefreshInterval = interval
}

// CreateAssistant creates a new assistant with the given parameters.
func (c *AssistantsClient) CreateAssistant(params assistants.CreateParams) (assistants.Assistant, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, assistantsURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating assistant: %s %s", resp.Status, string(data))
	}

	var dto assistantDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}
	return &assistantHandle{client: c, dto: dto}, nil
}

// LoadAssistant fetches an assistant by ID.
func (c *AssistantsClient) LoadAssistant(id string) (assistants.Assistant, error) {
	req, err := http.NewRequest(http.MethodGet, assistantsURL+"/"+id, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error loading assistant: %s %s", resp.Status, string(data))
	}

	var dto assistantDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}
	return &assistantHandle{client: c, dto: dto}, nil
}

// ListAssistant fetches all assistants (single page up to 100).
func (c *AssistantsClient) ListAssistant() ([]assistants.Assistant, error) {
	req, err := http.NewRequest(http.MethodGet, assistantsURL+"?limit=100", nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error listing assistants: %s %s", resp.Status, string(data))
	}

	var page struct {
		Data []*assistantDTO `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return nil, err
	}

	out := make([]assistants.Assistant, len(page.Data))
	for i, dto := range page.Data {
		out[i] = &assistantHandle{client: c, dto: *dto}
	}
	return out, nil
}

// DeleteAssistant deletes an assistant by ID.
func (c *AssistantsClient) DeleteAssistant(id string) error {
	req, err := http.NewRequest(http.MethodDelete, assistantsURL+"/"+id, nil)
	if err != nil {
		return err
	}
	c.addHeaders(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error deleting assistant: %s %s", resp.Status, string(data))
	}

	var dto struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return err
	}
	if !dto.Deleted {
		return fmt.Errorf("assistant not deleted")
	}

	return nil
}

// ID returns the assistant ID.
func (h *assistantHandle) ID() string { return h.dto.ID }

// Model returns the assistant model.
func (h *assistantHandle) Model() string { return h.dto.Model }

// Name returns the assistant name.
func (h *assistantHandle) Name() string { return h.dto.Name }

// SetName is not implemented yet.
func (h *assistantHandle) SetName(name string) error { return fmt.Errorf("not implemented") }

// Description returns assistant description.
func (h *assistantHandle) Description() string { return h.dto.Description }

// SetDescription is not implemented yet.
func (h *assistantHandle) SetDescription(desc string) error { return fmt.Errorf("not implemented") }

// Instructions returns system instructions.
func (h *assistantHandle) Instructions() string { return h.dto.Instructions }

// SetInstructions is not implemented yet.
func (h *assistantHandle) SetInstructions(ins string) error { return fmt.Errorf("not implemented") }

// Tools returns the assistant's tools (unimplemented).
func (h *assistantHandle) Tools() []assistants.ToolSpec { return nil }

// AddTool is not implemented yet.
func (h *assistantHandle) AddTool(tool assistants.ToolSpec) error {
	return fmt.Errorf("not implemented")
}

// RemoveTool is not implemented yet.
func (h *assistantHandle) RemoveTool(name string) error { return fmt.Errorf("not implemented") }

// FileIDs returns attached file IDs.
func (h *assistantHandle) FileIDs() []string { return h.dto.FileIDs }

// AttachFile is not implemented yet.
func (h *assistantHandle) AttachFile(fileID string) error { return fmt.Errorf("not implemented") }

// DetachFile is not implemented yet.
func (h *assistantHandle) DetachFile(fileID string) error { return fmt.Errorf("not implemented") }

// assistant DTO and handle for threads
// threadDTO maps JSON for a Thread object.
type threadDTO struct {
	ID          string              `json:"id"`
	Object      string              `json:"object"`
	CreatedAt   int                 `json:"created_at"`
	Metadata    assistants.Metadata `json:"metadata"`
	AssistantID string              `json:"assistant_id"`
}

// threadHandle implements the public Thread interface.
type threadHandle struct {
	client *AssistantsClient
	dto    threadDTO
}

// NewThread creates a new thread under this assistant.
func (h *assistantHandle) NewThread(meta assistants.Metadata, messages ...assistants.InputMessage) (assistants.Thread, error) {
	// no client-side validation on metadata; server will enforce limits
	// prepare payload
	data := struct {
		Messages []assistants.InputMessage `json:"messages"`
		Metadata assistants.Metadata       `json:"metadata"`
	}{Messages: messages, Metadata: meta}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, threadsURL, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	h.client.addHeaders(req)
	resp, err := h.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating thread: %s %s", resp.Status, string(d))
	}
	var dto threadDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}
	// preserve assistant_id for run operations
	dto.AssistantID = h.dto.ID
	return &threadHandle{client: h.client, dto: dto}, nil
}

// LoadThread fetches an existing thread by ID under this assistant.
func (h *assistantHandle) LoadThread(id string) (assistants.Thread, error) {
	req, err := http.NewRequest(http.MethodGet, threadsURL+"/"+id, nil)
	if err != nil {
		return nil, err
	}
	h.client.addHeaders(req)
	resp, err := h.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error loading thread: %s %s", resp.Status, string(d))
	}
	var dto threadDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}
	// preserve assistant_id for run operations
	dto.AssistantID = h.dto.ID
	return &threadHandle{client: h.client, dto: dto}, nil
}

// ID returns the thread ID.
func (t *threadHandle) ID() string { return t.dto.ID }

// AddMessage adds a user message to the thread.
func (t *threadHandle) AddMessage(msg assistants.InputMessage) (assistants.Message, error) {
	b, err := json.Marshal(msg)
	if err != nil {
		return assistants.Message{}, err
	}
	req, err := http.NewRequest(http.MethodPost, threadsURL+"/"+t.dto.ID+"/messages", bytes.NewBuffer(b))
	if err != nil {
		return assistants.Message{}, err
	}
	t.client.addHeaders(req)
	resp, err := t.client.HTTPClient.Do(req)
	if err != nil {
		return assistants.Message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		return assistants.Message{}, fmt.Errorf("error adding message: %s %s", resp.Status, string(d))
	}
	var m assistants.Message
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return assistants.Message{}, fmt.Errorf("failed to decode message: %w", err)
	}
	return m, nil
}

// Messages lists messages in the thread.
func (t *threadHandle) Messages(limit int, after string) ([]assistants.Message, bool, error) {
	if limit < 1 || limit > 100 {
		limit = 20
	}
	url := fmt.Sprintf("%s/%s/messages?limit=%d", threadsURL, t.dto.ID, limit)
	if after != "" {
		url += "&after=" + after
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	t.client.addHeaders(req)
	resp, err := t.client.HTTPClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		return nil, false, fmt.Errorf("error fetching messages: %s %s", resp.Status, string(d))
	}
	// parse raw messages with content.Any to preserve typed array
	var payload struct {
		Data []struct {
			Role    string       `json:"role"`
			Content []output.Any `json:"content"`
		} `json:"data"`
		HasMore bool `json:"has_more"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, false, fmt.Errorf("failed to decode messages: %w", err)
	}
	out := make([]assistants.Message, len(payload.Data))
	for i, m := range payload.Data {
		out[i] = assistants.Message{Role: m.Role, Content: m.Content}
	}
	return out, payload.HasMore, nil
}

// runDTO maps JSON for a run object.
type runDTO struct {
	ID             string `json:"id"`
	ThreadID       string `json:"thread_id"`
	AssistantID    string `json:"assistant_id"`
	Status         string `json:"status"`
	CompletedAt    int    `json:"completed_at"`
	RequiredAction struct {
		Type              string `json:"type"`
		SubmitToolOutputs struct {
			ToolCalls []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"submit_tool_outputs"`
		} `json:"submit_tool_outputs"`
	} `json:"required_action"`
}

// runHandle implements the public Run interface, storing run state.
type runHandle struct {
	client *AssistantsClient
	dto    runDTO
}

// ID returns the run ID.
func (r *runHandle) ID() string { return r.dto.ID }

// SubmitToolOutputs sends tool call results and refreshes the run state.
func (r *runHandle) SubmitToolOutputs(outputs ...assistants.ToolOutput) error {
	// prepare payload
	data := struct {
		ThreadID    string                  `json:"thread_id"`
		RunID       string                  `json:"run_id"`
		ToolOutputs []assistants.ToolOutput `json:"tool_outputs"`
	}{r.dto.ThreadID, r.dto.ID, outputs}
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	turl := fmt.Sprintf("%s/%s/runs/%s/submit_tool_outputs", threadsURL, r.dto.ThreadID, r.dto.ID)
	req, err := http.NewRequest(http.MethodPost, turl, bytes.NewBuffer(b))
	if err != nil {
		return err
	}
	r.client.addHeaders(req)
	resp, err := r.client.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("error submitting tool outputs: %s %s", resp.Status, string(data))
	}
	// update state
	if err := json.NewDecoder(resp.Body).Decode(&r.dto); err != nil {
		return err
	}
	return nil
}

// IsPending reports if the run is in a pending state.
func (r *runHandle) IsPending() bool {
	return r.dto.Status == "queued" || r.dto.Status == "in_progress" || r.dto.Status == "cancelling"
}

// IsExpectingToolOutputs reports if the run is awaiting tool outputs.
func (r *runHandle) IsExpectingToolOutputs() bool {
	return r.dto.Status == "requires_action" && r.dto.RequiredAction.Type == "submit_tool_outputs"
}

// Await polls the run until it leaves the pending states or context expires.
func (r *runHandle) Await(ctx context.Context) error {
	for r.IsPending() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.client.RunRefreshInterval):
		}
		// refresh run state
		runURL := fmt.Sprintf("%s/%s/runs/%s", threadsURL, r.dto.ThreadID, r.dto.ID)
		req, err := http.NewRequest(http.MethodGet, runURL, nil)
		if err != nil {
			return err
		}
		r.client.addHeaders(req)
		resp, err := r.client.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("error refreshing run: %s %s", resp.Status, string(body))
		}
		if err := json.NewDecoder(resp.Body).Decode(&r.dto); err != nil {
			return err
		}
	}
	return nil
}

// Run creates a new run on this thread.
func (t *threadHandle) Run(opts *assistants.RunOptions) (assistants.Run, error) {
	if opts == nil {
		opts = &assistants.RunOptions{}
	}
	// prepare run creation payload with required assistant_id
	payload := struct {
		AssistantID            string                `json:"assistant_id"`
		Model                  string                `json:"model,omitempty"`
		Instructions           string                `json:"instructions,omitempty"`
		AdditionalInstructions string                `json:"additional_instructions,omitempty"`
		Tools                  []assistants.ToolSpec `json:"tools,omitempty"`
		Metadata               assistants.Metadata   `json:"metadata,omitempty"`
	}{
		AssistantID:            t.dto.AssistantID,
		Model:                  opts.Model,
		Instructions:           opts.Instructions,
		AdditionalInstructions: opts.AdditionalInstructions,
		Tools:                  opts.Tools,
		Metadata:               opts.Metadata,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run payload: %w", err)
	}
	// POST to /threads/{threadID}/runs
	url := fmt.Sprintf("%s/%s/runs", threadsURL, t.dto.ID)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	t.client.addHeaders(req)
	resp, err := t.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		d, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("error creating run: %s %s", resp.Status, string(d))
	}
	var dto runDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, err
	}
	return &runHandle{client: t.client, dto: dto}, nil
}

// RunAndFetch runs the assistant on this thread, awaits completion, and returns the assistant message.
func (t *threadHandle) RunAndFetch(ctx context.Context, opts *assistants.RunOptions, msgs ...assistants.InputMessage) (assistants.Run, *assistants.Message, error) {
	if opts == nil {
		opts = &assistants.RunOptions{}
	}
	// add messages to thread
	for _, m := range msgs {
		if _, err := t.AddMessage(m); err != nil {
			return nil, nil, fmt.Errorf("failed to add message: %w", err)
		}
	}
	// create run
	runIface, err := t.Run(opts)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a run: %w", err)
	}
	run := runIface

	// await run completion or required action cycles
	for {
		// await terminal or requires_action
		if err := run.Await(ctx); err != nil {
			return run, nil, fmt.Errorf("failed to await run: %w", err)
		}
		// if functions called
		if run.IsExpectingToolOutputs() {
			// gather and execute
			rh := run.(*runHandle)
			var outputs []assistants.ToolOutput
			for _, tc := range rh.dto.RequiredAction.SubmitToolOutputs.ToolCalls {
				f, ok := t.client.Config.Tools.GetFunction(tc.Function.Name)
				if !ok {
					return run, nil, fmt.Errorf("function '%s' is not registered", tc.Function.Name)
				}
				// execute or skip
				if f.F != nil {
					res, ferr := f.F([]byte(tc.Function.Arguments))
					if ferr != nil && !errors.Is(ferr, tools.ErrDoNotRespond) {
						return run, nil, fmt.Errorf("failed to execute function '%s': %w", tc.Function.Name, ferr)
					}
					outputs = append(outputs, assistants.ToolOutput{ToolCallID: tc.ID, Output: res})
				} else {
					outputs = append(outputs, assistants.ToolOutput{ToolCallID: tc.ID, Output: ""})
				}
			}
			if err := run.SubmitToolOutputs(outputs...); err != nil {
				return run, nil, fmt.Errorf("failed to submit tool outputs: %w", err)
			}
			continue
		}
		break
	}
	// if not completed
	rc := run.(*runHandle).dto.Status
	if rc != "completed" {
		return run, nil, nil
	}
	// fetch only the assistant's reply
	msgs2, _, err := t.Messages(1, "")
	if err != nil {
		return run, nil, err
	}
	if len(msgs2) == 0 {
		return run, nil, nil
	}
	return run, &msgs2[0], nil
}
