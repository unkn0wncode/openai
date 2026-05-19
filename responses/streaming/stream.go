// Package streaming provides a streaming iterator API for OpenAI Responses API.
package streaming

import (
	"context"
	"fmt"
	"iter"
	"sync"
	"sync/atomic"
)

// Source provides events and terminal state for a stream.
type Source interface {
	// Events returns stream events. Implementations may leave buffered events
	// readable after Done is closed, so consumers should read any ready event
	// before treating Done as terminal.
	Events() <-chan any

	// Done returns a channel that is closed when the source has terminated
	// cleanly or with an error.
	Done() <-chan struct{}

	// Err returns the terminal error, or nil on clean completion. It blocks
	// until Done is closed.
	Err() error

	// Close stops the source for local early-exit cleanup. It must be safe to
	// call more than once and after Done is closed.
	Close()
}

// StreamError wraps a protocol-level error event received from the stream.
// The original typed event is preserved for callers that need structured data.
type StreamError struct {
	Event any
}

// Error implements the error interface.
func (e *StreamError) Error() string {
	switch event := e.Event.(type) {
	case Error:
		if event.Message != "" {
			return "stream error: " + event.Message
		}
		if event.Code != "" {
			return "stream error: " + event.Code
		}
	case WSError:
		if event.Error.Message != "" {
			return "stream error: " + event.Error.Message
		}
		if event.Error.Code != "" {
			return "stream error: " + event.Error.Code
		}
	}
	return fmt.Sprintf("stream error: %#v", e.Event)
}

// Stream iterates over streaming events with Next/Event/Err semantics.
//
// Concurrency contract:
//   - Next and Event are single-consumer and must not be called concurrently.
//   - Err is safe to call from any goroutine, but it only reports errors already observed by Next, Seq, or All.
//   - Close is safe to call from any goroutine to stop local iteration.
type Stream struct {
	src    Source
	ctx    context.Context
	events <-chan any
	doneCh <-chan struct{}

	// Owned by the Next/Event caller goroutine.
	current any

	done      atomic.Bool
	closeOnce sync.Once

	errMu sync.Mutex
	err   error
}

// NewStream creates a Stream bound to ctx that reads events from src.
func NewStream(ctx context.Context, src Source) *Stream {
	return &Stream{
		src:    src,
		ctx:    ctx,
		events: src.Events(),
		doneCh: src.Done(),
	}
}

// stop records err and closes the source.
func (s *Stream) stop(err error) {
	if err != nil {
		s.errMu.Lock()
		s.err = err
		s.errMu.Unlock()
	}
	s.done.Store(true)
	s.closeOnce.Do(s.src.Close)
}

// Next advances to the next event. It returns true when Event is available
// and false when the stream ends; call Err after false to inspect any
// terminal failure.
func (s *Stream) Next() bool {
	if s.done.Load() {
		return false
	}

	select {
	case ev, ok := <-s.events:
		return s.handleEvent(ev, ok)
	default:
	}

	select {
	case ev, ok := <-s.events:
		return s.handleEvent(ev, ok)
	case <-s.doneCh:
		select {
		case ev, ok := <-s.events:
			return s.handleEvent(ev, ok)
		default:
		}
		s.stop(s.src.Err())
		return false
	case <-s.ctx.Done():
		s.stop(s.ctx.Err())
		return false
	}
}

func (s *Stream) handleEvent(ev any, ok bool) bool {
	if !ok {
		select {
		case <-s.doneCh:
			s.stop(s.src.Err())
		default:
			s.stop(nil)
		}
		return false
	}
	switch ev.(type) {
	case Error, WSError:
		s.stop(&StreamError{Event: ev})
		return false
	}
	s.current = ev
	if IsTerminalEvent(ev) {
		s.stop(nil)
	}
	return true
}

// Event returns the event produced by the most recent Next call that
// returned true.
func (s *Stream) Event() any { return s.current }

// Err returns the terminal error after Next has returned false, or after Seq
// or All has consumed the stream. It is safe to call from another goroutine,
// but it only reports errors already observed by Next, Seq, or All.
func (s *Stream) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.err
}

// Close stops local iteration and releases the underlying source.
func (s *Stream) Close() { s.stop(nil) }

// StreamIterator exposes multiple iteration styles over a Stream.
type StreamIterator struct {
	*Stream
}

// NewStreamIterator creates a StreamIterator bound to ctx that reads from src.
func NewStreamIterator(ctx context.Context, src Source) *StreamIterator {
	return &StreamIterator{Stream: NewStream(ctx, src)}
}

// Seq returns a sequence of (event, error) pairs for range iteration.
//
// Normal events are yielded as (event, nil). If iteration ends with an error,
// one final (nil, err) pair is yielded. Breaking out of the range early is
// safe and does not leak any goroutine. The event source is then closed.
func (s *StreamIterator) Seq() iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		defer s.Close()

		for s.Next() {
			if !yield(s.Event(), nil) {
				return
			}
		}
		if err := s.Err(); err != nil {
			_ = yield(nil, err)
		}
	}
}

// All collects every non-error event into a slice and returns any terminal error.
func (s *StreamIterator) All() ([]any, error) {
	events := make([]any, 0)
	for ev, err := range s.Seq() {
		if err != nil {
			return events, err
		}
		events = append(events, ev)
	}
	return events, nil
}

// IsTerminalEvent reports whether event ends a response turn. Protocol error
// events are terminal for source routing; Stream converts them to Err instead
// of yielding them.
func IsTerminalEvent(event any) bool {
	switch event.(type) {
	case ResponseCompleted,
		ResponseFailed,
		ResponseIncomplete,
		Error,
		WSError:
		return true
	default:
		return false
	}
}
