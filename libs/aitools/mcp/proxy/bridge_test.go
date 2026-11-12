package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "Bearer test-token"

// progressEvent is a server-to-client notification, which is not an answer to any request.
const progressEvent = `{"jsonrpc":"2.0","method":"notifications/progress","params":{"progress":1}}`

// recorded is one upstream request as the fake endpoint received it.
type recorded struct {
	Method string
	Header http.Header
	Body   []byte
}

// fakeEndpoint records every upstream request and answers it with handle.
type fakeEndpoint struct {
	mu   sync.Mutex
	reqs []recorded
}

// newFakeEndpoint starts an httptest server that records each request before calling handle.
func newFakeEndpoint(t *testing.T, handle http.HandlerFunc) (*httptest.Server, *fakeEndpoint) {
	t.Helper()
	f := &fakeEndpoint{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.reqs = append(f.reqs, recorded{Method: r.Method, Header: r.Header.Clone(), Body: body})
		f.mu.Unlock()
		r.Body = io.NopCloser(bytes.NewReader(body))
		handle(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, f
}

func (f *fakeEndpoint) requests() []recorded {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.reqs)
}

// answer replies to a request with an empty result that carries its id.
func answer(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	m, _ := parseMessage(body)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
}

// bridgeFor returns a Bridge that posts to url with a fixed bearer token.
func bridgeFor(url string) *Bridge {
	return &Bridge{
		Endpoint: url,
		Authenticate: func(r *http.Request) error {
			r.Header.Set("Authorization", testToken)
			return nil
		},
	}
}

// setIdleTimeout shortens the idle watchdog for one test.
func setIdleTimeout(t *testing.T, d time.Duration) {
	t.Helper()
	old := idleTimeout
	idleTimeout = d
	t.Cleanup(func() { idleTimeout = old })
}

// setShutdownGrace sets the shutdown grace period for one test.
func setShutdownGrace(t *testing.T, d time.Duration) {
	t.Helper()
	old := shutdownGrace
	shutdownGrace = d
	t.Cleanup(func() { shutdownGrace = old })
}

// setNotificationWait sets how long a request waits for the notification before it, for one test.
func setNotificationWait(t *testing.T, d time.Duration) {
	t.Helper()
	old := notificationWait
	notificationWait = d
	t.Cleanup(func() { notificationWait = old })
}

// waitClosed fails the test unless ch is closed within 10 seconds.
func waitClosed(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for " + what)
	}
}

// syncBuffer is a bytes.Buffer that is safe to read while a bridge writes to it.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// outLines splits bridge output into lines.
func outLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// runAll runs b to completion with input as stdin and returns the stdout lines and stderr. The grace period
// is long so that a slow upstream round trip still completes before shutdown cancels it.
func runAll(t *testing.T, b *Bridge, input string) ([]string, string) {
	t.Helper()
	setShutdownGrace(t, 30*time.Second)
	var out, errOut syncBuffer
	b.In, b.Out, b.Err = strings.NewReader(input), &out, &errOut
	require.NoError(t, b.Run(t.Context()))
	return outLines(out.String()), errOut.String()
}

// piped is a bridge reading from a pipe, so a test can wait for each response before sending the next message.
type piped struct {
	in   *io.PipeWriter
	out  *syncBuffer
	err  *syncBuffer
	done chan error
}

func startPiped(t *testing.T, ctx context.Context, b *Bridge) *piped {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	p := &piped{in: pw, out: &syncBuffer{}, err: &syncBuffer{}, done: make(chan error, 1)}
	b.In, b.Out, b.Err = pr, p.out, p.err
	go func() { p.done <- b.Run(ctx) }()
	return p
}

func (p *piped) send(t *testing.T, line string) {
	t.Helper()
	_, err := io.WriteString(p.in, line+"\n")
	require.NoError(t, err)
}

// waitLines waits until stdout has at least n lines and returns them.
func (p *piped) waitLines(t *testing.T, n int) []string {
	t.Helper()
	require.Eventually(t, func() bool { return len(outLines(p.out.String())) >= n }, 10*time.Second, 5*time.Millisecond)
	return outLines(p.out.String())
}

// finish closes stdin and returns the error from Run.
func (p *piped) finish(t *testing.T) error {
	t.Helper()
	require.NoError(t, p.in.Close())
	select {
	case err := <-p.done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return")
		return nil
	}
}

// rpcReply is the part of a JSON-RPC response the tests check.
type rpcReply struct {
	ID     json.RawMessage `json:"id"`
	Result map[string]any  `json:"result"`
	Error  *rpcError       `json:"error"`
}

func parseReply(t *testing.T, line string) rpcReply {
	t.Helper()
	var r rpcReply
	require.NoError(t, json.Unmarshal([]byte(line), &r), line)
	return r
}

