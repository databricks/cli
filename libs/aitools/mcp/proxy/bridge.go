// Package proxy bridges MCP stdio (newline-delimited JSON-RPC) to a streamable HTTP MCP endpoint.
package proxy

import (
	"bufio"
	"bytes"
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
	logPrefix       = "databricks aitools mcp serve: "
	sessionHeader   = "Mcp-Session-Id"
	protocolHeader  = "MCP-Protocol-Version"
	acceptHeader    = "application/json, text/event-stream"
	eventStreamMIME = "text/event-stream"

	maxLineBytes   = 64 << 20
	connectTimeout = 10 * time.Second
	idleTimeout    = 10 * time.Minute
	shutdownGrace  = 2 * time.Second
	deleteTimeout  = 5 * time.Second

	codeParseError    = -32700
	codeInvalidParams = -32602
	codeUpstream      = -32000
)

var (
	newline        = []byte{'\n'}
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
		base:       base,
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
			c.handle(line)
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
	return sc.Err()
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

// conn is the state of one Run.
type conn struct {
	b          *Bridge
	client     *http.Client
	base       context.Context
	cancelBase context.CancelFunc
	wg         sync.WaitGroup

	ioMu sync.Mutex // serializes writes to Out and Err

	mu              sync.Mutex // guards the fields below
	sessionID       string
	protocolVersion string
	inflight        map[string]context.CancelFunc
}

// newClient returns the upstream HTTP client: a 10 s connect timeout and the environment's proxy settings.
func newClient() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DialContext = (&net.Dialer{Timeout: connectTimeout}).DialContext
	return &http.Client{Transport: t}
}

// handle processes one stdin line. Upstream work runs in its own goroutine.
func (c *conn) handle(line []byte) {
	m, err := parseMessage(line)
	if err != nil {
		c.logf("invalid JSON on stdin: %v", err)
		c.reply(errorResponse(nil, codeParseError, "Parse error"))
		return
	}
	switch m.Method {
	case "initialize":
		var p struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		if err := json.Unmarshal(m.Params, &p); err != nil {
			c.reply(errorResponse(m.ID, codeInvalidParams, "Invalid params"))
			return
		}
		c.setProtocol(p.ProtocolVersion)
	case "notifications/cancelled":
		c.cancelRequest(m.Params)
	}

	ctx, cancel := context.WithCancel(c.base)
	isRequest := m.isRequest()
	key := string(m.ID)
	if isRequest {
		c.track(key, cancel)
	}
	c.wg.Go(func() {
		defer cancel()
		if isRequest {
			defer c.untrack(key)
		}
		c.forward(ctx, line, m)
	})
}

// forward POSTs one client message exactly as received and writes what the upstream answers.
func (c *conn) forward(ctx context.Context, line []byte, m message) {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)
	idle := time.AfterFunc(idleTimeout, func() { cancel(errIdleTimeout) })
	defer idle.Stop()

	x := &exchange{c: c, ctx: ctx, line: line, m: m}
	protocol := ""
	if m.Method != "initialize" {
		protocol = c.protocol()
	}
	resp, err := c.post(ctx, line, protocol)
	if err != nil {
		x.fail(err)
		return
	}
	defer resp.Body.Close()
	x.status = resp.StatusCode
	if sid := resp.Header.Get(sessionHeader); sid != "" {
		c.setSession(sid)
	}
	resp.Body = idleBody{ReadCloser: resp.Body, idle: idle}
	if err := x.read(resp); err != nil {
		x.fail(err)
		return
	}
	if x.sent == 0 && m.isRequest() {
		x.fail(errNoResponse)
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
		req.Header.Set("Content-Type", "application/json")
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

// write sends one message to stdout as a single line.
func (c *conn) write(msg []byte) error {
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

// logf writes a diagnostic to stderr.
func (c *conn) logf(format string, args ...any) {
	c.ioMu.Lock()
	defer c.ioMu.Unlock()
	fmt.Fprintf(c.b.Err, logPrefix+format+"\n", args...)
}

func (c *conn) session() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sessionID
}

func (c *conn) setSession(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sessionID = id
}

func (c *conn) protocol() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.protocolVersion
}

func (c *conn) setProtocol(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.protocolVersion = version
}

func (c *conn) track(key string, cancel context.CancelFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inflight[key] = cancel
}

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

func (b idleBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.idle.Reset(idleTimeout)
	return n, err
}

// exchange is one client request and the upstream messages answering it.
type exchange struct {
	c      *conn
	ctx    context.Context
	line   []byte
	m      message
	status int
	sent   int
}

// read delivers every message in an upstream response.
func (x *exchange) read(resp *http.Response) error {
	switch {
	case resp.StatusCode/100 != 2:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if json.Valid(body) {
			return x.deliverJSON(body)
		}
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	case strings.HasPrefix(resp.Header.Get("Content-Type"), eventStreamMIME):
		return readEvents(resp.Body, x.deliverJSON)
	default:
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		return x.deliverJSON(body)
	}
}

// deliverJSON writes one upstream JSON message as a single stdout line.
func (x *exchange) deliverJSON(data []byte) error {
	if !json.Valid(data) {
		return errors.New("upstream sent invalid JSON")
	}
	if bytes.ContainsAny(data, "\r\n") {
		var compact bytes.Buffer
		if err := json.Compact(&compact, data); err != nil {
			return err
		}
		data = compact.Bytes()
	}
	return x.deliver(data)
}

// deliver applies the initialize edit and the hooks, then writes the resulting messages.
func (x *exchange) deliver(msg []byte) error {
	var err error
	if x.m.Method == "initialize" && (x.c.b.Instructions != "" || x.c.b.ForceListChanged) {
		if msg, err = editInitialize(msg, x.c.b.Instructions, x.c.b.ForceListChanged); err != nil {
			return err
		}
	}
	msgs := [][]byte{msg}
	if x.c.b.Hooks != nil {
		if msgs, err = x.c.b.Hooks.Result(x.ctx, x.line, x.status, msg); err != nil {
			return err
		}
	}
	for _, out := range msgs {
		if err := x.c.write(out); err != nil {
			return err
		}
	}
	x.sent++
	return nil
}

// fail logs err and, for a request the client still waits on, answers it with a JSON-RPC error.
func (x *exchange) fail(err error) {
	cause := context.Cause(x.ctx)
	if errors.Is(cause, errIdleTimeout) {
		err = cause
	}
	x.c.logf("upstream: %v", err)
	cancelled := cause != nil && !errors.Is(cause, errIdleTimeout)
	if cancelled || !x.m.isRequest() || x.sent > 0 {
		return
	}
	msg := errorResponse(x.m.ID, codeUpstream, fmt.Sprintf("MCP upstream error for request %s: %v", x.m.ID, err))
	if derr := x.deliver(msg); derr != nil {
		// The hook failed too; the client still needs an answer, so write the error unhooked.
		x.c.logf("upstream: %v", derr)
		x.c.reply(msg)
	}
}
