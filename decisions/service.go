// Package decisions / service.go contains the service layer for OpenAI Decisions API.
package decisions

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/roles"
)

// Question types. Answers use the type of their question, or AnswerTypeRefusal.
const (
	QuestionTypePredicate = "predicate" // probability that a condition is true
	QuestionTypeChoice    = "choice"    // one value from a fixed set
	QuestionTypeScore     = "score"     // probability-weighted average of ordered level indices

	AnswerTypeRefusal = "refusal" // the question was declined
)

// Service is the service layer interface for OpenAI Decisions API.
type Service interface {
	// Send evaluates questions against shared input. Answers are returned in question order.
	Send(ctx context.Context, req *Request) (*Decision, error)
}

// Request is the request body for the Decisions API.
type Request struct {
	// Required
	Model     string     `json:"model"`
	Input     []Message  `json:"input"`
	Questions []Question `json:"questions"`

	// Optional
	SafetyIdentifier string `json:"safety_identifier,omitempty"` // Stable unique identifier for end user, preferably anonymized
}

// Message is one user message of the shared input. Its text parts are joined without separators,
// while separate messages stay distinguishable to the model. Message may contain images.
type Message []Part

// TextInput converts a plain string into a valid Request.Input representation.
func TextInput(text string) []Message {
	return []Message{{Text(text)}}
}

// MarshalJSON marshals the message with the "user" role, the only one the API accepts.
func (m Message) MarshalJSON() ([]byte, error) {
	return openai.Marshal(struct {
		Role    string `json:"role"`
		Content []Part `json:"content"`
	}{Role: roles.User, Content: m})
}

// Part is a Text, an Image, or a Base64Image.
type Part interface{ part() }

// Text is a text part of a message.
type Text string

// Image is an inline image part of a message. The API accepts only base64 data URLs,
// so Data is encoded as such with its detected image type.
type Image struct {
	Data   []byte
	Detail string // "low", "high", "auto", or "original"; default "auto"
}

// Base64Image is an inline image part given as standard base64, optionally as a data URL.
// A data URL's declared type is replaced with the type detected from its data;
// Send logs a warning when they differ.
type Base64Image struct {
	Data   string
	Detail string // "low", "high", "auto", or "original"; default "auto"
}

func (Text) part()        {}
func (Image) part()       {}
func (Base64Image) part() {}

// MarshalJSON marshals the text as an "input_text" part.
func (t Text) MarshalJSON() ([]byte, error) {
	return openai.Marshal(struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}{Type: "input_text", Text: string(t)})
}

// MarshalJSON marshals the image as an "input_image" part with a base64 data URL.
func (i Image) MarshalJSON() ([]byte, error) {
	return Base64Image{Data: base64.StdEncoding.EncodeToString(i.Data), Detail: i.Detail}.MarshalJSON()
}

// MarshalJSON marshals the image as an "input_image" part with a base64 data URL.
func (i Base64Image) MarshalJSON() ([]byte, error) {
	_, payload, detected, err := i.parse()
	if err != nil {
		return nil, err
	}
	return openai.Marshal(struct {
		Type     string `json:"type"`
		ImageURL string `json:"image_url"`
		Detail   string `json:"detail,omitempty"`
	}{Type: "input_image", ImageURL: "data:" + detected + ";base64," + payload, Detail: i.Detail})
}

// MediaTypes returns the type declared by a data URL, empty for plain base64,
// and the type detected from the data, which is the one sent.
func (i Base64Image) MediaTypes() (declared, detected string, err error) {
	declared, _, detected, err = i.parse()
	return declared, detected, err
}

// parse splits off a data URL header and detects the image type from the base64 payload.
func (i Base64Image) parse() (declared, payload, detected string, err error) {
	payload = i.Data
	// Base64 has no commas, so only a data URL has a header to cut.
	if header, rest, ok := strings.Cut(payload, ","); ok {
		if mediaType, ok := strings.CutPrefix(header, "data:"); ok {
			declared, _, _ = strings.Cut(mediaType, ";")
			payload = rest
		}
	}
	// http.DetectContentType reads at most 512 bytes.
	prefix, err := base64.StdEncoding.DecodeString(payload[:min(len(payload), base64.StdEncoding.EncodedLen(512))])
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode base64 image: %w", err)
	}
	detected = http.DetectContentType(prefix)
	if !strings.HasPrefix(detected, "image/") {
		return "", "", "", fmt.Errorf("image data has non-image content type %q", detected)
	}
	return declared, payload, detected, nil
}

// Question is one question to evaluate against the request input.
type Question struct {
	Type         string   `json:"type"`           // "predicate", "choice", or "score"
	Name         string   `json:"name,omitempty"` // echoed in the answer
	Instructions string   `json:"instructions"`
	Choices      []Choice `json:"choices,omitempty"` // "choice" only
	Levels       []Level  `json:"levels,omitempty"`  // "score" only, ordered from lowest to highest
}

// Choice is one option of a "choice" question.
type Choice struct {
	Value       any    `json:"value"` // string or bool; "true" and true are distinct values
	Description string `json:"description,omitempty"`
}

// Level is one ordered level of a "score" question.
type Level struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Decision is the response body of the Decisions API.
type Decision struct {
	Model   string        `json:"model"`
	Answers []Answer      `json:"answers"`
	Usage   *models.Usage `json:"usage"`

	// ProcessingRegion is the region recorded from the API endpoint. "global"
	// means the global endpoint; an empty value means the routing is unknown.
	ProcessingRegion string `json:"-"`
}

// Answer returns the answer to the question with the given name.
// Unnamed questions can share an empty name, so their answers are only available by position in Answers.
func (d *Decision) Answer(name string) (Answer, bool) {
	if name == "" {
		return Answer{}, false
	}
	for _, answer := range d.Answers {
		if answer.Name == name {
			return answer, true
		}
	}
	return Answer{}, false
}

// EstimateCost estimates the charge in USD from the model, usage, and processing region.
// A non-nil error means the amount is only the known subtotal.
func (d *Decision) EstimateCost() (float64, error) {
	pricing, ok := models.DecisionData[d.Model]
	if !ok {
		return 0, fmt.Errorf("no Decisions pricing found for model %q", d.Model)
	}
	cost, costErr := pricing.Cost(d.Usage)
	cost, regionErr := pricing.RegionalCost(cost, d.ProcessingRegion)
	return cost, errors.Join(costErr, regionErr)
}

// Answer is the result of one question.
type Answer struct {
	Type          string        `json:"type"`          // "predicate", "choice", "score", or "refusal"
	Name          string        `json:"name"`          // empty for unnamed questions
	Probability   float64       `json:"probability"`   // "predicate" only
	Choice        any           `json:"choice"`        // "choice" only: the selected string or bool value
	Score         float64       `json:"score"`         // "score" only: may fall between levels
	Confidence    float64       `json:"confidence"`    // "choice" and "score"
	Probabilities []Probability `json:"probabilities"` // "choice" and "score"
}

// Probability is the probability of one choice or score level.
type Probability struct {
	Value       any     `json:"value"`           // string or bool for "choice"; level index as float64 for "score"
	Label       string  `json:"label,omitempty"` // "score" only
	Probability float64 `json:"probability"`
}
