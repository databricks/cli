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
	acceptHeader      = "application/json, text/event-stream"
	jsonMIME          = "application/json"
	eventStreamMIME   = "text/event-stream"

	maxLineBytes   = 64 << 20
	connectTimeout = 10 * time.Second
	deleteTimeout  = 5 * time.Second
	excerptRunes   = 200

	codeParseError     = -32700
	codeInvalidRequest = -32600
	codeUpstream       = -32000
)

var (
	newline = []byte{'\n'}
	// idleTimeout and shutdownGrace are variables so tests can shorten them.
	idleTimeout    = 10 * time.Minute
	shutdownGrace  = 2 * time.Second
	errIdleTimeout = errors.New("no data from upstream for 10 minutes")
	errNoResponse  = errors.New("upstream sent no response")
)

// Hooks intercepts upstream messages before they are written to stdout.
type Hooks interface {
	// Result receives each upstream message for a client request, with the HTTP status (0 when the
	// request failed before a response). It returns the messages to write in its place, in order:
	// a single copy of message passes it through, an empty slice drops it.
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
}

// Run forwards stdin to the endpoint until stdin closes or ctx is done. Shutdown waits up to 2 s for
// in-flight requests, cancels the rest, and sends DELETE for the session. Run returns an error only
// when reading stdin fails.
func (b *Bridge) Run(ctx context.Context) error {
	base, cancelBase := context.WithCancel(context.WithoutCancel(ctx))
	c := &conn{
		b:          b,
		client:     newClient(),
		cancelBase: cancelBase,
		inflight:   map[string]context.CancelFunc{},
	}

	lines := make(chan []byte)
	stop := make(chan struct{})
	defer close(stop)
	var readErr error
	go func() {
		defer close(lines)
		readErr = scanLines(b.In, lines, stop)
	}()

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
			if posted := c.handle(base, line); posted != nil {
				// Wait for the notification to be posted, but do not hold up shutdown.
				select {
				case <-posted:
				case <-ctx.Done():
				}
			}
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

// conn is the state of one Run.
type conn struct {
	b          *Bridge
	client     *http.Client
	cancelBase context.CancelFunc
	wg         sync.WaitGroup

	ioMu     sync.Mutex // serializes writes to Out and Err
	rejoinMu sync.Mutex // allows one session restart at a time

	mu              sync.Mutex // guards the fields below
	sessionID       string
	protocolVersion string
	inflight        map[string]context.CancelFunc
	initialize      clientMsg // the client's initialize, for replay
	initialized     clientMsg // the client's notifications/initialized, for replay
}

// newClient returns the upstream HTTP client: a 10 s connect timeout and the environment's proxy settings.
func newClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	return &http.Client{Transport: t}
}

// handle starts one stdin message. A request runs in its own goroutine, and handle returns nil. A
// notification or client response is posted before the next stdin line is read, so the server sees the
// client's messages in order (the lifecycle needs notifications/initialized before the first request).
// handle then returns a channel that closes when that exchange has finished.
func (c *conn) handle(base context.Context, line []byte) <-chan struct{} {
	m, err := parseMessage(line)
	if err != nil {
		if json.Valid(line) {
			c.logf("invalid JSON-RPC message on stdin: %v", err)
			c.reply(errorResponse(m.ID, codeInvalidRequest, "Invalid Request"))
		} else {
			c.logf("invalid JSON on stdin: %v", err)
			c.reply(errorResponse(nil, codeParseError, "Parse error"))
		}
		return nil
	}
	switch m.Method {
	case methodCancelled:
		c.cancelRequest(m.Params)
	case methodInitialize, methodInitialized:
		c.remember(m, line)
	}

	ctx, cancel := context.WithCancel(base)
	if !m.isRequest() {
		done := make(chan struct{})
		c.wg.Go(func() {
			defer close(done)
			defer cancel()
			c.forward(ctx, line, m)
		})
		return done
	}
	key := string(m.ID)
	c.track(key, cancel)
	c.wg.Go(func() {
		defer cancel()
		defer c.untrack(key)
		c.forward(ctx, line, m)
	})
	return nil
}

// forward POSTs one client message exactly as received and writes what the upstream answers. The idle
// watchdog covers every response, plain JSON included (SPEC 11.3).
func (c *conn) forward(ctx context.Context, line []byte, m message) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	idle := time.AfterFunc(idleTimeout, func() { cancel(errIdleTimeout) })
	defer idle.Stop()

	x := &exchange{c: c, line: line, m: m, idle: idle}
	if err := x.send(ctx, true); err != nil {
		x.fail(ctx, err)
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
		c.cancelBase()
		<-done
	}
	c.cancelBase()
	c.deleteSession(ctx)
}

