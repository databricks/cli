// Package proxy bridges MCP stdio (newline-delimited JSON-RPC) to a streamable HTTP MCP endpoint.
package proxy

import (
	"bufio"
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	logPrefix         = "databricks aitools mcp serve: "
	sessionHeader     = "Mcp-Session-Id"
	protocolHeader    = "MCP-Protocol-Version"
	contentTypeHeader = "Content-Type"
	acceptHeader      = "Accept"
	acceptTypes       = "application/json, text/event-stream"
	jsonMIME          = "application/json"
	eventStreamMIME   = "text/event-stream"

	maxLineBytes   = 64 << 20
	maxErrorBytes  = 1 << 20 // a non-2xx body is read up to this many bytes; its error message is an excerpt
	connectTimeout = 10 * time.Second
	deleteTimeout  = 5 * time.Second
	excerptRunes   = 200
	excerptBytes   = 4 * excerptRunes // a character takes at most four bytes

	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeUpstream       = -32000
)

var (
	newline = []byte{'\n'}
	// The idle timeout, shutdown grace and notification wait are variables so tests can shorten them.
	idleTimeout = 10 * time.Minute
	// shutdownGrace is how long shutdown waits for in-flight requests before it cancels them.
	shutdownGrace = 2 * time.Second
	// notificationWait is how long a request waits for the notification before it, so that a stalled
	// notification does not hold up later messages.
	notificationWait = 2 * time.Second
	errIdleTimeout   = errors.New("upstream idle")
	errNoResponse    = errors.New("upstream sent no response")
)

// Hooks intercepts upstream messages before they are written to stdout.
type Hooks interface {
	// Result receives each upstream message for a client request, with the HTTP status (0 when the
	// request failed before a response). It returns the messages to write in its place, in order:
	// a single copy of message passes it through, an empty slice drops it. Result is called
	// concurrently, from the goroutine of each request.
	Result(ctx context.Context, request []byte, status int, message []byte) ([][]byte, error)
}

// Bridge serves MCP over stdio by forwarding each client message to Endpoint.
type Bridge struct {
	In       io.Reader
	Out      io.Writer
	Err      io.Writer
	Endpoint string
	// Authenticate sets the Authorization header on each upstream request.
	Authenticate func(*http.Request) error
	// Headers are set on every upstream request.
	Headers map[string]string
	// Instructions is appended to the initialize result's instructions when not empty.
	Instructions string
	// ForceListChanged sets capabilities.tools.listChanged on the initialize result.
	ForceListChanged bool
	// Hooks, when set, sees every upstream message before it is written.
	Hooks Hooks
	// Transport performs the upstream requests. Nil uses the default transport with a 10 s connect timeout.
	Transport http.RoundTripper
}

// Run forwards stdin to the endpoint until stdin closes or ctx is done. It returns an error only when reading stdin
// fails.
func (b *Bridge) Run(ctx context.Context) error {
	base, cancelBase := context.WithCancel(context.WithoutCancel(ctx))
	c := &conn{
		b:          b,
		client:     newClient(b.Transport),
		cancelBase: cancelBase,
		inflight:   map[string]context.CancelFunc{},
		rejoinSlot: make(chan struct{}, 1),
	}

	lines := make(chan []byte)
	stop := make(chan struct{})
	defer close(stop)
	var readErr error
	go func() {
		defer close(lines)
		readErr = scanLines(b.In, lines, stop)
	}()

	// gate closes when the latest notification has been posted. Stdin is never blocked on it.
	var gate <-chan struct{}
	for {
		select {
		case <-ctx.Done():
			c.shutdown(ctx)
			return nil
		case line, ok := <-lines:
			if !ok {
				c.shutdown(ctx)
				if readErr != nil {
					return fmt.Errorf("read stdin: %w", readErr)
				}
				return nil
			}
			// select picks at random when several cases are ready, so a line read after ctx is cancelled is
			// not handled.
			if ctx.Err() != nil {
				c.shutdown(ctx)
				return nil
			}
			gate = c.handle(base, gate, line)
		}
	}
}