func TestRunPassesUpstreamBodyThroughUnchanged(t *testing.T) {
	const body = `{"jsonrpc": "2.0", "id": 1, "result": {"tools": [], "n": 1.50, "s": "a<b&c"}}`
	const request = `{"jsonrpc" : "2.0", "id":1, "method":"tools/list"}`
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), request+"\n")

	assert.Equal(t, []string{body}, out)
	reqs := fake.requests()
	require.Len(t, reqs, 1)
	assert.Equal(t, request, string(reqs[0].Body))
	assert.Equal(t, testToken, reqs[0].Header.Get("Authorization"))
	assert.Equal(t, "application/json", reqs[0].Header.Get("Content-Type"))
	assert.Equal(t, "application/json, text/event-stream", reqs[0].Header.Get("Accept"))
}

func TestRunWritesEachSSEEventAsOneLine(t *testing.T) {
	const stream = ": keepalive\n\n" +
		"event: message\nid: 1\ndata: {\"jsonrpc\":\"2.0\",\"method\":\"notifications/progress\",\n" +
		"data: \"params\":{}}\n\n" +
		"data: {\"jsonrpc\":\"2.0\",\"id\":1,\"result\":{}}\r\n\r\n"
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, stream)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{
		`{"jsonrpc":"2.0","method":"notifications/progress","params":{}}`,
		`{"jsonrpc":"2.0","id":1,"result":{}}`,
	}, out)
}

// TestRunSkipsSSEPrimingEvent covers the event with an id and empty data that a server sends first, so a
// client can resume (MCP spec 2025-11-25, Streamable HTTP, "Sending Messages to the Server").
func TestRunSkipsSSEPrimingEvent(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "id: 0\ndata:\n\nid: 1\ndata: {\"jsonrpc\":\"2.0\",\"id\":2,\"result\":{}}\n\n")
	})

	out, _ := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{`{"jsonrpc":"2.0","id":2,"result":{}}`}, out)
}

func TestRunSendsNothingForAcceptedNotification(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")

	assert.Empty(t, out)
	assert.Empty(t, errOut)
}

// TestRunPostsNotificationBeforeNextMessage: the server rejects tools/list until notifications/initialized
// has been fully handled, as servers that enforce the lifecycle do.
func TestRunPostsNotificationBeforeNextMessage(t *testing.T) {
	var ready atomic.Bool
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case m.Method == "notifications/initialized":
			time.Sleep(100 * time.Millisecond)
			ready.Store(true)
			w.WriteHeader(http.StatusAccepted)
		case !ready.Load():
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32600,"message":"not initialized"}}`, m.ID)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[]}}`, m.ID)
		}
	})

	out, _ := runAll(t, bridgeFor(srv.URL),
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"+
			`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	assert.Nil(t, parseReply(t, out[0]).Error)
}

func TestRunForwardsSessionAndProtocolHeaders(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		if m.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "s1")
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	// The client asks for a newer version than the server picks; later requests carry the server's choice.
	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	reqs := fake.requests()
	require.Len(t, reqs, 3)
	assert.Empty(t, reqs[0].Header.Get("Mcp-Protocol-Version"), "initialize must not carry the protocol version")
	assert.Empty(t, reqs[0].Header.Get("Mcp-Session-Id"))
	assert.Equal(t, "2025-06-18", reqs[1].Header.Get("Mcp-Protocol-Version"))
	assert.Equal(t, "s1", reqs[1].Header.Get("Mcp-Session-Id"))
	assert.Equal(t, http.MethodDelete, reqs[2].Method)
	assert.Equal(t, "s1", reqs[2].Header.Get("Mcp-Session-Id"))
}

func TestRunSetsConfiguredHeaders(t *testing.T) {
	srv, fake := newFakeEndpoint(t, answer)
	b := bridgeFor(srv.URL)
	b.Headers = map[string]string{"X-Databricks-Workspace-Id": "42"}

	runAll(t, b, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	reqs := fake.requests()
	require.Len(t, reqs, 1)
	assert.Equal(t, "42", reqs[0].Header.Get("X-Databricks-Workspace-Id"))
	assert.Equal(t, testToken, reqs[0].Header.Get("Authorization"))
}

func TestRunEditsInitializeResult(t *testing.T) {
	const result = `{"protocolVersion":"2025-06-18","capabilities":{"tools":{"listChanged":false},"logging":{}},` +
		`"serverInfo":{"name":"genie_chat"},"instructions":"Base."}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":`+result+`}`)
	})
	b := bridgeFor(srv.URL)
	b.Instructions = "Bound to a workspace."
	b.ForceListChanged = true

	out, _ := runAll(t, b, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	assert.Equal(t, "Base.\n\nBound to a workspace.", reply.Result["instructions"])
	assert.Equal(t, "2025-06-18", reply.Result["protocolVersion"])
	assert.Equal(t, map[string]any{"name": "genie_chat"}, reply.Result["serverInfo"])
	assert.Equal(t, map[string]any{"tools": map[string]any{"listChanged": true}, "logging": map[string]any{}},
		reply.Result["capabilities"])
}

