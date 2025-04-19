// Package input provides types that can be used as input when sending messages.
package input

import (
	"encoding/json"
	"fmt"
	openai "openai/internal"
)

// Any is a partial representation of a content object with only the "type" field unmarshaled.
// It can be used to find a correct type and further unmarshal the raw content.
type Any struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (a *Any) UnmarshalJSON(data []byte) error {
	// Extract only the "type" field, then save raw JSON for later.
	var tmp struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	a.Type = tmp.Type
	a.raw = data
	return nil
}

// UnmarshalToTarget unmarshals the content into a given target.
func (a *Any) UnmarshalToTarget(target any) error {
	return json.Unmarshal(a.raw, target)
}

// Unmarshal unmarshals the full content into a type specified in the "type" field.
func (a *Any) Unmarshal() (any, error) {
	switch a.Type {
	case "text":
		return unmarshalToType[Text](a)
	case "image_url":
		return unmarshalToType[ImageURL](a)
	case "image_file":
		return unmarshalToType[ImageFile](a)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", a.Type)
	}
}

// unmarshalToType is a generic function that unmarshals Any into a given type.
func unmarshalToType[T any](a *Any) (T, error) {
	var t T
	if err := a.UnmarshalToTarget(&t); err != nil {
		return t, err
	}
	return t, nil
}

// MarshalJSON implements the json.Marshaler interface.
// It just returns the saved raw content.
func (a Any) MarshalJSON() ([]byte, error) {
	return a.raw, nil
}

// String implements the fmt.Stringer interface.
// Returns the raw content as a string.
func (a Any) String() string {
	return string(a.raw)
}

// Text is a string content.
type Text struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "text", discarding any prior value.
func (t Text) MarshalJSON() ([]byte, error) {
	t.Type = "text"
	type alias Text
	return openai.Marshal(alias(t))
}

// String implements the fmt.Stringer interface.
// Returns the text content.
func (t Text) String() string {
	return t.Text
}

// ImageURL is an image referenced by a URL or as base64 encoded data.
type ImageURL struct {
	Type  string `json:"type"` // "image_url"
	Image struct {
		URL    string `json:"url"`              // required
		Detail string `json:"detail,omitempty"` // optional; "auto", "high", "low"
	} `json:"image_url"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "image_url", discarding any prior value.
func (i ImageURL) MarshalJSON() ([]byte, error) {
	i.Type = "image_url"
	type alias ImageURL
	return openai.Marshal(alias(i))
}

// String implements the fmt.Stringer interface.
// Returns the image URL content.
func (i ImageURL) String() string {
	return i.Image.URL
}

// ImageFile is an image referenced by a file ID.
type ImageFile struct {
	Type string `json:"type"` // "image_file"
	File struct {
		FileID string `json:"file_id"`          // required
		Detail string `json:"detail,omitempty"` // optional; "auto", "high", "low"
	} `json:"image_file"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "image_file", discarding any prior value.
func (i ImageFile) MarshalJSON() ([]byte, error) {
	i.Type = "image_file"
	type alias ImageFile
	return openai.Marshal(alias(i))
}

// String implements the fmt.Stringer interface.
// Returns the image file content.
func (i ImageFile) String() string {
	return i.File.FileID
}