// scanLines sends each non-blank stdin line to lines, keeping its bytes exactly as received.
func scanLines(in io.Reader, lines chan<- []byte, stop <-chan struct{}) error {
	sc := bufio.NewScanner(in)
	sc.Buffer(nil, maxLineBytes)
	sc.Split(splitLines)
	for sc.Scan() {
		if len(bytes.TrimSpace(sc.Bytes())) == 0 {
			continue
		}
		select {
		case lines <- bytes.Clone(sc.Bytes()):
		case <-stop:
			return nil
		}
	}
	err := sc.Err()
	if errors.Is(err, bufio.ErrTooLong) {
		return fmt.Errorf("a stdin message exceeds the 64 MiB limit: %w", err)
	}
	return err
}

// splitLines is bufio.ScanLines without dropping a trailing carriage return.
func splitLines(data []byte, atEOF bool) (int, []byte, error) {
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// clientMsg is a client message kept so it can be replayed.
type clientMsg struct {
	line []byte
	m    message
}

// upstreamState is the session and protocol version that requests are sent with.
type upstreamState struct {
	id       string // Mcp-Session-Id, "" when the server has none
	protocol string // the version the server negotiated, "" before initialize completes
}

// conn is the state of one Run.
type conn struct {
	b          *Bridge
	client     *http.Client
	cancelBase context.CancelFunc
	wg         sync.WaitGroup

	ioMu       sync.Mutex    // serializes writes to Out and Err
	rejoinSlot chan struct{} // holds the slot of the one rejoin that runs at a time

	mu          sync.Mutex // guards the fields below
	state       upstreamState
	inflight    map[string]context.CancelFunc
	initialize  clientMsg // the client's initialize, for replay
	initialized clientMsg // the client's notifications/initialized, for replay
}

// newClient returns the upstream HTTP client over rt, or over the default transport when rt is nil. A redirect is
// not followed: a 3xx response is a failed response with its status.
func newClient(rt http.RoundTripper) *http.Client {
	if rt == nil {
		t := http.DefaultTransport.(*http.Transport).Clone()
		t.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
		rt = t
	}
	return &http.Client{
		Transport:     rt,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// handle starts one stdin message and returns the gate for the next one. Each message waits for the notification
// before it (see await), so the server sees the lifecycle in order.
func (c *conn) handle(base context.Context, gate <-chan struct{}, line []byte) <-chan struct{} {
	m, err := parseMessage(line)
	if err != nil {
		if _, ok := errors.AsType[*json.SyntaxError](err); ok {
			c.logf("invalid JSON on stdin: %v", err)
			c.reply(errorResponse(nil, codeParseError, "Parse error"))
		} else {
			c.logf("invalid JSON-RPC message on stdin: %v", err)
			c.reply(errorResponse(m.ID, codeInvalidRequest, "Invalid Request"))
		}
		return gate
	}
	switch m.Method {
	case methodCancelled:
		c.cancelRequest(line)
	case methodInitialize, methodInitialized:
		c.remember(m, line)
	}

	ctx, cancel := context.WithCancel(base)
	if m.isRequest() {
		key := idKey(m.ID)
		c.track(key, cancel)
		c.wg.Go(func() {
			defer cancel()
			defer c.untrack(key)
			c.forward(ctx, gate, line, m)
		})
		return gate
	}
	// The gate closes when the POST finishes or after notificationWait, whichever is first, so a stalled notification
	// holds later messages for at most that long.
	done := make(chan struct{})
	release := sync.OnceFunc(func() { close(done) })
	c.wg.Go(func() {
		timer := time.AfterFunc(notificationWait, release)
		defer timer.Stop()
		defer release()
		defer cancel()
		c.forward(ctx, gate, line, m)
	})
	return done
}

// await waits for gate to close or ctx to be done. A nil gate has nothing to wait for.
func await(ctx context.Context, gate <-chan struct{}) {
	if gate == nil {
		return
	}
	select {
	case <-gate:
	case <-ctx.Done():
	}
}

// forward waits (see await), then POSTs one client message exactly as received and writes what the upstream
// answers. The idle watchdog covers every response, plain JSON included. It starts before the POST, so it also
// bounds the wait for response headers, and the run time of a hook counts toward it.
func (c *conn) forward(ctx context.Context, gate <-chan struct{}, line []byte, m message) {
	await(ctx, gate)
	idleCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	idle := time.AfterFunc(idleTimeout, func() { cancel(errIdleTimeout) })
	defer idle.Stop()

	x := &exchange{c: c, line: line, m: m, idle: idle}
	if err := x.send(idleCtx, ctx, m.isRequest()); err != nil {
		x.fail(idleCtx, ctx, err)
	}
}

// shutdown waits for in-flight requests (up to shutdownGrace), cancels the rest, then ends the session.
func (c *conn) shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(shutdownGrace):
	}
	c.cancelBase()
	<-done
	c.deleteSession(ctx, c.current())
}

// deleteSession ends the session sent, best effort.
func (c *conn) deleteSession(ctx context.Context, sent upstreamState) {
	if sent.id == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteTimeout)
	defer cancel()
	if err := c.sendDelete(ctx, sent); err != nil {
		c.logf("session DELETE: %v", err)
	}
}