func TestRunAnswersFailedRequestsWithJSONRPCError(t *testing.T) {
	const passthrough = `{"jsonrpc":"2.0","id":1,"error":{"code":-32006,"message":"sign in"}}`
	cases := []struct {
		name        string
		status      int
		body        string
		wantError   string
		wantPassing string
	}{
		{name: "HTTP 500", status: http.StatusInternalServerError, body: "boom", wantError: "HTTP 500: boom"},
		{name: "HTTP 401 without JSON", status: http.StatusUnauthorized, body: "unauthorized", wantError: "HTTP 401: unauthorized"},
		{name: "HTTP 401 with JSON-RPC body", status: http.StatusUnauthorized, body: passthrough, wantPassing: passthrough},
		{
			name:      "HTTP 500 with JSON-RPC error without id",
			status:    http.StatusInternalServerError,
			body:      `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"bad request"}}`,
			wantError: "HTTP 500: bad request",
		},
		{
			name:      "Databricks error body",
			status:    http.StatusForbidden,
			body:      `{"error_code":"PERMISSION_DENIED","message":"User lacks access"}`,
			wantError: "HTTP 403: User lacks access",
		},
		{
			name:      "HTML body is excerpted on one line",
			status:    http.StatusForbidden,
			body:      "<html>\n<h1>Forbidden</h1>\n</html>",
			wantError: "HTTP 403: <html> <h1>Forbidden</h1> </html>",
		},
		{name: "no response", status: http.StatusAccepted, wantError: "no response"},
		{name: "invalid JSON", status: http.StatusOK, body: "<html>", wantError: "invalid JSON-RPC message"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})

			out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

			require.Len(t, out, 1)
			if tc.wantPassing != "" {
				assert.Equal(t, tc.wantPassing, out[0])
				return
			}
			reply := parseReply(t, out[0])
			assert.JSONEq(t, `1`, string(reply.ID))
			require.NotNil(t, reply.Error)
			assert.Equal(t, codeUpstream, reply.Error.Code)
			assert.Contains(t, reply.Error.Message, tc.wantError)
			assert.Contains(t, errOut, tc.wantError)
		})
	}
}

// TestRunWritesNothingForFailedNotification: a failed notification has no request to answer, and a JSON body
// on it is not forwarded as if it were a response.
func TestRunWritesNothingForFailedNotification(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"bad"}}`)
	})

	out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")

	assert.Empty(t, out)
	assert.Contains(t, errOut, "HTTP 500")
}

// TestRunAnswersStreamsThatEndWithoutResponse: a stream that ends after only a notification, goes idle, or is cut
// is answered with an error, since the progress notification is not the answer.
func TestRunAnswersStreamsThatEndWithoutResponse(t *testing.T) {
	cases := []struct {
		name    string
		idle    time.Duration
		finish  http.HandlerFunc
		wantMsg string
	}{
		{
			name:    "stream ends",
			wantMsg: "no response",
			finish:  func(http.ResponseWriter, *http.Request) {},
		},
		{
			name:    "stream goes idle",
			idle:    100 * time.Millisecond,
			wantMsg: "no data from upstream",
			finish: func(w http.ResponseWriter, r *http.Request) {
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			},
		},
		{
			name:    "connection cut",
			wantMsg: "MCP upstream error",
			finish:  func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) },
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.idle > 0 {
				setIdleTimeout(t, tc.idle)
			}
			srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, "data: "+progressEvent+"\n\n")
				w.(http.Flusher).Flush()
				tc.finish(w, r)
			})

			out, _ := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":4,"method":"tools/call"}`+"\n")

			require.Len(t, out, 2)
			assert.Equal(t, progressEvent, out[0])
			reply := parseReply(t, out[1])
			assert.JSONEq(t, `4`, string(reply.ID))
			require.NotNil(t, reply.Error)
			assert.Equal(t, codeUpstream, reply.Error.Code)
			assert.Contains(t, reply.Error.Message, tc.wantMsg)
		})
	}
}

