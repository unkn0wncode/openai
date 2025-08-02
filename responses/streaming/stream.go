// Package streaming provides a streaming iterator API for OpenAI Responses API.
package streaming

import (
	"context"
	"sync"
)

// Stream represents a streaming response iterator with a Next() method.
type Stream struct {
	eventChan <-chan any
	ctx       context.Context
	current   any
	err       error
	done      bool
}

// NewStream creates a new Stream from an event channel and context.
func NewStream(ctx context.Context, eventChan <-chan any) *Stream {
	return &Stream{
		eventChan: eventChan,
		ctx:       ctx,
	}
}

// Next advances the stream to the next event.
// It returns true if there is an event available, false if the stream is done or an error occurred.
// After Next returns false, use Err() to check if it was due to an error.
func (s *Stream) Next() bool {
	if s.done {
		return false
	}

	select {
	case event, ok := <-s.eventChan:
		if !ok {
			s.done = true
			return false
		}

		// Check if the event is an error
		if err, isErr := event.(error); isErr {
			s.err = err
			s.done = true
			return false
		}

		s.current = event
		return true
	case <-s.ctx.Done():
		s.err = s.ctx.Err()
		s.done = true
		return false
	}
}

// Event returns the current event. Only valid after Next() returns true.
func (s *Stream) Event() any {
	return s.current
}

// Err returns any error that occurred during iteration.
func (s *Stream) Err() error {
	return s.err
}

// Close closes the stream.
func (s *Stream) Close() {
	s.done = true
}

// StreamIterator provides both Next() iteration and channel-based iteration.
type StreamIterator struct {
	*Stream
	chanOnce   sync.Once
	outputChan chan any
}

// NewStreamIterator creates a new StreamIterator from a Stream.
func NewStreamIterator(ctx context.Context, eventChan <-chan any) *StreamIterator {
	return &StreamIterator{
		Stream: NewStream(ctx, eventChan),
	}
}

// Chan returns the underlying channel for range iteration.
// This allows: for event := range stream.Chan() { ... }
// Errors are sent through the channel AND stored for later access via Err().
func (s *StreamIterator) Chan() <-chan any {
	s.chanOnce.Do(func() {
		s.outputChan = make(chan any)
		go func() {
			defer close(s.outputChan)

			for {
				select {
				case event, ok := <-s.eventChan:
					if !ok {
						return
					}
					if err, isErr := event.(error); isErr {
						s.err = err
						s.done = true
						s.outputChan <- err
						return
					}
					s.outputChan <- event
				case <-s.ctx.Done():
					s.err = s.ctx.Err()
					s.done = true
					s.outputChan <- s.ctx.Err()
					return
				}
			}
		}()
	})
	return s.outputChan
}

// All collects all events from the stream into a slice.
// This is a convenience method that consumes the entire stream.
// After completion, check Err() for any errors that occurred.
func (s *StreamIterator) All() []any {
	if s.ctx.Err() != nil {
		return nil
	}

	events := []any{}
	for event := range s.Chan() {
		events = append(events, event)
	}
	return events
}