// sendDelete sends the DELETE that ends the session sent.
func (c *conn) sendDelete(ctx context.Context, sent upstreamState) error {
	req, err := c.newRequest(ctx, http.MethodDelete, nil, sent)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// post sends one JSON-RPC message to the endpoint.
func (c *conn) post(ctx context.Context, body []byte, sent upstreamState) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPost, body, sent)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// newRequest builds an upstream request with the configured headers, the session and protocol headers from
// sent, and the authorization set by Authenticate. A nil body means no Content-Type.
func (c *conn) newRequest(ctx context.Context, method string, body []byte, sent upstreamState) (*http.Request, error) {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.b.Endpoint, r)
	if err != nil {
		return nil, err
	}
	for k, v := range c.b.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set(acceptHeader, acceptTypes)
	if body != nil {
		req.Header.Set(contentTypeHeader, jsonMIME)
	}
	if sent.protocol != "" {
		req.Header.Set(protocolHeader, sent.protocol)
	}
	if sent.id != "" {
		req.Header.Set(sessionHeader, sent.id)
	}
	if err := c.b.Authenticate(req); err != nil {
		return nil, err
	}
	return req, nil
}

// write sends one message to stdout as a single line. Upstream bodies and hook output may be
// pretty-printed, so a message with line breaks is compacted first.
func (c *conn) write(msg []byte) error {
	if bytes.ContainsAny(msg, "\r\n") {
		var compact bytes.Buffer
		if err := json.Compact(&compact, msg); err != nil {
			return err
		}
		msg = compact.Bytes()
	}
	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	if _, err := c.b.Out.Write(msg); err != nil {
		return err
	}
	_, err := c.b.Out.Write(newline)
	return err
}

// reply writes msg to stdout, logging a failed write since there is no caller to return it to.
func (c *conn) reply(msg []byte) {
	if err := c.write(msg); err != nil {
		c.logf("stdout: %v", err)
	}
}

// logf writes a diagnostic to stderr. It does not use libs/log: the agent shows the server's stderr to the
// user, so these lines are always written.
func (c *conn) logf(format string, args ...any) {
	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	fmt.Fprintf(c.b.Err, logPrefix+format+"\n", args...)
}

// current returns the session and protocol version that requests are sent with.
func (c *conn) current() upstreamState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// setSession records the Mcp-Session-Id to send on later requests.
func (c *conn) setSession(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.id = id
}

// setProtocol records the protocol version from the initialize result, to send on later requests.
func (c *conn) setProtocol(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.protocol = version
}

// publish replaces the session and protocol version with those of a replay that has finished.
func (c *conn) publish(s upstreamState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = s
}

// remember keeps the client's initialize or notifications/initialized so it can be replayed if the server
// ends the session.
func (c *conn) remember(m message, line []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if m.Method == methodInitialize {
		c.initialize = clientMsg{line: line, m: m}
	} else {
		c.initialized = clientMsg{line: line, m: m}
	}
}

// track records the cancel function of an in-flight request, keyed by its id.
func (c *conn) track(key string, cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight[key] = cancel
}

// untrack removes a request that has finished.
func (c *conn) untrack(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.inflight, key)
}