// TestRunHookOutlivesIdleTimeout: the hook for the idle error keeps its context after the idle timeout, so it
// is cancelled only by shutdown (or the client), and a hook that waits for its context does not hold Run.
func TestRunHookOutlivesIdleTimeout(t *testing.T) {
	setIdleTimeout(t, 100*time.Millisecond)
	setShutdownGrace(t, 100*time.Millisecond)
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(20 * time.Second):
		}
	})
	entered := make(chan struct{})
	cancelled := make(chan bool, 1)
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(ctx context.Context, _ []byte, _ int, msg []byte) ([][]byte, error) {
		close(entered)
		select {
		case <-ctx.Done():
			cancelled <- true
		case <-time.After(10 * time.Second):
			cancelled <- false
		}
		return [][]byte{msg}, nil
	})
	p := startPiped(t, t.Context(), b)

	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call"}`)
	waitClosed(t, entered, "the hook")
	require.NoError(t, p.finish(t))

	assert.True(t, <-cancelled, "the hook's context was cancelled by shutdown, not by the idle timeout")
	out := outLines(p.out.String())
	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	require.NotNil(t, reply.Error)
	assert.Contains(t, reply.Error.Message, "no data from upstream")
}

// expiringSession answers like a server whose first session has expired: tools/list gets 404 for session s1
// (for every session when always is set), and each initialize starts the next session.
func expiringSession(t *testing.T, always bool) (*httptest.Server, *fakeEndpoint) {
	t.Helper()
	var mu sync.Mutex
	starts := 0
	return newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case m.Method == "initialize":
			mu.Lock()
			starts++
			sid := fmt.Sprintf("s%d", starts)
			mu.Unlock()
			w.Header().Set("Mcp-Session-Id", sid)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case m.Method == "tools/list" && (always || r.Header.Get(sessionHeader) == "s1"):
			http.Error(w, "session not found", http.StatusNotFound)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[]}}`, m.ID)
		}
	})
}

// TestRunStartsNewSessionAfterSessionExpires: a 404 for a session the bridge had replays the client's
// initialize and notifications/initialized silently, then posts the request once more.
func TestRunStartsNewSessionAfterSessionExpires(t *testing.T) {
	const initLine = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`
	srv, fake := expiringSession(t, false)
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initLine)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	require.Len(t, out, 2, "the replayed handshake writes nothing to stdout")
	reply := parseReply(t, out[1])
	assert.JSONEq(t, `2`, string(reply.ID))
	assert.Nil(t, reply.Error)

	reqs := fake.requests()
	require.Len(t, reqs, 7)
	assert.Equal(t, "s1", reqs[2].Header.Get(sessionHeader), "the expired session was used first")
	assert.Equal(t, initLine, string(reqs[3].Body), "the client's initialize is replayed as sent")
	assert.Empty(t, reqs[3].Header.Get(sessionHeader))
	assert.Equal(t, "2025-06-18", reqs[4].Header.Get(protocolHeader), "the replayed handshake negotiates again")
	assert.Equal(t, "s2", reqs[4].Header.Get(sessionHeader))
	assert.Equal(t, "s2", reqs[5].Header.Get(sessionHeader))
	assert.Equal(t, http.MethodDelete, reqs[6].Method)
}

// TestRunAnswersRequestWhenNewSessionAlsoFails: the request is retried once, and an error that persists is
// answered with -32000.
func TestRunAnswersRequestWhenNewSessionAlsoFails(t *testing.T) {
	srv, fake := expiringSession(t, true)
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}`)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	require.Len(t, out, 2)
	reply := parseReply(t, out[1])
	assert.JSONEq(t, `2`, string(reply.ID))
	require.NotNil(t, reply.Error)
	assert.Equal(t, codeUpstream, reply.Error.Code)
	assert.Contains(t, reply.Error.Message, "HTTP 404")

	initializes := 0
	for _, r := range fake.requests() {
		if strings.Contains(string(r.Body), `"method":"initialize"`) {
			initializes++
		}
	}
	assert.Equal(t, 2, initializes, "the handshake is replayed once, not in a loop")
}

func TestRunKeepsRunningAfterMalformedLine(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answer)

	out, errOut := runAll(t, bridgeFor(srv.URL),
		"{not json\n"+`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	require.Len(t, out, 2)
	parseErr := parseReply(t, out[0])
	assert.JSONEq(t, `null`, string(parseErr.ID))
	require.NotNil(t, parseErr.Error)
	assert.Equal(t, codeParseError, parseErr.Error.Code)
	assert.JSONEq(t, `2`, string(parseReply(t, out[1]).ID))
	assert.Contains(t, errOut, "invalid JSON on stdin")
}

