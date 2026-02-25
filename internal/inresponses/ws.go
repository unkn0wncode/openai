package inresponses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	openai "github.com/unkn0wncode/openai/internal"
	"github.com/unkn0wncode/openai/models"
	"github.com/unkn0wncode/openai/responses"
	"github.com/unkn0wncode/openai/responses/streaming"

	"github.com/gorilla/websocket"
)

type wsClient struct {
	client *Client
	conn   *websocket.Conn

	mu      sync.Mutex
	writeMu sync.Mutex
	closed  bool
	turns   []*wsTurn
}

type wsTurn struct {
	events       chan any
	consumerDone chan struct{}
	finished     chan struct{}
	consumerOnce sync.Once
	finishOnce   sync.Once
}

func newWSTurn() *wsTurn {
	return &wsTurn{
		events:       make(chan any, 64),
		consumerDone: make(chan struct{}),
		finished:     make(chan struct{}),
	}
}

func (t *wsTurn) send(event any) bool {
	select {
	case <-t.consumerDone:
		return false
	case t.events <- event:
		return true
	}
}

func (t *wsTurn) stopConsumer(err error) {
	t.consumerOnce.Do(func() {
		if err != nil {
			select {
			case t.events <- err:
			default:
			}
		}
		close(t.consumerDone)
	})
}

func (t *wsTurn) complete(err error) {
	t.finishOnce.Do(func() {
		t.stopConsumer(err)
		close(t.finished)
		close(t.events)
	})
}

// WebSocket opens a persistent WebSocket connection for response.create events.
// Context is only used for the dialer and doesn't limit connection lifetime.
func (c *Client) WebSocket(ctx context.Context) (responses.WSConn, error) {
	targetURL, err := wsURLFromBase(c.BaseAPI)
	if err != nil {
		return nil, err
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+c.Token)

	dialer := newWebSocketDialer(c)
	conn, _, err := dialer.DialContext(ctx, targetURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to connect websocket: %w", err)
	}

	ws := &wsClient{
		client: c,
		conn:   conn,
	}
	go ws.readLoop()

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
	if !ok || !lt.EnableLog {
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

func (w *wsClient) pushEvent(event any) {
	turn := w.getHeadTurn()
	if turn == nil {
		return
	}

	turn.send(event)
	if isTerminalEvent(event) {
		w.finishTurn(nil)
	}
}

func isTerminalEvent(event any) bool {
	switch event.(type) {
	case streaming.ResponseCompleted,
		streaming.ResponseFailed,
		streaming.ResponseIncomplete,
		streaming.Error,
		streaming.WSError:
		return true
	default:
		return false
	}
}

func (w *wsClient) getHeadTurn() *wsTurn {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.turns) == 0 {
		return nil
	}
	return w.turns[0]
}

func (w *wsClient) finishTurn(err error) {
	w.mu.Lock()
	turn := w.turns[0]
	w.turns = w.turns[1:]
	w.mu.Unlock()

	turn.complete(err)
}

func (w *wsClient) finishAllTurns(err error) {
	w.mu.Lock()
	turns := w.turns
	w.turns = nil
	w.mu.Unlock()

	for _, turn := range turns {
		turn.complete(err)
	}
}

// failAllTurns atomically marks the websocket closed before draining all pending turns,
// preventing new turns from being queued in between.
func (w *wsClient) failAllTurns(err error) {
	w.mu.Lock()
	w.closed = true
	conn := w.conn
	w.mu.Unlock()

	w.finishAllTurns(errors.Join(err, conn.Close()))
}

// Send wraps req as a response.create event, writes it to the WebSocket, and
// returns a streaming iterator for the server's reply. Requests on the same
// connection are queued and processed sequentially by the server.
func (w *wsClient) Send(ctx context.Context, req *responses.Request) (*streaming.StreamIterator, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
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

	turn := newWSTurn()

	w.logPayload("send", eventBytes)

	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil, fmt.Errorf("websocket connection is closed")
	}
	w.turns = append(w.turns, turn)
	w.mu.Unlock()

	w.writeMu.Lock()
	writeErr := w.conn.WriteMessage(websocket.TextMessage, eventBytes)
	w.writeMu.Unlock()
	if writeErr != nil {
		w.failAllTurns(fmt.Errorf("websocket write failed: %w", writeErr))
		return nil, fmt.Errorf("failed to send websocket payload: %w", writeErr)
	}

	if done := ctx.Done(); done != nil {
		go func() {
			select {
			case <-done:
				turn.stopConsumer(ctx.Err())
			case <-turn.finished:
			}
		}()
	}

	return streaming.NewStreamIterator(ctx, turn.events), nil
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
	w.mu.Unlock()

	return conn.Close()
}