// cancelRequest cancels the in-flight request named by the requestId of a notifications/cancelled.
func (c *conn) cancelRequest(line []byte) {
	var n struct {
		Params struct {
			RequestID json.RawMessage `json:"requestId"`
		} `json:"params"`
	}
	if err := json.Unmarshal(line, &n); err != nil {
		c.logf("notifications/cancelled: %v", err)
		return
	}
	c.mu.Lock()
	cancel := c.inflight[idKey(n.Params.RequestID)]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// lockRejoin takes the slot for a rejoin. A request queued behind a rejoin that is still running returns when its
// context is done.
func (c *conn) lockRejoin(ctx context.Context) error {
	select {
	case c.rejoinSlot <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlockRejoin releases the slot taken by lockRejoin.
func (c *conn) unlockRejoin() {
	<-c.rejoinSlot
}

// isEventStream reports whether a Content-Type is text/event-stream. The media type is case-insensitive.
func isEventStream(contentType string) bool {
	mt, _, err := mime.ParseMediaType(contentType)
	return err == nil && mt == eventStreamMIME
}

// readBody reads a 2xx body of at most limit bytes. A longer body is an error, as an oversized stdin message is.
func readBody(r io.Reader, limit int) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("an upstream message exceeds the %d MiB limit", limit>>20)
	}
	return body, nil
}

// idleBody resets the idle timer on every read, so a response is cut only after idleTimeout without data.
type idleBody struct {
	io.ReadCloser
	idle *time.Timer
}

// Read reads from the response body and resets the idle timer.
func (b idleBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.idle.Reset(idleTimeout)
	return n, err
}

// exchange is one client message and the upstream messages answering it.
type exchange struct {
	c    *conn
	line []byte
	m    message
	idle *time.Timer // reset by every read of the response

	status   int
	answered bool // the upstream sent the response to m
}

// sending returns the session and protocol version to send with x. An initialize request starts a session,
// so it carries neither (https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#session-management).
func (x *exchange) sending() upstreamState {
	if x.m.Method == methodInitialize {
		return upstreamState{}
	}
	return x.c.current()
}

// recordProtocol keeps the protocol version from an initialize result, before the result is written.
// The version the server chose governs the requests that follow.
func (x *exchange) recordProtocol(data []byte) error {
	v, ok, err := initializeProtocol(data)
	if err != nil || !ok {
		return err
	}
	x.c.setProtocol(v)
	return nil
}

// send posts the message and writes its answer. ctx is cancelled by the idle timeout; reqCtx is the request's
// context, which hooks receive.
func (x *exchange) send(ctx, reqCtx context.Context, retry bool) error {
	x.status = 0 // an attempt that fails before a response has no status of its own
	sent := x.sending()
	resp, err := x.c.post(ctx, x.line, sent)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resp.Body = idleBody{ReadCloser: resp.Body, idle: x.idle}
	x.status = resp.StatusCode
	if resp.StatusCode/100 != 2 {
		return x.httpFailure(ctx, reqCtx, resp, sent.id, retry)
	}
	if id := resp.Header.Get(sessionHeader); id != "" && x.m.Method == methodInitialize {
		x.c.setSession(id)
	}
	if err := x.read(reqCtx, resp); err != nil {
		return err
	}
	if x.m.isRequest() && !x.answered {
		return errNoResponse
	}
	return nil
}

// httpFailure handles a non-2xx response. A body that is the server's response to x.m passes through. Otherwise
// a 404 for a request that carried a session id means the server ended the session, so the client starts a new
// one and posts the request once more.
func (x *exchange) httpFailure(ctx, reqCtx context.Context, resp *http.Response, sent string, retry bool) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	if err != nil {
		return err
	}
	e := decodeErrorBody(body)
	// A 404 ends the session, unless its body is the result for the request. An error under 404 is the server's
	// session-not-found answer, even when it carries the request's id.
	if e != nil && e.isResponseTo(x.m) && (resp.StatusCode != http.StatusNotFound || e.Result != nil) {
		return x.deliverMessage(reqCtx, body, e.message)
	}
	if resp.StatusCode == http.StatusNotFound && sent != "" && retry {
		if err := x.rejoin(ctx, sent); err != nil {
			return err
		}
		return x.send(ctx, reqCtx, false)
	}
	return httpError(resp.StatusCode, body, e)
}

// rejoin starts a new session after the server ended stale. The new session replaces the current one only when the
// replay succeeds. One rejoin runs at a time, and a request that finds the session already replaced just retries.
// The replay sends only initialize and notifications/initialized, so session state set later (a logging level,
// subscriptions) is not restored.
func (x *exchange) rejoin(ctx context.Context, stale string) error {
	c := x.c
	if err := c.lockRejoin(ctx); err != nil {
		return err
	}
	defer c.unlockRejoin()
	if c.current().id != stale {
		return nil
	}
	c.logf("the server ended the session; starting a new one")
	c.mu.Lock()
	init, initialized := c.initialize, c.initialized
	c.mu.Unlock()
	msgs := []clientMsg{init}
	if initialized.line != nil {
		msgs = append(msgs, initialized)
	}
	next, err := c.replay(ctx, x.idle, msgs)
	if err != nil {
		if next.id != "" {
			// The replay created a session that is not used.
			c.deleteSession(ctx, next)
		}
		return fmt.Errorf("start a new session: %w", err)
	}
	c.publish(next)
	return nil
}