// TestRunRejectsValidJSONThatIsNotAnObject: valid JSON that is not a message object is an invalid request,
// not a parse error.
func TestRunRejectsValidJSONThatIsNotAnObject(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answer)

	out, errOut := runAll(t, bridgeFor(srv.URL), "[1,2]\n5\nnull\n")

	require.Len(t, out, 3)
	for _, line := range out {
		reply := parseReply(t, line)
		assert.JSONEq(t, `null`, string(reply.ID))
		require.NotNil(t, reply.Error)
		assert.Equal(t, codeInvalidRequest, reply.Error.Code)
	}
	assert.Contains(t, errOut, "invalid JSON-RPC message on stdin")
}

// TestRunReportsOversizedStdinMessage: a line over the limit ends the run with the limit named.
func TestRunReportsOversizedStdinMessage(t *testing.T) {
	var out, errOut syncBuffer
	b := bridgeFor("http://mcp.test/mcp")
	b.In = strings.NewReader(strings.Repeat("a", maxLineBytes+1))
	b.Out, b.Err = &out, &errOut

	err := b.Run(t.Context())

	require.ErrorIs(t, err, bufio.ErrTooLong)
	assert.Contains(t, err.Error(), "64 MiB")
}

func TestRunForwardsLargeMessages(t *testing.T) {
	const size = 8 << 20
	request := `{"jsonrpc":"2.0","id":1,"method":"echo","params":{"text":"` + strings.Repeat("a", size) + `"}}`
	response := `{"jsonrpc":"2.0","id":1,"result":{"text":"` + strings.Repeat("b", size) + `"}}`
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, response)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), request+"\n")

	reqs := fake.requests()
	require.Len(t, reqs, 1)
	assert.Equal(t, request, string(reqs[0].Body), "the request reaches the endpoint byte for byte")
	assert.Equal(t, []string{response}, out, "the response reaches stdout byte for byte")
}

func TestRunHandlesRequestsConcurrently(t *testing.T) {
	const n = 8
	var arrived sync.WaitGroup
	arrived.Add(n)
	allArrived := make(chan struct{})
	go func() {
		arrived.Wait()
		close(allArrived)
	}()
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		arrived.Done()
		select {
		case <-allArrived:
		case <-time.After(10 * time.Second):
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		answer(w, r)
	})
	var input strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&input, `{"jsonrpc":"2.0","id":%d,"method":"tools/call"}`+"\n", i)
	}

	out, _ := runAll(t, bridgeFor(srv.URL), input.String())

	require.Len(t, out, n)
	seen := map[string]bool{}
	for _, line := range out {
		reply := parseReply(t, line)
		assert.Nil(t, reply.Error)
		seen[string(reply.ID)] = true
	}
	assert.Len(t, seen, n)
}

func TestRunCancelsUpstreamRequestOnNotification(t *testing.T) {
	cancelled := make(chan struct{})
	started := make(chan struct{})
	markStarted := sync.OnceFunc(func() { close(started) })
	markCancelled := sync.OnceFunc(func() { close(cancelled) })
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		// The notification is also POSTed upstream, so only the first request blocks.
		body, _ := io.ReadAll(r.Body)
		if m, _ := parseMessage(body); m.Method != "tools/call" {
			return
		}
		markStarted()
		<-r.Context().Done()
		markCancelled()
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":7,"method":"tools/call"}`)
	<-started
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":7,"reason":"user"}}`)

	select {
	case <-cancelled:
	case <-time.After(10 * time.Second):
		t.Fatal("upstream request was not cancelled")
	}
	require.NoError(t, p.finish(t))
	assert.Empty(t, outLines(p.out.String()), "a cancelled request gets no response")
	assert.NotContains(t, p.err.String(), "upstream", "a cancellation the client asked for is not an error")
}

func TestRunShutsDownOnEOFAndDeletesSession(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Mcp-Session-Id", "s1")
		answer(w, r)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	p.waitLines(t, 1)
	require.NoError(t, p.finish(t))

	reqs := fake.requests()
	require.NotEmpty(t, reqs)
	assert.Equal(t, http.MethodDelete, reqs[len(reqs)-1].Method)
	assert.Equal(t, "s1", reqs[len(reqs)-1].Header.Get("Mcp-Session-Id"))
}

