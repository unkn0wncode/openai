package streaming

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type testSource struct {
	events chan any
	done   chan struct{}
	closed chan struct{}
	once   sync.Once
	err    error
}

func newTestSource() *testSource {
	return &testSource{
		events: make(chan any),
		done:   make(chan struct{}),
		closed: make(chan struct{}),
	}
}

func (s *testSource) Events() <-chan any    { return s.events }
func (s *testSource) Done() <-chan struct{} { return s.done }
func (s *testSource) Err() error {
	<-s.done
	return s.err
}

func (s *testSource) Close() {
	s.once.Do(func() {
		close(s.done)
		close(s.closed)
	})
}

func requireClosed(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("stream source was not closed")
	}
}

func TestSeqEarlyBreakClosesSource(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStreamIterator(context.Background(), src)
	producerStopped := make(chan struct{})

	go func() {
		defer close(producerStopped)

		select {
		case src.events <- "first":
		case <-src.done:
			return
		}

		select {
		case src.events <- "second":
		case <-src.done:
			return
		}
	}()

	for event, err := range stream.Seq() {
		require.NoError(t, err)
		require.Equal(t, "first", event)
		break
	}

	requireClosed(t, src.closed)
	requireClosed(t, producerStopped)
}

func TestNextAfterTerminalEventClosesSource(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStream(context.Background(), src)

	go func() {
		select {
		case src.events <- ResponseCompleted{}:
		case <-src.done:
		}
	}()

	require.True(t, stream.Next())
	require.IsType(t, ResponseCompleted{}, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireClosed(t, src.closed)
}

func TestNextTreatsClosedEventsAsEnd(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	close(src.events)
	stream := NewStream(context.Background(), src)

	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireClosed(t, src.closed)
}

func TestNextConvertsErrorEventToErr(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStream(context.Background(), src)
	event := Error{
		BaseEvent: BaseEvent{Type: "error"},
		Message:   "bad request",
		Code:      "invalid_request",
	}

	go func() {
		select {
		case src.events <- event:
		case <-src.done:
		}
	}()

	require.False(t, stream.Next())

	var streamErr *StreamError
	require.ErrorAs(t, stream.Err(), &streamErr)
	require.Equal(t, event, streamErr.Event)
	require.EqualError(t, streamErr, "stream error: bad request")
	requireClosed(t, src.closed)
}

func TestResponseFailedIsDeliveredAsEvent(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStream(context.Background(), src)
	event := ResponseFailed{
		BaseEvent: BaseEvent{Type: "response.failed"},
	}

	go func() {
		select {
		case src.events <- event:
		case <-src.done:
		}
	}()

	require.True(t, stream.Next())
	require.Equal(t, event, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
	requireClosed(t, src.closed)
}

func TestBufferedTerminalEventAfterDoneIsDelivered(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	src.events = make(chan any, 1)
	event := ResponseCompleted{
		BaseEvent: BaseEvent{Type: "response.completed"},
	}
	src.events <- event
	src.Close()

	stream := NewStream(context.Background(), src)

	require.True(t, stream.Next())
	require.Equal(t, event, stream.Event())
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
}

func TestBufferedProtocolErrorAfterDoneBecomesErr(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	src.events = make(chan any, 1)
	event := WSError{
		BaseEvent: BaseEvent{Type: "error"},
		Status:    400,
	}
	event.Error.Code = "bad_request"
	event.Error.Message = "bad request"
	src.events <- event
	src.Close()

	stream := NewStream(context.Background(), src)

	require.False(t, stream.Next())
	var streamErr *StreamError
	require.ErrorAs(t, stream.Err(), &streamErr)
	require.Equal(t, event, streamErr.Event)
}

func TestSeqYieldsWSErrorEventAsErr(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStreamIterator(context.Background(), src)
	event := WSError{
		BaseEvent: BaseEvent{Type: "error"},
		Status:    400,
	}
	event.Error.Code = "previous_response_not_found"
	event.Error.Message = "previous response not found"

	go func() {
		select {
		case src.events <- event:
		case <-src.done:
		}
	}()

	var gotErr error
	for event, err := range stream.Seq() {
		require.Nil(t, event)
		gotErr = err
	}

	var streamErr *StreamError
	require.ErrorAs(t, gotErr, &streamErr)
	require.Equal(t, event, streamErr.Event)
	requireClosed(t, src.closed)
}

func TestAllReturnsTerminalError(t *testing.T) {
	t.Parallel()

	src := newTestSource()
	stream := NewStreamIterator(context.Background(), src)
	event := Error{
		BaseEvent: BaseEvent{Type: "error"},
		Message:   "bad request",
		Code:      "invalid_request",
	}

	go func() {
		select {
		case src.events <- "first":
		case <-src.done:
			return
		}
		select {
		case src.events <- event:
		case <-src.done:
		}
	}()

	events, err := stream.All()

	require.Equal(t, []any{"first"}, events)
	var streamErr *StreamError
	require.ErrorAs(t, err, &streamErr)
	require.Equal(t, event, streamErr.Event)
	requireClosed(t, src.closed)
}
