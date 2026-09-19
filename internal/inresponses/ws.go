// Package inresponses / ws.go implements the WebSocket transport for the Responses API.
package inresponses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/unkn0wncode/openai/content/output"
	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"
	"github.com/unkn0wncode/openai/tools"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	client *Client
	conn   *websocket.Conn

	mu         sync.Mutex
	writeGate  chan struct{}
	closed     bool
	turns      []*wsTurn
	active     *wsTurn
	writeReady chan struct{}
}

// wsSteer remains owned until its input is committed or returned by the server.
type wsSteer struct {
	id                 string // empty until acknowledged
	previousResponseID string
	pending            bool // waiting for explicit tool results or approval
}

// wsTurn tracks a response.create and its automatic steering continuations.
type wsTurn struct {
	owner            *wsClient
	ctx              context.Context
	events           chan any
	done             chan struct{}
	once             sync.Once
	err              error
	request          *responses.Request
	tools            []tools.Tool
	processingRegion string

	// Protected by wsClient.mu. Responses remain associated with the turn until
	// pending steering and injection acknowledgements have been delivered.
	responseID    string
	responses     map[string]bool // response ID to terminal status
	received      bool            // the server has responded to response.create
	steers        []wsSteer
	injections    int
	pendingEvents int
	delivering    bool
	deferredError any
	toolResults   bool
	payload       []byte
	sent          bool
}

func newWSTurn(ctx context.Context) *wsTurn {
	return &wsTurn{
		ctx: ctx,
		// Buffer one event so Send can return before the caller starts consuming.
		// Later events wait until the caller reads or closes the stream.
		events:    make(chan any, 1),
		done:      make(chan struct{}),
		responses: make(map[string]bool),
	}
}

var _ streaming.Source = (*wsTurn)(nil)

func (t *wsTurn) Events() <-chan any { return t.events }

func (t *wsTurn) Done() <-chan struct{} { return t.done }

func (t *wsTurn) ContinuesAfterResponse() bool { return true }

// EventConsumed keeps buffered terminal events from closing the iterator before
// the caller can steer or inject in response to the preceding event.
func (t *wsTurn) EventConsumed() {
	if t.owner == nil {
		return
	}
	t.owner.mu.Lock()
	t.pendingEvents--
	if t.canFinishIteration() {
		if len(t.steers) == 0 {
			t.owner.removeTurn(t)
		}
		t.finish(nil)
	}
	t.owner.mu.Unlock()
}

// canFinishIteration reports whether the response is settled and its events
// have been consumed or abandoned. Pending steering may still need to be retained.
// The caller holds wsClient.mu.
func (t *wsTurn) canFinishIteration() bool {
	if !t.responseSettled() {
		return false
	}
	if t.pendingEvents == 0 {
		return true
	}
	select {
	case <-t.done:
		return true
	default:
		return false
	}
}

// responseSettled reports whether generation ended and control replies are
// resolved or waiting for explicit input. Event delivery may still be in progress.
// The caller holds wsClient.mu.
func (t *wsTurn) responseSettled() bool {
	if !t.responses[t.responseID] || t.injections != 0 || t.deferredError != nil {
		return false
	}
	for _, steer := range t.steers {
		if !steer.pending {
			return false
		}
	}
	return true
}

// awaitingCreation excludes rejected requests retained for steering failures.
// The caller holds wsClient.mu.
func (t *wsTurn) awaitingCreation() bool {
	return t.sent && t.responseID == "" && t.deferredError == nil
}

func (t *wsTurn) Err() error {
	<-t.done
	return t.err
}

func (t *wsTurn) Close() { t.finish(nil) }

func (t *wsTurn) deliver(event any) bool {
	select {
	case <-t.done:
		return false
	default:
	}
	select {
	case <-t.done:
		return false
	case t.events <- event:
		return true
	}
}

func (t *wsTurn) finish(err error) {
	t.once.Do(func() {
		t.err = err
		close(t.done)
		if t.owner != nil {
			t.owner.wakeWriter()
		}
	})
}

// finishOnShutdown preserves an interruption error when event delivery is incomplete.
// The caller holds wsClient.mu.
func (t *wsTurn) finishOnShutdown(err error) {
	if t.responseSettled() && !t.delivering {
		err = nil
	}
	t.finish(err)
}