// replay posts the client's handshake messages, in order, to a new session. Their answers are read here and not
// written to stdout. It returns the session and protocol version they establish, and on failure the session
// created so far.
func (c *conn) replay(ctx context.Context, idle *time.Timer, msgs []clientMsg) (upstreamState, error) {
	var next upstreamState
	for _, msg := range msgs {
		if err := c.replayOne(ctx, idle, msg, &next); err != nil {
			return next, err
		}
	}
	return next, nil
}

// replayOne posts one handshake message with the session and protocol version replayed so far, and records what its
// answer establishes in next: the session id from the response headers, the protocol version from an initialize result.
func (c *conn) replayOne(ctx context.Context, idle *time.Timer, msg clientMsg, next *upstreamState) error {
	resp, err := c.post(ctx, msg.line, *next)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	resp.Body = idleBody{ReadCloser: resp.Body, idle: idle}
	if resp.StatusCode/100 != 2 {
		return statusError(resp)
	}
	if id := resp.Header.Get(sessionHeader); id != "" && msg.m.Method == methodInitialize {
		next.id = id
	}

	var answer []byte
	if isEventStream(resp.Header.Get(contentTypeHeader)) {
		err = readEvents(resp.Body, func(data []byte) (bool, error) {
			m, err := parseUpstream(data)
			if err != nil {
				return false, err
			}
			if m.isResponseTo(msg.m) {
				answer = data
				return false, nil
			}
			return true, nil
		})
	} else {
		var body []byte
		body, err = readBody(resp.Body, maxLineBytes)
		if err == nil && len(bytes.TrimSpace(body)) > 0 && msg.m.isRequest() {
			if _, err = responseBody(body, msg.m); err == nil {
				answer = body
			}
		}
	}
	if err != nil {
		return err
	}
	if !msg.m.isRequest() {
		return nil
	}
	if answer == nil {
		return errNoResponse
	}
	if msg.m.Method == methodInitialize {
		return recordInitialize(answer, next)
	}
	return nil
}

// recordInitialize sets next's protocol version from the result of an initialize answer. Without a result the
// session is not usable, so the replay fails.
func recordInitialize(answer []byte, next *upstreamState) error {
	v, ok, err := initializeProtocol(answer)
	if err != nil {
		return err
	}
	if !ok {
		return rejectedInitialize(answer)
	}
	next.protocol = v
	return nil
}

// rejectedInitialize is the error for an initialize answer without a result: the server's message when it has one.
func rejectedInitialize(answer []byte) error {
	var r struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(answer, &r); err == nil && r.Error != nil {
		return fmt.Errorf("the server rejected initialize: %s", r.Error.Message)
	}
	return fmt.Errorf("initialize has no result: %s", excerpt(answer))
}

// read delivers the messages in a 2xx response: an event stream, or one JSON body.
func (x *exchange) read(reqCtx context.Context, resp *http.Response) error {
	if isEventStream(resp.Header.Get(contentTypeHeader)) {
		return readEvents(resp.Body, func(data []byte) (bool, error) {
			return x.deliverEvent(reqCtx, data)
		})
	}
	body, err := readBody(resp.Body, maxLineBytes)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if !x.m.isRequest() {
		// A notification or client response is answered with 202 and no body.
		x.c.logf("upstream sent a JSON body for a notification or client response; it is not forwarded")
		return nil
	}
	msg, err := responseBody(body, x.m)
	if err != nil {
		return err
	}
	return x.deliverMessage(reqCtx, body, msg)
}

// deliverEvent handles one event of a 2xx event stream. It reports false once the response is written, since the
// server SHOULD end the stream after it (https://modelcontextprotocol.io/specification/2025-11-25/basic/transports#sending-messages-to-the-server).
func (x *exchange) deliverEvent(reqCtx context.Context, data []byte) (bool, error) {
	msg, err := parseUpstream(data)
	if err != nil {
		return false, err
	}
	if err := x.deliverMessage(reqCtx, data, msg); err != nil {
		return false, err
	}
	return !x.answered, nil
}

