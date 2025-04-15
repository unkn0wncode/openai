package assistants

import (
	"bytes"
	"encoding/json"
	"fmt"
	"macbot/openai/chat"
	"net/http"
)

// Assistant is an object that stores settings for running a chat model in threads.
type Assistant struct {
	objectFields

	// required for creation

	// ID of the model to use.
	// You can use the List models API to see all of your available models, or see our Model overview for descriptions of them.
	Model string `json:"model"` // chat models can be used

	// optional for creation

	// The name of the assistant.
	// The maximum length is 256 characters.
	Name string `json:"name"`

	// The description of the assistant.
	// The maximum length is 512 characters.
	Description string `json:"description"`

	// The system instructions that the assistant uses.
	// The maximum length is 32768 characters.
	Instructions string `json:"instructions"`

	// A list of tool enabled on the assistant.
	// There can be a maximum of 128 tools per assistant.
	// Tools can be of types code_interpreter, retrieval, or function.
	Tools []Tool `json:"tools"`

	// A list of file IDs attached to this assistant.
	// There can be a maximum of 20 files attached to the assistant.
	// Files are ordered by their creation date in ascending order.
	FileIDs []string `json:"file_ids"`
}

// New uses provided assistant data to create a new assistant.
// Can create one with default settings if data is nil or empty.
func New(ast *Assistant) (*Assistant, error) {
	if ast == nil {
		ast = &Assistant{}
	}

	// Model is the only required field, fill it if not provided
	if ast.Model == "" {
		ast.Model = chat.DefaultModel
	}

	// validate what we can
	if err := ast.Metadata.Validate(); err != nil {
		return nil, fmt.Errorf("invalid metadata: %w", err)
	}

	// marshal the assistant object
	b, err := json.Marshal(ast)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal assistant object: %w", err)
	}

	// create request
	req, err := http.NewRequest(http.MethodPost, assistantsURL, bytes.NewBuffer(b))
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
	var newAst Assistant
	if err := json.NewDecoder(resp.Body).Decode(&newAst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &newAst, nil
}

// Load fetches an assistant by ID.
func Load(id string) (*Assistant, error) {
	// create request
	req, err := http.NewRequest(http.MethodGet, assistantsURL+"/"+id, nil)
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
	var ast Assistant
	if err := json.NewDecoder(resp.Body).Decode(&ast); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &ast, nil
}

// LoadAll fetches all existing assistants.
// Handles pagination.
func LoadAll() ([]*Assistant, error) {
	var asts []*Assistant
	var last string
	more := true

	type page struct {
		Data    []*Assistant `json:"data"`
		HasMore bool         `json:"has_more"`
		// other fields exist but are not used here
	}

	for more {
		// create request
		url := assistantsURL + "?limit=100" // max allowed limit
		if last != "" {
			url += "&after=" + last
		}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// execute request
		resp, err := executeRequest(req)
		if err != nil {
			return nil, err
		}

		// unmarshal response
		var p page
		if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to unmarshal response: %w", err)
		}
		resp.Body.Close()

		// extract results
		asts = append(asts, p.Data...)
		more = p.HasMore
		if len(p.Data) > 0 {
			last = p.Data[len(p.Data)-1].ID
		}
	}

	return asts, nil
}