func TestRunShutsDownOnContextCancelAndDeletesSession(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		w.Header().Set("Mcp-Session-Id", "s1")
		answer(w, r)
	})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	p := startPiped(t, ctx, bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	p.waitLines(t, 1)
	cancel()

	select {
	case err := <-p.done:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
	reqs := fake.requests()
	require.NotEmpty(t, reqs)
	assert.Equal(t, http.MethodDelete, reqs[len(reqs)-1].Method)
	assert.Equal(t, "s1", reqs[len(reqs)-1].Header.Get("Mcp-Session-Id"))
}

func TestRunCancelsInFlightRequestsAfterShutdownGrace(t *testing.T) {
	setShutdownGrace(t, 100*time.Millisecond)
	cancelled := make(chan struct{})
	started := make(chan struct{})
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(cancelled)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`)
	<-started
	require.NoError(t, p.finish(t))

	select {
	case <-cancelled:
	case <-time.After(10 * time.Second):
		t.Fatal("in-flight request was not cancelled")
	}
	assert.Empty(t, outLines(p.out.String()))
}

// hookFunc adapts a function to Hooks.
type hookFunc func(ctx context.Context, request []byte, status int, msg []byte) ([][]byte, error)

func (f hookFunc) Result(ctx context.Context, request []byte, status int, msg []byte) ([][]byte, error) {
	return f(ctx, request, status, msg)
}

func TestRunHooksReplaceAndInjectMessages(t *testing.T) {
	const request = `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	const synthetic = `{"jsonrpc":"2.0","id":2,"result":{"tools":[]}}`
	const notification = `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	var gotRequest []byte
	var gotStatus int
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(_ context.Context, req []byte, status int, _ []byte) ([][]byte, error) {
		gotRequest, gotStatus = req, status
		return [][]byte{[]byte(synthetic), []byte(notification)}, nil
	})

	out, _ := runAll(t, b, request+"\n")

	assert.Equal(t, request, string(gotRequest))
	assert.Equal(t, http.StatusUnauthorized, gotStatus)
	assert.Equal(t, []string{synthetic, notification}, out)
}

// TestRunCompactsHookMessagesWithLineBreaks: a hook message with line breaks is still one stdout line.
func TestRunCompactsHookMessagesWithLineBreaks(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answer)
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(context.Context, []byte, int, []byte) ([][]byte, error) {
		return [][]byte{[]byte("{\n\"jsonrpc\": \"2.0\",\n\"id\": 3,\n\"result\": {}}")}, nil
	})

	out, _ := runAll(t, b, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{`{"jsonrpc":"2.0","id":3,"result":{}}`}, out)
}

func TestRunWritesHookErrorAsJSONRPCError(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answer)
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(context.Context, []byte, int, []byte) ([][]byte, error) {
		return nil, errors.New("hook failed")
	})

	out, errOut := runAll(t, b, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	require.NotNil(t, reply.Error)
	assert.Contains(t, reply.Error.Message, "hook failed")
	assert.Contains(t, errOut, "hook failed")
}

// endingServer is an endpoint whose first session ends on the first tools/list it sees, as a server does when it
// drops sessions. Each initialize starts the next session (s1, s2, ...) once beforeStart returns true for its
// number; false answers 503. Only a request that carries the current session id is answered, and any other
// request gets 404. onStale, when set, runs for each request that still carries s1 after s1 has ended.
type endingServer struct {
	beforeStart func(n int, r *http.Request) bool
	onStale     func()
}

func newEndingServer(t *testing.T, es endingServer) (*httptest.Server, *fakeEndpoint) {
	t.Helper()
	var mu sync.Mutex
	starts := 0
	current := ""
	return newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		if m.Method == methodInitialize {
			mu.Lock()
			starts++
			n := starts
			mu.Unlock()
			if es.beforeStart != nil && !es.beforeStart(n, r) {
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
				return
			}
			mu.Lock()
			current = fmt.Sprintf("s%d", n)
			sid := current
			mu.Unlock()
			w.Header().Set(sessionHeader, sid)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
			return
		}
		sid := r.Header.Get(sessionHeader)
		mu.Lock()
		if m.Method == "tools/list" && current == "s1" {
			current = ""
		}
		accepted := current != "" && sid == current
		mu.Unlock()
		if !accepted {
			if sid == "s1" && es.onStale != nil {
				es.onStale()
			}
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		if !m.isRequest() {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[]}}`, m.ID)
	})
}