// deleteSession ends the server-side session, best effort.
func (c *conn) deleteSession(ctx context.Context) {
	if c.session() == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), deleteTimeout)
	defer cancel()
	req, err := c.newRequest(ctx, http.MethodDelete, nil, c.protocol())
	if err != nil {
		c.logf("session DELETE: %v", err)
		return
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.logf("session DELETE: %v", err)
		return
	}
	resp.Body.Close()
}

// post sends one JSON-RPC message to the endpoint.
func (c *conn) post(ctx context.Context, body []byte, protocol string) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPost, body, protocol)
	if err != nil {
		return nil, err
	}
	return c.client.Do(req)
}

// newRequest builds an upstream request with the configured headers, the session and protocol headers,
// and the authorization set by Authenticate. A nil body means no Content-Type.
func (c *conn) newRequest(ctx context.Context, method string, body []byte, protocol string) (*http.Request, error) {
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
	req.Header.Set("Accept", acceptHeader)
	if body != nil {
		req.Header.Set(contentTypeHeader, jsonMIME)
	}
	if protocol != "" {
		req.Header.Set(protocolHeader, protocol)
	}
	if sid := c.session(); sid != "" {
		req.Header.Set(sessionHeader, sid)
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

// logf writes a diagnostic to stderr. It does not use libs/log: stderr is the agent's server log, which
// users see, and it has no levels to filter by.
func (c *conn) logf(format string, args ...any) {
	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	fmt.Fprintf(c.b.Err, logPrefix+format+"\n", args...)
}

// session returns the Mcp-Session-Id the server assigned, or "" when there is none.
func (c *conn) session() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

// setSession records the Mcp-Session-Id to send on later requests.
func (c *conn) setSession(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}

// protocol returns the MCP protocol version the server negotiated, or "" before initialize completes.
func (c *conn) protocol() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.protocolVersion
}

// setProtocol records the protocol version from the initialize result, to send on later requests.
func (c *conn) setProtocol(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.protocolVersion = version
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

// cancelRequest cancels the in-flight request named by a notifications/cancelled params.requestId.
func (c *conn) cancelRequest(params json.RawMessage) {
	var p struct {
		RequestID json.RawMessage `json:"requestId"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		c.logf("notifications/cancelled: %v", err)
		return
	}
	c.mu.Lock()
	cancel := c.inflight[string(p.RequestID)]
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
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
	c        *conn
	line     []byte
	m        message
	idle     *time.Timer
	silent   bool // a replayed handshake: its answers stay inside the bridge
	status   int
	answered bool // the upstream sent the response to m
}

// protocol returns the MCP-Protocol-Version to send with this message. The initialize request carries none.
func (x *exchange) protocol() string {
	if x.m.Method == methodInitialize {
		return ""
	}
	return x.c.protocol()
}

// send posts the message and writes its answer. A 404 for a request that carried a session id means the
// server ended the session. The client then starts a new one and posts the request once more (MCP spec
// 2025-11-25, Session Management).
func (x *exchange) send(ctx context.Context, retry bool) error {
	stale := x.c.session()
	resp, err := x.c.post(ctx, x.line, x.protocol())
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound && stale != "" && retry {
		resp.Body.Close()
		if err := x.rejoin(ctx, stale); err != nil {
			return err
		}
		return x.send(ctx, false)
	}
	x.status = resp.StatusCode
	if sid := resp.Header.Get(sessionHeader); sid != "" {
		x.c.setSession(sid)
	}
	resp.Body = idleBody{ReadCloser: resp.Body, idle: x.idle}
	if err := x.read(ctx, resp); err != nil {
		return err
	}
	if x.m.isRequest() && !x.answered {
		return errNoResponse
	}
	return nil
}

// rejoin starts a new session after the server ended stale. The client's initialize and
// notifications/initialized are replayed silently, since the client already has their answers. The new
// session id and negotiated protocol version are what the replay leaves behind. One rejoin runs at a
// time, and a request that finds the session already replaced just retries.
func (x *exchange) rejoin(ctx context.Context, stale string) error {
	c := x.c
	c.rejoinMu.Lock()
	defer c.rejoinMu.Unlock()
	if c.session() != stale {
		return nil
	}
	c.logf("the server ended the session; starting a new one")
	c.setSession("")
	c.mu.Lock()
	init, initialized := c.initialize, c.initialized
	c.mu.Unlock()
	if init.line == nil {
		return errors.New("the server ended the session, and no initialize request was sent to replay")
	}
	replay := []clientMsg{init}
	if initialized.line != nil {
		replay = append(replay, initialized)
	}
	for _, msg := range replay {
		s := &exchange{c: c, line: msg.line, m: msg.m, idle: x.idle, silent: true}
		if err := s.send(ctx, false); err != nil {
			return fmt.Errorf("start a new session: %w", err)
		}
	}
	return nil
}

// read delivers every message in an upstream response.
func (x *exchange) read(ctx context.Context, resp *http.Response) error {
	switch {
	case resp.StatusCode/100 != 2:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// Only the server's own answer to this request passes through, such as a -32006 under HTTP 401.
		// Any other body is an HTTP failure, and its content reaches the client only in the error message.
		if msg, err := parseMessage(body); err == nil && msg.isResponseTo(x.m) {
			return x.deliverJSON(ctx, body)
		}
		return httpError(resp.StatusCode, body)
	case strings.HasPrefix(resp.Header.Get(contentTypeHeader), eventStreamMIME):
		return readEvents(resp.Body, func(data []byte) error { return x.deliverJSON(ctx, data) })
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		return x.deliverJSON(ctx, body)
	}
}

// deliverJSON writes one upstream JSON message. A response to the request marks it answered once the
// message has been written.
func (x *exchange) deliverJSON(ctx context.Context, data []byte) error {
	if !json.Valid(data) {
		return errors.New("upstream sent invalid JSON")
	}
	msg, err := parseMessage(data)
	if err != nil {
		return fmt.Errorf("upstream sent an invalid JSON-RPC message: %w", err)
	}
	answer := msg.isResponseTo(x.m)
	if answer && x.m.Method == methodInitialize && msg.Result != nil {
		// The version the server chose governs the requests that follow, so record it before the answer is written.
		var r struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(msg.Result, &r); err != nil {
			return fmt.Errorf("initialize result: %w", err)
		}
		x.c.setProtocol(r.ProtocolVersion)
	}
	if err := x.deliver(ctx, data); err != nil {
		return err
	}
	if answer {
		x.answered = true
	}
	return nil
}

// deliver applies the initialize edit and the hooks, then writes the resulting messages. A silent exchange
// writes nothing.
func (x *exchange) deliver(ctx context.Context, msg []byte) error {
	if x.silent {
		return nil
	}
	var err error
	if x.m.Method == methodInitialize && (x.c.b.Instructions != "" || x.c.b.ForceListChanged) {
		if msg, err = editInitialize(msg, x.c.b.Instructions, x.c.b.ForceListChanged); err != nil {
			return err
		}
	}
	msgs := [][]byte{msg}
	if x.c.b.Hooks != nil {
		if msgs, err = x.c.b.Hooks.Result(ctx, x.line, x.status, msg); err != nil {
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
// that the client or shutdown cancelled is neither logged nor answered.
func (x *exchange) fail(ctx context.Context, err error) {
	cause := context.Cause(ctx)
	switch {
	case errors.Is(cause, errIdleTimeout):
		err = cause
	case cause != nil:
		return
	}
	x.c.logf("upstream: %v", err)
	if !x.m.isRequest() || x.answered {
		return
	}
	msg := errorResponse(x.m.ID, codeUpstream, fmt.Sprintf("MCP upstream error for request %s: %v", x.m.ID, err))
	// The hooks see the error with a context that the idle timeout has not cancelled.
	if derr := x.deliver(context.WithoutCancel(ctx), msg); derr != nil {
		// The hook failed too; the client still needs an answer, so write the error unhooked.
		x.c.logf("upstream: %v", derr)
		x.c.reply(msg)
	}
}

// httpError describes a non-2xx response: its status, then the message of a JSON error body or an
// excerpt of any other body.
func httpError(status int, body []byte) error {
	if s := bodySummary(body); s != "" {
		return fmt.Errorf("HTTP %d: %s", status, s)
	}
	return fmt.Errorf("HTTP %d", status)
}

// bodySummary returns the message of a JSON error body, either a JSON-RPC error or a Databricks error_code
// body. Any other body gives a single-line excerpt.
func bodySummary(body []byte) string {
	var e struct {
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &e) == nil {
		if msg := cmp.Or(e.Message, e.Error.Message); msg != "" {
			return msg
		}
	}
	return excerpt(body)
}

// excerpt returns body as a single line, cut to excerptRunes characters.
func excerpt(body []byte) string {
	r := []rune(strings.Join(strings.Fields(string(body)), " "))
	if len(r) > excerptRunes {
		return string(r[:excerptRunes]) + "..."
	}
	return string(r)
}
