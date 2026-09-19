// Package streaming / continuations.go contains steering and tool injection events.
package streaming

import (
	"encoding/json"

	"github.com/unkn0wncode/openai/content/output"
)

// SteerReference identifies queued steering input and its original response.
type SteerReference struct {
	ID                 string `json:"id"`
	PreviousResponseID string `json:"previous_response_id"`
}

// SteerRequiredInput identifies a tool result or approval needed before continuation.
type SteerRequiredInput struct {
	Type              string `json:"type"`
	CallID            string `json:"call_id,omitempty"`
	Name              string `json:"name,omitempty"`
	Execution         string `json:"execution,omitempty"`
	ApprovalRequestID string `json:"approval_request_id,omitempty"`
}

// ResponseSteerAccepted acknowledges that the server has queued the input.
// The updated response is delivered later on the same connection.
type ResponseSteerAccepted struct {
	BaseEvent
	Steer SteerReference `json:"steer"`
}

// ResponseSteerPending reports the input needed to apply accepted steering.
type ResponseSteerPending struct {
	BaseEvent
	Steer         SteerReference       `json:"steer"`
	Reason        string               `json:"reason"`
	RequiredInput []SteerRequiredInput `json:"required_input"`
}

// ResponseSteerFailed returns steering input that the server could not apply.
type ResponseSteerFailed struct {
	BaseEvent
	Steer struct {
		ID                 string          `json:"id,omitempty"`
		PreviousResponseID string          `json:"previous_response_id"`
		Input              json.RawMessage `json:"input"`
	} `json:"steer"`
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ResponseInjectCreated confirms that all supplied tool outputs were accepted.
type ResponseInjectCreated struct {
	BaseEvent
	ResponseID string `json:"response_id"`
}

// ResponseInjectFailed returns tool outputs that could not be added to an active response.
// Input can be submitted in a later request when continuing the completed response.
type ResponseInjectFailed struct {
	BaseEvent
	ResponseID string       `json:"response_id"`
	Input      []output.Any `json:"input"`
	Error      struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