const (
	initializeLine  = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`
	initializedLine = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
)

// sessionsSent returns the Mcp-Session-Id of each upstream POST whose body contains marker, in order.
func sessionsSent(fake *fakeEndpoint, marker string) []string {
	var sessions []string
	for _, r := range fake.requests() {
		if r.Method == http.MethodPost && strings.Contains(string(r.Body), marker) {
			sessions = append(sessions, r.Header.Get(sessionHeader))
		}
	}
	return sessions
}

// TestRunRecoversAfterFailedReplay: when the replayed initialize fails, the session id is kept, so the next
// request that hits the ended session starts a new one.
func TestRunRecoversAfterFailedReplay(t *testing.T) {
	srv, fake := newEndingServer(t, endingServer{beforeStart: func(n int, _ *http.Request) bool { return n != 2 }})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	p.waitLines(t, 2)
	p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	require.NoError(t, p.finish(t))

	out := outLines(p.out.String())
	require.Len(t, out, 3)
	failed := parseReply(t, out[1])
	assert.JSONEq(t, `2`, string(failed.ID))
	require.NotNil(t, failed.Error)
	assert.Contains(t, failed.Error.Message, "start a new session")
	recovered := parseReply(t, out[2])
	assert.JSONEq(t, `3`, string(recovered.ID))
	assert.Nil(t, recovered.Error)
	assert.Equal(t, []string{"s1", "s3"}, sessionsSent(fake, `"id":3,`), "the request first hits the ended session, then the new one")
}

// TestRunRecoversWhenReplayIsCancelled: the client cancels the request that started a replay. The replay stops
// without an answer for that request, and the next request starts a new session.
func TestRunRecoversWhenReplayIsCancelled(t *testing.T) {
	replaying := make(chan struct{})
	srv, _ := newEndingServer(t, endingServer{beforeStart: func(n int, r *http.Request) bool {
		if n == 2 {
			close(replaying)
			<-r.Context().Done()
			return false
		}
		return true
	}})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	waitClosed(t, replaying, "the replayed initialize")
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":2}}`)
	p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	out := outLines(p.out.String())
	require.Len(t, out, 2, "the cancelled request gets no answer")
	reply := parseReply(t, out[1])
	assert.JSONEq(t, `3`, string(reply.ID))
	assert.Nil(t, reply.Error)
}

// TestRunRequestDuringRejoinUsesNewSession: a request sent while a rejoin is running still uses the old session,
// gets 404, waits for the rejoin, and is posted again on the new session.
func TestRunRequestDuringRejoinUsesNewSession(t *testing.T) {
	replayStarted := make(chan struct{})
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	staleSeen := make(chan struct{})
	var stale atomic.Int32
	srv, fake := newEndingServer(t, endingServer{
		beforeStart: func(n int, _ *http.Request) bool {
			if n == 2 {
				close(replayStarted)
				<-release
			}
			return true
		},
		onStale: func() {
			if stale.Add(1) == 2 {
				close(staleSeen)
			}
		},
	})
	t.Cleanup(releaseOnce) // registered after the server, so it runs first and the held handler returns
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	waitClosed(t, replayStarted, "the replayed initialize")
	p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
	waitClosed(t, staleSeen, "the request sent during the rejoin to get its 404")
	releaseOnce()
	out := p.waitLines(t, 3)
	require.NoError(t, p.finish(t))

	for _, line := range out[1:] {
		assert.Nil(t, parseReply(t, line).Error, line)
	}
	assert.Equal(t, []string{"s1", "s2"}, sessionsSent(fake, `"id":3,`))
	assert.Equal(t, []string{"s1", "s2"}, sessionsSent(fake, `"id":2,`))
}

// TestRunAnswersRequestEndedBy404WithResponse: a 404 whose body is the server's response to the request is not an
// ended session, so the request is not posted again and no new session starts.
func TestRunAnswersRequestEndedBy404WithResponse(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":2,"error":{"code":-32001,"message":"unknown tool"}}`
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		switch m.Method {
		case methodInitialize:
			w.Header().Set(sessionHeader, "s1")
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
		case "tools/call":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, response)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/call"}`)
	p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	assert.Equal(t, []string{response}, outLines(p.out.String())[1:])
	assert.Len(t, sessionsSent(fake, `"method":"initialize"`), 1, "no new session was started")
	assert.Len(t, sessionsSent(fake, `"method":"tools/call"`), 1, "the request was posted once")
}

