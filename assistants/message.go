package assistants

import (
	"encoding/json"
	"strings"
)

// Message is a thread-specific format for messages.
type Message struct {
	objectFields

	// The thread ID that this message belongs to.
	ThreadID string `json:"thread_id"`

	// The entity that produced the message.
	// One of user or assistant.
	Role string `json:"role"` // no "system"

	// If applicable, the ID of the assistant that authored this message.
	AssistantID string `json:"assistant_id"`

	// If applicable, the ID of the run associated with the authoring of this message.
	RunID string `json:"run_id"`

	// A list of file IDs that the assistant should use.
	// Useful for tools like retrieval and code_interpreter that can access files.
	// A maximum of 10 files can be attached to a message.
	FileIDs []string `json:"file_ids"`

	// The content of the message in array of text and/or images.
	Content []MessageContent `json:"content"`
}

// MessageContent is either text or image_file.
type MessageContent struct {
	Type string `json:"type"` // "text" or "image_file"

	// The text content that is part of a message.
	Text struct {
		// The data that makes up the text.
		Value       string            `json:"value"`
		Annotations []json.RawMessage `json:"annotations"` // has multiple possible types (file citation or file path), can be expanded later
	} `json:"text"`

	// References an image File in the content of a message.
	ImageFile struct {
		FileID string `json:"file_id"`
	} `json:"image_file"`
}

// String returns the contents of the message as newline-separated strings.
// Images are represented as "Image: <file_id>".
func (m Message) String() string {
	lines := []string{}
	for _, c := range m.Content {
		switch c.Type {
		case "text":
			lines = append(lines, c.Text.Value)
		case "image_file":
			lines = append(lines, "Image: "+c.ImageFile.FileID)
		}
	}
	return strings.Join(lines, "\n")
}

// InputMessage represents message data to be sent to the API, as oppesed to received messages.
type InputMessage struct {
	// The role of the entity that is creating the message.
	// Currently only "user" is supported.
	// It will be filled/overwritten automatically.
	Role string `json:"role"`

	// The text content of the message.
	Content string `json:"content"`

	// A list of File IDs that the message should use.
	// There can be a maximum of 10 files attached to a message.
	// Useful for tools like retrieval and code_interpreter that can access and use files.
	FileIDs []string `json:"file_ids"`

	Metadata Metadata `json:"metadata"`
}