// deliverMessage writes one upstream message, whose parsed envelope is msg. The response to the request marks it
// answered once written.
func (x *exchange) deliverMessage(reqCtx context.Context, data []byte, msg message) error {
	answer := msg.isResponseTo(x.m)
	if answer && x.m.Method == methodInitialize {
		if err := x.recordProtocol(data); err != nil {
			return err
		}
	}
	if err := x.deliver(reqCtx, data); err != nil {
		return err
	}
	if answer {
		x.answered = true
	}
	return nil
}

// deliver applies the initialize edit and the hooks, then writes the resulting messages.
func (x *exchange) deliver(reqCtx context.Context, msg []byte) error {
	var err error
	if x.m.Method == methodInitialize && (x.c.b.Instructions != "" || x.c.b.ForceListChanged) {
		if msg, err = editInitialize(msg, x.c.b.Instructions, x.c.b.ForceListChanged); err != nil {
			return err
		}
	}
	msgs := [][]byte{msg}
	if x.c.b.Hooks != nil {
		if msgs, err = x.c.b.Hooks.Result(reqCtx, x.line, x.status, msg); err != nil {
			return err
		}
	}
	for _, out := range msgs {
		if err := x.c.write(out); err != nil {
			return err
		}
	}
	return nil
}

// fail logs err and, for a request the client still waits on, answers it with a JSON-RPC error. A request
// that the client or shutdown cancelled is neither logged nor answered. ctx is the idle-watched context; reqCtx is
// the request's context, which hooks receive.
func (x *exchange) fail(ctx, reqCtx context.Context, err error) {
	cause := context.Cause(ctx)
	switch {
	case errors.Is(cause, errIdleTimeout):
		err = fmt.Errorf("no data from upstream for %s", idleTimeout)
	case cause != nil:
		return
	}
	x.c.logUpstream(err)
	if !x.m.isRequest() || x.answered {
		return
	}
	msg := errorResponse(x.m.ID, codeUpstream, fmt.Sprintf("MCP upstream error for request %s: %v", x.m.ID, err))
	if derr := x.deliver(reqCtx, msg); derr != nil {
		// The hook failed too; the client still needs an answer, so write the error unhooked.
		x.c.logUpstream(derr)
		x.c.reply(msg)
	}
}

// logUpstream logs an error from the upstream.
func (c *conn) logUpstream(err error) {
	c.logf("upstream: %v", err)
}

// parseUpstream parses one message the upstream sent.
func parseUpstream(data []byte) (message, error) {
	m, err := parseMessage(data)
	if err != nil {
		return m, fmt.Errorf("upstream sent an invalid JSON-RPC message: %w", err)
	}
	return m, nil
}

// responseBody parses a JSON body that must be the response to request. Any other message is an error that names it.
func responseBody(body []byte, request message) (message, error) {
	m, err := parseUpstream(body)
	if err != nil {
		return m, err
	}
	if !m.isResponseTo(request) {
		return m, fmt.Errorf("upstream sent %s instead of the response to the request", excerpt(body))
	}
	return m, nil
}

// statusError is the error for a non-2xx response, read as httpFailure reads one.
func statusError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBytes))
	if err != nil {
		return err
	}
	return httpError(resp.StatusCode, body, decodeErrorBody(body))
}

// httpError describes a non-2xx response: its status, then the message of a JSON error body, or an excerpt of
// any other body.
func httpError(status int, body []byte, e *errorBody) error {
	text := excerpt(body)
	if e != nil {
		text = cmp.Or(e.Message, e.Error.Message, text)
	}
	if text == "" {
		return fmt.Errorf("HTTP %d", status)
	}
	return fmt.Errorf("HTTP %d: %s", status, text)
}

// excerpt returns the start of body as a single line of at most excerptRunes characters. Only the first excerptBytes
// bytes are converted.
func excerpt(body []byte) string {
	cut := len(body) > excerptBytes
	if cut {
		body = body[:excerptBytes]
	}
	r := []rune(strings.Join(strings.Fields(string(body)), " "))
	if len(r) > excerptRunes {
		r = r[:excerptRunes]
		cut = true
	}
	if cut {
		return string(r) + "..."
	}
	return string(r)
}