// TestRunMatchesResponseIDsByValue: a server may re-encode a request id, so the response is matched by the value
// of the id, and the protocol version is recorded from it.
func TestRunMatchesResponseIDsByValue(t *testing.T) {
	cases := []struct {
		name       string
		requestID  string
		responseID string
	}{
		{name: "string with an escaped character", requestID: `"a<b"`, responseID: `"a<b"`},
		{name: "number with a fraction", requestID: `7`, responseID: `7.0`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					return
				}
				body, _ := io.ReadAll(r.Body)
				m, _ := parseMessage(body)
				w.Header().Set("Content-Type", "application/json")
				if m.Method == methodInitialize {
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, tc.responseID)
					return
				}
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
			})
			p := startPiped(t, t.Context(), bridgeFor(srv.URL))

			p.send(t, `{"jsonrpc":"2.0","id":`+tc.requestID+`,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
			p.waitLines(t, 1)
			p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
			out := p.waitLines(t, 2)
			require.NoError(t, p.finish(t))

			assert.Equal(t, `{"jsonrpc":"2.0","id":`+tc.responseID+`,"result":{"protocolVersion":"2025-06-18"}}`, out[0])
			assert.Nil(t, parseReply(t, out[1]).Error)
			reqs := fake.requests()
			require.GreaterOrEqual(t, len(reqs), 2)
			assert.Equal(t, "2025-06-18", reqs[1].Header.Get(protocolHeader))
		})
	}
}

// TestRunStopsReadingStreamAfterResponse: once the response is written, the bridge stops reading the stream
// instead of waiting for the server to close it.
func TestRunStopsReadingStreamAfterResponse(t *testing.T) {
	streamEnded := make(chan struct{})
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: "+`{"jsonrpc":"2.0","id":1,"result":{}}`+"\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(20 * time.Second):
		}
		close(streamEnded)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	p.waitLines(t, 1)
	waitClosed(t, streamEnded, "the bridge to close the stream")
	require.NoError(t, p.finish(t))
	assert.Empty(t, p.err.String(), "no idle error for a request that succeeded")
}

// TestRunShutsDownOnEOFDuringStalledNotification: stdin EOF is seen while a notification's POST is stalled, and
// shutdown does not wait for the POST past the grace period.
func TestRunShutsDownOnEOFDuringStalledNotification(t *testing.T) {
	setShutdownGrace(t, 100*time.Millisecond)
	posted := make(chan struct{})
	markPosted := sync.OnceFunc(func() { close(posted) })
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		markPosted()
		<-r.Context().Done()
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializedLine)
	waitClosed(t, posted, "the notification to be posted")
	require.NoError(t, p.finish(t))
}

// TestRunDoesNotHoldRequestsBehindStalledNotification: a stalled notification delays the next request only for the
// notification wait, and the request is then posted and answered while the notification is still stalled.
func TestRunDoesNotHoldRequestsBehindStalledNotification(t *testing.T) {
	setNotificationWait(t, 100*time.Millisecond)
	setShutdownGrace(t, 100*time.Millisecond)
	release := make(chan struct{})
	defer close(release)
	stalled := make(chan struct{})
	markStalled := sync.OnceFunc(func() { close(stalled) })
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if m, _ := parseMessage(body); m.Method == "tools/list" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
			return
		}
		markStalled()
		select {
		case <-release:
		case <-r.Context().Done():
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializedLine)
	waitClosed(t, stalled, "the notification to be posted")
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 1)
	require.NoError(t, p.finish(t))

	reply := parseReply(t, out[0])
	assert.JSONEq(t, `2`, string(reply.ID))
	assert.Nil(t, reply.Error)
}

// TestRunWritesNothingForJSONBodyOfNotification: a notification has no response, so a JSON body on its POST is
// not written to stdout.
func TestRunWritesNothingForJSONBodyOfNotification(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), initializedLine+"\n")

	assert.Empty(t, out)
}

// TestRunAnswersJSONBodyThatIsNotItsResponse: a 2xx JSON body that does not answer the request is not written. The
// request gets one error that names what came back.
func TestRunAnswersJSONBodyThatIsNotItsResponse(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "empty object", body: `{}`},
		{name: "error with null id", body: `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"bad"}}`},
		{name: "response to another id", body: `{"jsonrpc":"2.0","id":9,"result":{}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			})

			out, _ := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

			require.Len(t, out, 1)
			reply := parseReply(t, out[0])
			assert.JSONEq(t, `1`, string(reply.ID))
			require.NotNil(t, reply.Error)
			assert.Equal(t, codeUpstream, reply.Error.Code)
			assert.Contains(t, reply.Error.Message, "instead of the response to the request")
		})
	}
}

// TestRunStopsReadingWhenContextIsCancelled: a line that is ready when ctx is cancelled is not sent upstream.
func TestRunStopsReadingWhenContextIsCancelled(t *testing.T) {
	srv, fake := newFakeEndpoint(t, answer)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var input strings.Builder
	for i := 1; i <= 50; i++ {
		fmt.Fprintf(&input, `{"jsonrpc":"2.0","id":%d,"method":"tools/list"}`+"\n", i)
	}
	var out, errOut syncBuffer
	b := bridgeFor(srv.URL)
	b.In, b.Out, b.Err = strings.NewReader(input.String()), &out, &errOut

	require.NoError(t, b.Run(ctx))

	assert.Empty(t, fake.requests())
	assert.Empty(t, outLines(out.String()))
}

// TestRunSendsInitializeWithoutSession: an initialize starts a session, so it never carries the session of an
// earlier one.
func TestRunSendsInitializeWithoutSession(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set(sessionHeader, "s1")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	reqs := fake.requests()
	require.Len(t, reqs, 3)
	assert.Empty(t, reqs[0].Header.Get(sessionHeader))
	assert.Empty(t, reqs[1].Header.Get(sessionHeader))
	assert.Empty(t, reqs[1].Header.Get(protocolHeader))
}
