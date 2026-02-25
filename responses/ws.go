package responses

import (
	"context"

	"github.com/unkn0wncode/openai/responses/streaming"
)

// WSConn is a persistent WebSocket connection to the Responses API.
// It is created by Service.WebSocket.
type WSConn interface {
	// Send sends one response.create event and returns a streaming iterator for
	// the resulting server events.
	Send(ctx context.Context, req *Request) (*streaming.StreamIterator, error)
	// Warmup sends one response.create event with generate=false and returns the
	// response ID that can be used as PreviousResponseID.
	Warmup(ctx context.Context, req *Request) (string, error)
	// Close closes the WebSocket connection.
	Close() error
}
