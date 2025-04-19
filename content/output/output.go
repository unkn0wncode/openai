// Package output provides types that can be used as output when receiving messages.
package output

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
	case "refusal":
		return unmarshalToType[Refusal](a)
	default:
		return nil, fmt.Errorf("unsupported content type: %s", a.Type)
	}
}

// unmarshalToType is a generic function that unmarshals Any into a given type.
func unmarshalToType[T any](a interface{ UnmarshalToTarget(any) error }) (T, error) {
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
	Text struct {
		Value       string `json:"value"`
		Annotations any    `json:"annotations,omitempty"`
	} `json:"text"`
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
	return t.Text.Value
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

// AnyAnnotation is an annotation for a text value with only the "type" field unmarshaled.
type AnyAnnotation struct {
	Type string `json:"type"`
	raw  json.RawMessage
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It extracts only the "type" field, then saves the raw JSON for later.
func (a *AnyAnnotation) UnmarshalJSON(data []byte) error {
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

// UnmarshalToTarget unmarshals the annotation into a given target.
func (a *AnyAnnotation) UnmarshalToTarget(target any) error {
	return json.Unmarshal(a.raw, target)
}

// Unmarshal unmarshals the full annotation content into a type specified in the "type" field.
func (a *AnyAnnotation) Unmarshal() (any, error) {
	switch a.Type {
	case "file_citation":
		return unmarshalToType[AnnotationFileCitation](a)
	case "file_path":
		return unmarshalToType[AnnotationFilePath](a)
	default:
		return nil, fmt.Errorf("unsupported annotation type: %s", a.Type)
	}
}

// AnnotationFileCitation is an annotation type that references a part of a file.
type AnnotationFileCitation struct {
	Type         string `json:"type"` // "file_citation"
	Text         string `json:"text"`
	StartIndex   int    `json:"start_index"`
	EndIndex     int    `json:"end_index"`
	FileCitation struct {
		FileID string `json:"file_id"`
	} `json:"file_citation"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "file_citation", discarding any prior value.
func (a AnnotationFileCitation) MarshalJSON() ([]byte, error) {
	a.Type = "file_citation"
	type alias AnnotationFileCitation
	return openai.Marshal(alias(a))
}

// AnnotationFilePath is an annotation type that references a part of a file path.
type AnnotationFilePath struct {
	Type       string `json:"type"` // "file_path"
	Text       string `json:"text"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	FilePath   struct {
		FileID string `json:"file_id"`
	} `json:"file_path"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "file_path", discarding any prior value.
func (a AnnotationFilePath) MarshalJSON() ([]byte, error) {
	a.Type = "file_path"
	type alias AnnotationFilePath
	return openai.Marshal(alias(a))
}

// Refusal is a refusal to process the request.
type Refusal struct {
	Type    string `json:"type"` // "refusal"
	Refusal string `json:"refusal"`
}

// MarshalJSON implements the json.Marshaler interface.
// It fills in the "type" field with "refusal", discarding any prior value.
func (r Refusal) MarshalJSON() ([]byte, error) {
	r.Type = "refusal"
	type alias Refusal
	return openai.Marshal(alias(r))
}

// String implements the fmt.Stringer interface.
// Returns the refusal content.
func (r Refusal) String() string {
	return r.Refusal
}