// WebSocket opens a persistent WebSocket connection for Responses events.
// Context is only used for the dialer and doesn't limit connection lifetime.
// Betas are optional OpenAI-Beta header values, such as responses_multi_agent=v1.
func (c *Client) WebSocket(ctx context.Context, betas ...string) (responses.WSConn, error) {
	targetURL, err := wsURLFromBase(c.BaseAPI)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.Token)
	if len(betas) > 0 {
		headers.Set("OpenAI-Beta", strings.Join(betas, ","))
	}

	dialer := newWebSocketDialer(c)
	conn, _, err := dialer.DialContext(ctx, targetURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect websocket: %w", err)
	}

	ws := &wsClient{
		client: c,
		conn:   conn,
		// Wakeups coalesce; pending payloads remain in turns until writable.
		writeReady: make(chan struct{}, 1),
		writeGate:  make(chan struct{}, 1),
	}
	go ws.readLoop()
	go ws.writeLoop()

	return ws, nil
}

func newWebSocketDialer(c *Client) *websocket.Dialer {
	if c.WebSocketDialer != nil {
		dialerCopy := *c.WebSocketDialer
		return &dialerCopy
	}

	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
	}

	if c.HTTPClient == nil || c.HTTPClient.Client == nil {
		return &dialer
	}

	httpClient := c.HTTPClient.Client
	if httpClient.Timeout > 0 {
		dialer.HandshakeTimeout = httpClient.Timeout
	}

	if transport, ok := httpClient.Transport.(*http.Transport); ok && transport != nil {
		if transport.Proxy != nil {
			dialer.Proxy = transport.Proxy
		}
		dialer.NetDialContext = transport.DialContext
		dialer.TLSClientConfig = transport.TLSClientConfig
	}

	return &dialer
}

func wsURLFromBase(base string) (string, error) {
	rawURL, err := url.JoinPath(base, "v1/responses")
	if err != nil {
		return "", fmt.Errorf("failed to build websocket URL: %w", err)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse websocket URL: %w", err)
	}

	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	case "wss", "ws":
	default:
		return "", fmt.Errorf("unsupported base API scheme: %s", u.Scheme)
	}

	return u.String(), nil
}

func (w *wsClient) logPayload(direction string, data []byte) {
	lt, ok := w.client.HTTPClient.Transport.(*openai.LoggingTransport)
	if !ok || !lt.Enabled() {
		return
	}
	w.client.Log.Debug(fmt.Sprintf("websocket %s:\n%s", direction, string(data)))
}

func (w *wsClient) readLoop() {
	for {
		_, message, err := w.conn.ReadMessage()
		if err != nil {
			w.failAllTurns(fmt.Errorf("websocket read failed: %w", err))
			return
		}

		w.logPayload("recv", message)

		event, err := streaming.Unmarshal(message)
		if err != nil {
			w.failAllTurns(fmt.Errorf("failed to unmarshal websocket event: %w", err))
			return
		}

		w.pushEvent(event)
	}
}

func (w *wsClient) wakeWriter() {
	select {
	case w.writeReady <- struct{}{}:
	default:
	}
}

// writeLoop keeps later response events from blocking control acknowledgements
// for the iterator the caller is still consuming.
func (w *wsClient) writeLoop() {
	for range w.writeReady {
		w.mu.Lock()
		closed := w.closed
		w.mu.Unlock()
		if closed {
			return
		}
		if err := w.writePending(); err != nil {
			w.failAllTurns(fmt.Errorf("websocket write failed: %w", err))
			return
		}
	}
}

// writePending sends the next eligible request. writeGate also serializes steering
// and injection writes; mu protects queue selection from readers and cancellation.
func (w *wsClient) writePending() error {
	w.writeGate <- struct{}{}
	defer func() { <-w.writeGate }()
	for {
		w.mu.Lock()
		if w.closed {
			w.mu.Unlock()
			return nil
		}
		var next *wsTurn
		for i := 0; i < len(w.turns); {
			turn := w.turns[i]
			if turn.sent {
				select {
				case <-turn.done:
					if turn.canFinishIteration() && len(turn.steers) == 0 {
						w.removeTurn(turn)
						continue
					}
					i++
					continue
				default:
					w.mu.Unlock()
					return nil
				}
			}
			select {
			case <-turn.done:
				w.removeTurn(turn)
				continue
			default:
			}
			if err := turn.ctx.Err(); err != nil {
				turn.finish(err)
				w.removeTurn(turn)
				continue
			}
			next = turn
			break
		}
		w.mu.Unlock()
		if next == nil {
			return nil
		}
		w.logPayload("send", next.payload)

		// Logging can block. Recheck before committing the request to the wire.
		w.mu.Lock()
		if w.closed {
			w.mu.Unlock()
			return nil
		}
		if err := next.ctx.Err(); err != nil {
			next.finish(err)
			w.removeTurn(next)
			w.mu.Unlock()
			continue
		}
		select {
		case <-next.done:
			w.removeTurn(next)
			w.mu.Unlock()
			continue
		default:
		}
		next.sent = true
		w.handoffSteering(next)
		w.mu.Unlock()
		err := w.writeMessage(next.ctx, next.payload, next, "response.create")
		next.payload = nil
		return err
	}
}

