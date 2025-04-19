package assistants

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"openai/assistants"
	"openai/content"
	openai "openai/internal"
)

const (
	assistantsURL = openai.BaseAPI + "v1/assistants"
	threadsURL    = openai.BaseAPI + "v1/threads"
	filesURL      = openai.BaseAPI + "v1/files"
)

// NewClient creates a new internal AssistantsClient with given configuration.
func NewClient(cfg *openai.Config) *AssistantsClient {
	return &AssistantsClient{Config: cfg}
}

// AssistantsClient provides methods to manage assistants.
type AssistantsClient struct {
	*openai.Config
}

// type conformity checks
var (
	_ assistants.AssistantsService = &AssistantsClient{}
	_ assistants.Assistant         = &assistantHandle{}
	_ assistants.Thread            = &threadHandle{}
	// _ assistants.Run             = &runHandle{}
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
	ID        string              `json:"id"`
	Object    string              `json:"object"`
	CreatedAt int                 `json:"created_at"`
	Metadata  assistants.Metadata `json:"metadata"`
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

	b, err = io.ReadAll(resp.Body)
	if err != nil {
		return assistants.Message{}, err
	}

	var raw struct {
		Role    string        `json:"role"`
		Content []content.Any `json:"content"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return assistants.Message{}, fmt.Errorf(
			"error unmarshalling message: %w\nbody: %s",
			err, string(b),
		)
	}

	return assistants.Message{Role: raw.Role, Content: raw.Content}, nil
}

// Messages lists messages in the thread with pagination.
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
	var data struct {
		Data    []struct{ Role, Content string } `json:"data"`
		HasMore bool                             `json:"has_more"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, false, err
	}
	out := make([]assistants.Message, len(data.Data))
	for i, m := range data.Data {
		out[i] = assistants.Message{Role: m.Role, Content: m.Content}
	}
	return out, data.HasMore, nil
}

// Run is not implemented yet.
func (t *threadHandle) Run(opts assistants.RunOptions) (assistants.Run, error) {
	return nil, fmt.Errorf("not implemented")
}

// RunAndFetch is not implemented yet.
func (t *threadHandle) RunAndFetch(ctx context.Context, opts assistants.RunOptions, messages ...assistants.InputMessage) (assistants.Run, assistants.Message, error) {
	return nil, assistants.Message{}, fmt.Errorf("not implemented")
}
