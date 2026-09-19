// Package responses / ws.go defines the Responses WebSocket connection.
package responses

import (
	"context"

	"github.com/unkn0wncode/openai/content/output"
	"github.com/unkn0wncode/openai/responses/streaming"
)

// MultiAgentBeta enables the Multi-agent Responses API on a WebSocket connection.
const MultiAgentBeta = "responses_multi_agent=v1"

// WSConn is a persistent WebSocket connection to the Responses API.
// It is created by Service.WebSocket.
// Canceling an active socket write closes the connection because a partial frame
// cannot be resumed. Canceling while waiting to write affects only that request.
type WSConn interface {
	// Send queues one response.create event and returns a streaming iterator for
	// the resulting server events, including automatic steering continuations.
	// When steering needs tool results or approvals, the iterator yields pending
	// events before ending; return the required input in a subsequent Send.
	// That continuation receives any steering failures during its creation before
	// its terminal request error, so returned steering input remains available.
	// Requests are written in order after the previous iterator finishes or closes.
	// Consume or close earlier iterators before waiting for later ones. Write errors
	// are reported through the iterator.
	Send(ctx context.Context, req *Request) (*streaming.StreamIterator, error)
	// Steer queues user input for a response on this connection. Use the ID from
	// response.created and continue consuming its Send iterator for acknowledgement
	// and continuation events. Input is a string or an array of user messages.
	// Steer must be called while that iterator is active.
	Steer(ctx context.Context, previousResponseID string, input any) error
	// Inject submits function outputs to an active Multi-agent response. Continue
	// consuming its Send iterator until the response and all injection
	// acknowledgements arrive. A failed injection is yielded as an event, including
	// the input needed to retry through a new Send after response completion.
	Inject(ctx context.Context, responseID string, input []output.FunctionCallOutput) error
	// Warmup sends one response.create event with generate=false and returns the
	// response ID that can be used as PreviousResponseID.
	Warmup(ctx context.Context, req *Request) (string, error)
	// Close closes the WebSocket connection.
	Close() error
}