// writeMessage binds an active write to its context. An interrupted WebSocket
// frame cannot be resumed, so cancellation closes the connection.
// The caller holds writeGate and checks ctx before committing the write state.
func (w *wsClient) writeMessage(ctx context.Context, payload []byte, turn *wsTurn, eventType string) error {
	if ctx.Done() == nil {
		return w.conn.WriteMessage(websocket.TextMessage, payload)
	}
	canceled := make(chan struct{})
	interrupted := false
	stop := context.AfterFunc(ctx, func() {
		w.mu.Lock()
		acknowledged := false
		switch eventType {
		case "response.create":
			acknowledged = turn.received
		case "response.steer":
			acknowledged = !slices.ContainsFunc(turn.steers, func(steer wsSteer) bool { return steer.id == "" })
		case "response.inject":
			acknowledged = turn.injections == 0
		}
		if !acknowledged {
			interrupted = true
			w.closed = true
		}
		w.mu.Unlock()
		// A server acknowledgement proves the frame arrived even if the
		// writer has not resumed to retire this cancellation callback yet.
		if !acknowledged {
			w.failAllTurns(ctx.Err())
		}
		close(canceled)
	})
	err := w.conn.WriteMessage(websocket.TextMessage, payload)
	if !stop() {
		// Wait before releasing the writer so this callback cannot close a
		// connection reused by the next write after this one has finished.
		<-canceled
		if interrupted {
			return errors.Join(ctx.Err(), err)
		}
	}
	return err
}

// handoffSteering moves queued input only when its explicit continuation is sent.
// The caller holds mu; canceled requests still in the queue cannot take ownership.
func (w *wsClient) handoffSteering(next *wsTurn) {
	parent := next.request.PreviousResponseID
	previous := w.responseTurn(parent)
	if previous == nil || previous == next {
		return
	}
	matching := next.toolResults
	for _, steer := range previous.steers {
		matching = matching || steer.previousResponseID == parent && steer.pending
	}
	for i := 0; i < len(previous.steers); {
		steer := previous.steers[i]
		if steer.previousResponseID != parent || !matching {
			i++
			continue
		}
		steer.pending = false
		next.steers = append(next.steers, steer)
		previous.steers = slices.Delete(previous.steers, i, i+1)
	}
	if previous.canFinishIteration() && len(previous.steers) == 0 {
		w.removeTurn(previous)
	}
}

func (w *wsClient) pushEvent(event any) {
	w.mu.Lock()
	turn := w.eventTurn(event)
	if turn == nil {
		w.mu.Unlock()
		return
	}
	switch event.(type) {
	case streaming.ResponseSteerAccepted, streaming.ResponseSteerPending, streaming.ResponseSteerFailed,
		streaming.ResponseInjectCreated, streaming.ResponseInjectFailed:
	default:
		turn.received = true
	}
	switch event.(type) {
	case streaming.Error, streaming.WSError:
		if len(turn.steers) > 0 {
			// Returned steering input must be observable before the request error
			// terminates the iterator, regardless of server event order.
			if turn.deferredError == nil {
				turn.deferredError = event
			}
			w.mu.Unlock()
			return
		}
	}
	logCost := w.recordEvent(turn, event)
	turn.pendingEvents++
	turn.delivering = true
	w.mu.Unlock()

	if logCost {
		w.client.logStreamingCost(turn.request, turn.tools, turn.processingRegion, event)
	}
	for {
		delivered := turn.deliver(event)
		w.mu.Lock()
		turn.delivering = false
		if !delivered {
			turn.pendingEvents--
		}
		if len(turn.steers) == 0 && turn.deferredError != nil {
			event = turn.deferredError
			turn.deferredError = nil
			turn.pendingEvents++
			turn.delivering = true
			w.mu.Unlock()
			continue
		}
		finished := turn.canFinishIteration()
		switch event.(type) {
		case streaming.Error, streaming.WSError:
			finished = true
		}
		if finished && len(turn.steers) == 0 {
			w.removeTurn(turn)
		}
		w.mu.Unlock()
		if finished {
			turn.finish(nil)
		}
		return
	}
}

// eventTurn routes late acknowledgements by response ID while ordinary events
// follow the response most recently started by the server. The caller holds mu.
func (w *wsClient) eventTurn(event any) *wsTurn {
	var responseID string
	switch event := event.(type) {
	case streaming.Error, streaming.WSError:
		// Match rejections before response.created in send order, even for
		// abandoned requests. Known, closed responses only drain old events.
		var oldestSent *wsTurn
		for _, turn := range w.turns {
			if !turn.sent {
				continue
			}
			if oldestSent == nil {
				oldestSent = turn
			}
			if turn.awaitingCreation() {
				return turn
			}
			select {
			case <-turn.done:
				continue
			default:
				return turn
			}
		}
		return oldestSent
	case streaming.ResponseCreated:
		if turn := w.responseTurn(event.Response.ID); turn != nil {
			return turn
		}
		for _, turn := range w.turns {
			if turn.awaitingCreation() && len(turn.steers) > 0 &&
				(event.Response.PreviousResponseID == nil || *event.Response.PreviousResponseID == turn.request.PreviousResponseID) {
				return turn
			}
		}
		// An automatic continuation belongs to the original Send, even if a
		// later explicit Send is already queued on the connection.
		for _, turn := range w.turns {
			if turn.deferredError != nil {
				continue
			}
			for _, steer := range turn.steers {
				if steer.pending {
					continue
				}
				if event.Response.PreviousResponseID != nil && *event.Response.PreviousResponseID == steer.previousResponseID ||
					event.Response.PreviousResponseID == nil && turn.responses[turn.responseID] {
					return turn
				}
			}
		}
		for _, turn := range w.turns {
			if turn.awaitingCreation() {
				return turn
			}
		}
	case streaming.ResponseCompleted:
		responseID = event.Response.ID
	case streaming.ResponseIncomplete:
		responseID = event.Response.ID
	case streaming.ResponseFailed:
		responseID = event.Response.ID
	case streaming.ResponseSteerAccepted:
		return w.steerTurn(event.Steer.ID, event.Steer.PreviousResponseID)
	case streaming.ResponseSteerPending:
		return w.steerTurn(event.Steer.ID, event.Steer.PreviousResponseID)
	case streaming.ResponseSteerFailed:
		return w.steerTurn(event.Steer.ID, event.Steer.PreviousResponseID)
	case streaming.ResponseInjectCreated:
		return w.responseTurn(event.ResponseID)
	case streaming.ResponseInjectFailed:
		return w.responseTurn(event.ResponseID)
	}
	if turn := w.responseTurn(responseID); turn != nil {
		return turn
	}
	if w.active != nil {
		return w.active
	}
	if len(w.turns) > 0 {
		return w.turns[0]
	}
	return nil
}

// steerTurn finds a submission after its ownership may have moved to a continuation.
// The caller holds mu.
func (w *wsClient) steerTurn(id, previousID string) *wsTurn {
	for _, turn := range w.turns {
		if turn.steerIndex(id, previousID) >= 0 {
			return turn
		}
	}
	return nil
}

// steerIndex matches an acknowledged submission or the next unanswered send.
// The caller holds wsClient.mu.
func (t *wsTurn) steerIndex(id, previousID string) int {
	unacknowledged := -1
	for i, steer := range t.steers {
		if steer.previousResponseID != previousID {
			continue
		}
		if id != "" && steer.id == id {
			return i
		}
		if steer.id == "" && unacknowledged < 0 {
			unacknowledged = i
		}
	}
	return unacknowledged
}

// responseTurn finds a response still owned by a Send iterator. The caller holds mu.
func (w *wsClient) responseTurn(responseID string) *wsTurn {
	if responseID != "" {
		for _, turn := range w.turns {
			if _, ok := turn.responses[responseID]; ok {
				return turn
			}
		}
	}
	return nil
}

// recordEvent updates workflow state and reports whether terminal usage is new.
// The caller holds mu.
func (w *wsClient) recordEvent(turn *wsTurn, event any) bool {
	var responseID string
	var committedParent string
	switch event := event.(type) {
	case streaming.ResponseCreated:
		committedParent = turn.responseID
		if event.Response.PreviousResponseID != nil {
			committedParent = *event.Response.PreviousResponseID
		} else if committedParent == "" {
			committedParent = turn.request.PreviousResponseID
		}
		turn.responseID = event.Response.ID
		turn.responses[turn.responseID] = false
		w.active = turn
	case streaming.ResponseCompleted:
		responseID = event.Response.ID
	case streaming.ResponseIncomplete:
		responseID = event.Response.ID
	case streaming.ResponseFailed:
		responseID = event.Response.ID
	case streaming.ResponseSteerAccepted:
		if i := turn.steerIndex(event.Steer.ID, event.Steer.PreviousResponseID); i >= 0 {
			turn.steers[i].id = event.Steer.ID
		}
	case streaming.ResponseSteerPending:
		if i := turn.steerIndex(event.Steer.ID, event.Steer.PreviousResponseID); i >= 0 {
			turn.steers[i].pending = true
		}
	case streaming.ResponseSteerFailed:
		if i := turn.steerIndex(event.Steer.ID, event.Steer.PreviousResponseID); i >= 0 {
			turn.steers = slices.Delete(turn.steers, i, i+1)
		}
	case streaming.ResponseInjectCreated, streaming.ResponseInjectFailed:
		if turn.injections > 0 {
			turn.injections--
		}
	default:
	}
	if responseID != "" && turn.responseID == "" {
		turn.responseID = responseID
		committedParent = turn.request.PreviousResponseID
	}
	if committedParent != "" {
		turn.steers = slices.DeleteFunc(turn.steers, func(steer wsSteer) bool {
			return steer.id != "" && steer.previousResponseID == committedParent
		})
	}
	if responseID == "" {
		return false
	}
	alreadyCompleted := turn.responses[responseID]
	turn.responses[responseID] = true
	return !alreadyCompleted
}

// removeTurn releases a workflow after its final event. The caller holds mu.
func (w *wsClient) removeTurn(turn *wsTurn) {
	for i, queued := range w.turns {
		if queued == turn {
			w.turns = slices.Delete(w.turns, i, i+1)
			break
		}
	}
	if w.active == turn {
		w.active = nil
	}
}

func (w *wsClient) finishAllTurns(err error) {
	w.mu.Lock()
	turns := w.turns
	w.turns = nil
	w.active = nil
	for _, turn := range turns {
		turn.finishOnShutdown(err)
	}
	w.mu.Unlock()
}

// failAllTurns synchronously marks the websocket closed before draining all pending turns,
// preventing new turns from being queued in between.
func (w *wsClient) failAllTurns(err error) {
	w.mu.Lock()
	w.closed = true
	conn := w.conn
	w.mu.Unlock()
	w.wakeWriter()

	w.finishAllTurns(errors.Join(err, conn.Close()))
}

// Steer sends input for an automatic continuation of the identified response.
func (w *wsClient) Steer(ctx context.Context, previousResponseID string, input any) error {
	payload := struct {
		Type               string `json:"type"`
		PreviousResponseID string `json:"previous_response_id"`
		Input              any    `json:"input"`
	}{"response.steer", previousResponseID, input}
	return w.sendControl(ctx, previousResponseID, payload.Type, payload)
}

// Inject returns function outputs without starting a new response.create turn.
func (w *wsClient) Inject(ctx context.Context, responseID string, input []output.FunctionCallOutput) error {
	payload := struct {
		Type       string                      `json:"type"`
		ResponseID string                      `json:"response_id"`
		Input      []output.FunctionCallOutput `json:"input"`
	}{"response.inject", responseID, input}
	return w.sendControl(ctx, responseID, payload.Type, payload)
}

// sendControl associates an acknowledgement with the original Send before the
// server can reply. It does not enqueue a new response.create turn.
func (w *wsClient) sendControl(ctx context.Context, responseID, eventType string, payload any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := openai.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %s: %w", eventType, err)
	}

	select {
	case w.writeGate <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-w.writeGate }()
	if err := ctx.Err(); err != nil {
		return err
	}
	w.logPayload("send", data)
	w.mu.Lock()
	if err := ctx.Err(); err != nil {
		w.mu.Unlock()
		return err
	}
	if w.closed {
		w.mu.Unlock()
		return fmt.Errorf("websocket connection is closed")
	}
	turn := w.responseTurn(responseID)
	if turn == nil {
		w.mu.Unlock()
		return fmt.Errorf("response %q has no active Send iterator", responseID)
	}
	select {
	case <-turn.done:
		w.mu.Unlock()
		return fmt.Errorf("response %q Send iterator is closed", responseID)
	default:
	}
	switch eventType {
	case "response.steer":
		turn.steers = append(turn.steers, wsSteer{previousResponseID: responseID})
	case "response.inject":
		turn.injections++
	}
	w.mu.Unlock()

	if err := w.writeMessage(ctx, data, turn, eventType); err != nil {
		w.failAllTurns(fmt.Errorf("websocket write failed: %w", err))
		return fmt.Errorf("failed to send %s: %w", eventType, err)
	}
	return nil
}

// Send queues req as a response.create and returns its streaming iterator.
// Later requests wait for the active iterator to finish or close. Deferred write
// failures are reported by the iterator.
func (w *wsClient) Send(ctx context.Context, req *responses.Request) (*streaming.StreamIterator, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data := req.Clone()
	if data.Model == "" {
		data.Model = models.Default
	}
	if data.Input == nil {
		return nil, fmt.Errorf("input is required")
	}
	data.Stream = false
	data.Background = false

	reqBytes, err := w.client.marshalRequest(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(reqBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed to decode request payload: %w", err)
	}
	payload["type"] = "response.create"

	eventBytes, err := openai.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal websocket payload: %w", err)
	}

	turn := newWSTurn(ctx)
	turn.owner = w
	turn.request = data
	turn.payload = eventBytes
	if items, ok := payload["input"].([]any); ok {
		for _, item := range items {
			if item, ok := item.(map[string]any); ok {
				switch item["type"] {
				case "function_call_output", "custom_tool_call_output", "computer_call_output", "shell_call_output",
					"local_shell_call_output", "apply_patch_call_output", "tool_search_output", "mcp_approval_response":
					turn.toolResults = true
				}
			}
		}
	}
	var sent struct {
		Tools []tools.Tool `json:"tools"`
	}
	if err := json.Unmarshal(reqBytes, &sent); err != nil {
		return nil, fmt.Errorf("failed to decode sent tool definitions: %w", err)
	}
	turn.tools = sent.Tools
	if endpoint, err := url.Parse(w.client.BaseAPI); err == nil {
		turn.processingRegion = processingRegion(endpoint)
	}

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil, fmt.Errorf("websocket connection is closed")
	}
	w.turns = append(w.turns, turn)
	w.mu.Unlock()

	if done := ctx.Done(); done != nil {
		go func() {
			select {
			case <-done:
				turn.finish(ctx.Err())
			case <-turn.done:
			}
		}()
	}
	w.wakeWriter()

	return streaming.NewStreamIterator(ctx, turn), nil
}

// Warmup sends a response.create with generate=false and returns the response ID
// for use as PreviousResponseID in a subsequent Send call.
func (w *wsClient) Warmup(ctx context.Context, req *responses.Request) (string, error) {
	warmupReq := req.Clone()
	generate := false
	warmupReq.Generate = &generate

	stream, err := w.Send(ctx, warmupReq)
	if err != nil {
		return "", err
	}

	var responseID string
	for stream.Next() {
		if e, ok := stream.Event().(streaming.ResponseCreated); ok {
			responseID = e.Response.ID
		}
	}

	if err := stream.Err(); err != nil {
		return "", err
	}
	if responseID == "" {
		return "", fmt.Errorf("warmup response ID not found")
	}

	return responseID, nil
}

// Close closes the WebSocket connection and completes any pending turns.
func (w *wsClient) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	conn := w.conn
	turns := w.turns
	w.turns = nil
	w.active = nil
	w.mu.Unlock()
	w.wakeWriter()

	err := conn.Close()
	streamErr := errors.Join(errors.New("websocket connection closed"), err)
	w.mu.Lock()
	for _, turn := range turns {
		turn.finishOnShutdown(streamErr)
	}
	w.mu.Unlock()
	return err
}
