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

	"github.com/databricks/databricks-sdk-go/config"
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
			name: "stream goes idle",
			// The progress event must arrive before the idle timeout, so the timeout is long enough for a slow runner.
			idle:    time.Second,
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

func (hookFunc) Local(context.Context, []byte, func([]byte)) ([][]byte, bool) {
	return nil, false
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

// TestRunAnswers404ResultWithoutNewSession: a 404 whose body is the server's result for the request is the answer,
// so the request is not posted again and no new session starts.
func TestRunAnswers404ResultWithoutNewSession(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":2,"result":{"content":[]}}`
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

// writeInit answers an initialize request with the given session id and protocol version 2025-06-18.
func writeInit(w http.ResponseWriter, m message, sid string) {
	w.Header().Set(contentTypeHeader, jsonMIME)
	w.Header().Set(sessionHeader, sid)
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
}

// TestRunRecoversAfterReplayAnswerIsAnError: an initialize answered with a JSON-RPC error is not a replay that
// succeeded, whether it comes with 200 or with a 401. The next request starts the replay again.
func TestRunRecoversAfterReplayAnswerIsAnError(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		wantError string
	}{
		{
			name:      "200 with a JSON-RPC error",
			status:    http.StatusOK,
			body:      `{"jsonrpc":"2.0","id":%s,"error":{"code":-32602,"message":"unsupported protocol version"}}`,
			wantError: "unsupported protocol version",
		},
		{
			name:      "401 with a JSON-RPC error",
			status:    http.StatusUnauthorized,
			body:      `{"jsonrpc":"2.0","id":%s,"error":{"code":-32006,"message":"sign in"}}`,
			wantError: "HTTP 401: sign in",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			starts := 0
			srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					return
				}
				body, _ := io.ReadAll(r.Body)
				m, _ := parseMessage(body)
				sid := r.Header.Get(sessionHeader)
				switch {
				case m.Method == methodInitialize:
					mu.Lock()
					starts++
					n := starts
					mu.Unlock()
					if n == 2 {
						w.Header().Set(contentTypeHeader, jsonMIME)
						w.WriteHeader(tc.status)
						fmt.Fprintf(w, tc.body, m.ID)
						return
					}
					writeInit(w, m, fmt.Sprintf("s%d", n))
				case !m.isRequest():
					w.WriteHeader(http.StatusAccepted)
				case sid == "s1":
					http.Error(w, "session not found", http.StatusNotFound)
				default:
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
				}
			})
			p := startPiped(t, t.Context(), bridgeFor(srv.URL))

			p.send(t, initializeLine)
			p.waitLines(t, 1)
			p.send(t, initializedLine)
			p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
			p.waitLines(t, 2)
			p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
			out := p.waitLines(t, 3)
			require.NoError(t, p.finish(t))

			failed := parseReply(t, out[1])
			assert.JSONEq(t, `2`, string(failed.ID))
			require.NotNil(t, failed.Error)
			assert.Contains(t, failed.Error.Message, "start a new session")
			assert.Contains(t, failed.Error.Message, tc.wantError)
			recovered := parseReply(t, out[2])
			assert.JSONEq(t, `3`, string(recovered.ID))
			assert.Nil(t, recovered.Error)
			assert.Equal(t, []string{"s1", "s3"}, sessionsSent(fake, `"id":3,`),
				"the second replay starts after the first failed; the request is posted to each session once")
		})
	}
}

// TestRunTakesSessionOnlyFromInitialize: a late answer on an ended session carries that session's id. The id must
// not replace the session that a rejoin has started.
func TestRunTakesSessionOnlyFromInitialize(t *testing.T) {
	held := make(chan struct{})
	release := make(chan struct{})
	releaseOnce := sync.OnceFunc(func() { close(release) })
	t.Cleanup(releaseOnce)
	var mu sync.Mutex
	starts := 0
	current := "s1"
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		if m.Method == methodInitialize {
			mu.Lock()
			starts++
			current = fmt.Sprintf("s%d", starts)
			sid := current
			mu.Unlock()
			writeInit(w, m, sid)
			return
		}
		if !m.isRequest() {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		mu.Lock()
		if string(m.ID) == "2" && current == "s1" && r.Header.Get(sessionHeader) == "s1" {
			current = "" // the server ends s1 on request 2
		}
		accepted := current != "" && r.Header.Get(sessionHeader) == current
		mu.Unlock()
		if !accepted {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		w.Header().Set(contentTypeHeader, jsonMIME)
		if string(m.ID) == "5" {
			close(held)
			<-release
			w.Header().Set(sessionHeader, "s1")
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":5,"method":"tools/call"}`)
	waitClosed(t, held, "request 5 to reach the endpoint")
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/call"}`)
	p.waitLines(t, 2)
	releaseOnce()
	p.waitLines(t, 3)
	p.send(t, `{"jsonrpc":"2.0","id":6,"method":"tools/call"}`)
	p.waitLines(t, 4)
	require.NoError(t, p.finish(t))

	assert.Equal(t, []string{"s2"}, sessionsSent(fake, `"id":6,`), "request 6 uses the session the rejoin started")
}

// TestRunDoesNotRejoinForNotificationOrClientResponse: a 404 for a notification or a client response is reported,
// and the session is not restarted to post it again.
func TestRunDoesNotRejoinForNotificationOrClientResponse(t *testing.T) {
	cases := []struct {
		name   string
		line   string
		marker string
	}{
		{name: "notification", line: initializedLine, marker: `"method":"notifications/initialized"`},
		{name: "client response", line: `{"jsonrpc":"2.0","id":9,"result":{}}`, marker: `"id":9,`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					return
				}
				body, _ := io.ReadAll(r.Body)
				m, _ := parseMessage(body)
				switch {
				case m.Method == methodInitialize:
					writeInit(w, m, "s1")
				case r.Header.Get(sessionHeader) == "s1":
					http.Error(w, "session not found", http.StatusNotFound)
				default:
					w.WriteHeader(http.StatusAccepted)
				}
			})

			p := startPiped(t, t.Context(), bridgeFor(srv.URL))
			p.send(t, initializeLine)
			p.waitLines(t, 1)
			p.send(t, tc.line)
			require.NoError(t, p.finish(t))

			require.Len(t, outLines(p.out.String()), 1, "only the initialize answer is written")
			assert.Contains(t, p.err.String(), "HTTP 404: session not found")
			assert.Len(t, sessionsSent(fake, `"method":"initialize"`), 1, "the session was not restarted")
			assert.Len(t, sessionsSent(fake, tc.marker), 1, "the message was posted once")
		})
	}
}

// TestRunSkipsEventsOfOtherTypes: an event whose type is not message is not a JSON-RPC message, so a keepalive does
// not end the request.
func TestRunSkipsEventsOfOtherTypes(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":1,"result":{}}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, "text/event-stream")
		_, _ = io.WriteString(w, "event: ping\ndata: ping\n\nevent: keepalive\ndata: {}\n\ndata: "+response+"\n\n")
	})

	out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{response}, out)
	assert.Empty(t, errOut)
}

// TestRunReadsEventStreamWithAnyCaseMediaType: the media type of a stream is case-insensitive.
func TestRunReadsEventStreamWithAnyCaseMediaType(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":1,"result":{}}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, "Text/Event-Stream; charset=utf-8")
		_, _ = io.WriteString(w, "data: "+response+"\n\n")
	})

	out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{response}, out)
	assert.Empty(t, errOut)
}

// TestRunDoesNotFollowRedirects: a redirect is a failed response with its status. Following it would post the
// message to the target, or turn a POST into a GET of a login page.
func TestRunDoesNotFollowRedirects(t *testing.T) {
	for _, status := range []int{
		http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var loginHit atomic.Bool
			srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/login" {
					loginHit.Store(true)
					w.Header().Set(contentTypeHeader, "text/html")
					_, _ = io.WriteString(w, "<html>sign in</html>")
					return
				}
				w.Header().Set("Location", "/login")
				w.WriteHeader(status)
			})

			out, _ := runAll(t, bridgeFor(srv.URL+"/mcp"), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

			require.Len(t, out, 1)
			reply := parseReply(t, out[0])
			require.NotNil(t, reply.Error)
			assert.Contains(t, reply.Error.Message, fmt.Sprintf("HTTP %d", status))
			assert.False(t, loginHit.Load(), "the redirect target was not requested")
		})
	}
}

// TestAwaitWaitsOnlyForGateOrContext: a request waits for the notification before it and nothing else. A timer of
// its own would release it while the notification is still pending.
func TestAwaitWaitsOnlyForGateOrContext(t *testing.T) {
	setNotificationWait(t, time.Millisecond)
	gate := make(chan struct{})
	returned := make(chan struct{})
	go func() {
		await(t.Context(), gate)
		close(returned)
	}()
	select {
	case <-returned:
		t.Fatal("await returned while the notification was still pending")
	case <-time.After(100 * time.Millisecond):
	}
	close(gate)
	waitClosed(t, returned, "await to return once the gate is closed")
}

// TestRunStartsNewSessionWhen404CarriesError: a session-not-found answer carries the request's id with an error, and it
// still ends the session. The request is answered by the new session.
func TestRunStartsNewSessionWhen404CarriesError(t *testing.T) {
	var mu sync.Mutex
	starts := 0
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		switch {
		case m.Method == methodInitialize:
			mu.Lock()
			starts++
			sid := fmt.Sprintf("s%d", starts)
			mu.Unlock()
			writeInit(w, m, sid)
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case r.Header.Get(sessionHeader) == "s1":
			w.Header().Set(contentTypeHeader, jsonMIME)
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32001,"message":"Session not found"}}`, m.ID)
		default:
			w.Header().Set(contentTypeHeader, jsonMIME)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	reply := parseReply(t, out[1])
	assert.JSONEq(t, `2`, string(reply.ID))
	assert.Nil(t, reply.Error, out[1])
	assert.Equal(t, []string{"s1", "s2"}, sessionsSent(fake, `"id":2,`))
}

// TestRunCancelsRequestByExactID: request ids that differ only above 2^53 are distinct, so cancelling one does not
// cancel the other.
func TestRunCancelsRequestByExactID(t *testing.T) {
	const idA = "9007199254740993" // 2^53 + 1
	const idB = "9007199254740992" // 2^53
	started := make(chan struct{})
	cancelled := make(chan struct{})
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		if !m.isRequest() {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set(contentTypeHeader, jsonMIME)
		if string(m.ID) == idA {
			close(started)
			<-r.Context().Done()
			close(cancelled)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":`+idA+`,"method":"tools/call"}`)
	waitClosed(t, started, "request A to reach the endpoint")
	p.send(t, `{"jsonrpc":"2.0","id":`+idB+`,"method":"tools/call"}`)
	out := p.waitLines(t, 1)
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":`+idA+`}}`)
	waitClosed(t, cancelled, "request A to be cancelled")
	require.NoError(t, p.finish(t))

	require.Len(t, out, 1)
	assert.JSONEq(t, idB, string(parseReply(t, out[0]).ID))
}

// TestRunReportsStatusOfRetriedRequest: a retry that fails before any response reports status 0 to the hooks, not
// the 404 of the first attempt.
func TestRunReportsStatusOfRetriedRequest(t *testing.T) {
	statuses := make(chan int, 2)
	var mu sync.Mutex
	starts := 0
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		switch {
		case m.Method == methodInitialize:
			mu.Lock()
			starts++
			sid := fmt.Sprintf("s%d", starts)
			mu.Unlock()
			writeInit(w, m, sid)
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case r.Header.Get(sessionHeader) == "s1":
			http.Error(w, "session not found", http.StatusNotFound)
		default:
			panic(http.ErrAbortHandler) // the connection is cut before any response
		}
	})
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(_ context.Context, _ []byte, status int, msg []byte) ([][]byte, error) {
		if bytes.Contains(msg, []byte(`"error"`)) {
			statuses <- status
		}
		return [][]byte{msg}, nil
	})
	p := startPiped(t, t.Context(), b)

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	assert.Contains(t, parseReply(t, out[1]).Error.Message, "MCP upstream error")
	assert.Equal(t, 0, <-statuses, "the retry had no response")
}

// TestRunDeletesSessionOfFailedReplay: a replay whose initialize created a session and whose initialized
// notification failed leaves that session on the server, so the bridge deletes it.
func TestRunDeletesSessionOfFailedReplay(t *testing.T) {
	var mu sync.Mutex
	starts := 0
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		sid := r.Header.Get(sessionHeader)
		switch {
		case m.Method == methodInitialize:
			mu.Lock()
			starts++
			n := starts
			mu.Unlock()
			writeInit(w, m, fmt.Sprintf("s%d", n))
		case m.Method == methodInitialized && sid == "s2":
			http.Error(w, "boom", http.StatusInternalServerError)
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case sid == "s1":
			http.Error(w, "session not found", http.StatusNotFound)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	assert.Contains(t, parseReply(t, out[1]).Error.Message, "start a new session")
	var deleted []string
	for _, r := range fake.requests() {
		if r.Method == http.MethodDelete {
			deleted = append(deleted, r.Header.Get(sessionHeader))
		}
	}
	assert.Contains(t, deleted, "s2", "the session of the failed replay is deleted")
}

// TestRunUsesConfiguredTransport: the upstream requests go through Transport, so a profile's TLS settings apply.
func TestRunUsesConfiguredTransport(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(answer))
	t.Cleanup(srv.Close)
	b := bridgeFor(srv.URL)
	b.Transport = srv.Client().Transport

	out, errOut := runAll(t, b, `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	assert.JSONEq(t, `1`, string(parseReply(t, out[0]).ID))
	assert.Empty(t, errOut)
}

// TestLockRejoinHonorsContext: a request queued behind a rejoin that is still running returns when its context is
// cancelled, rather than waiting for the rejoin.
func TestLockRejoinHonorsContext(t *testing.T) {
	c := &conn{rejoinSlot: make(chan struct{}, 1)}
	c.rejoinSlot <- struct{}{} // a rejoin is running
	ctx, cancel := context.WithCancel(t.Context())
	locked := make(chan error, 1)
	go func() { locked <- c.lockRejoin(ctx) }()
	cancel()

	select {
	case err := <-locked:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(10 * time.Second):
		t.Fatal("lockRejoin did not return after the context was cancelled")
	}
}

// TestRunAnswersOversizedUpstreamBody: a 2xx JSON body is limited like a stdin message, so an oversized one is an
// error for its request and is not written.
func TestRunAnswersOversizedUpstreamBody(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, jsonMIME)
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":1,"result":{"text":"`+strings.Repeat("a", maxLineBytes)+`"}}`)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	require.NotNil(t, reply.Error)
	assert.Contains(t, reply.Error.Message, "64 MiB limit")
}

// TestRunAnswersPingDuringReplay: a server may ping the client before it answers the replayed initialize, and it waits
// for the ping answer before it sends that answer. The answer is posted on the session being started.
func TestRunAnswersPingDuringReplay(t *testing.T) {
	const pingID = `"ping-1"`
	pingAnswered := make(chan struct{})
	markAnswered := sync.OnceFunc(func() { close(pingAnswered) })
	var mu sync.Mutex
	starts := 0
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		switch {
		case m.Method == "" && string(m.ID) == pingID:
			markAnswered()
			w.WriteHeader(http.StatusAccepted)
		case m.Method == methodInitialize:
			mu.Lock()
			starts++
			n := starts
			mu.Unlock()
			if n == 1 {
				writeInit(w, m, "s1")
				return
			}
			w.Header().Set(contentTypeHeader, eventStreamMIME)
			w.Header().Set(sessionHeader, "s2")
			_, _ = io.WriteString(w, "data: "+`{"jsonrpc":"2.0","id":`+pingID+`,"method":"ping"}`+"\n\n")
			w.(http.Flusher).Flush()
			select {
			case <-pingAnswered:
			case <-time.After(5 * time.Second):
			}
			fmt.Fprintf(w, "data: "+`{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`+"\n\n", m.ID)
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case r.Header.Get(sessionHeader) == "s1":
			http.Error(w, "session not found", http.StatusNotFound)
		default:
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	require.Len(t, out, 2)
	assert.Nil(t, parseReply(t, out[1]).Error)
	assert.Equal(t, []string{"s2"}, sessionsSent(fake, pingID), "the ping was answered on the session being started")
}

// TestRunPassesThroughErrorOf404WithoutSession: without a session id a 404 cannot mean the session ended, so a JSON-RPC
// error for the request is passed through as the server sent it.
func TestRunPassesThroughErrorOf404WithoutSession(t *testing.T) {
	const errFormat = `{"jsonrpc":"2.0","id":%s,"error":{"code":-32002,"message":"Resource not found","data":{"uri":"x"}}}`
	cases := []struct {
		name string
		// initOK makes initialize succeed without a session id; otherwise every request gets the 404.
		initOK bool
		input  string
		marker string // the request whose POST count is checked
		want   []string
	}{
		{
			name:   "initialize",
			input:  initializeLine + "\n",
			marker: `"method":"initialize"`,
			want:   []string{`{"jsonrpc":"2.0","id":1,"error":{"code":-32002,"message":"Resource not found","data":{"uri":"x"}}}`},
		},
		{
			name:   "request after an initialize that set no session",
			initOK: true,
			input:  initializeLine + "\n" + `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n",
			marker: `"method":"tools/list"`,
			want: []string{
				`{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18"}}`,
				`{"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"Resource not found","data":{"uri":"x"}}}`,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				m, _ := parseMessage(body)
				w.Header().Set(contentTypeHeader, jsonMIME)
				if m.Method == methodInitialize && tc.initOK {
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
					return
				}
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, errFormat, m.ID)
			})

			out, errOut := runAll(t, bridgeFor(srv.URL), tc.input)

			assert.ElementsMatch(t, tc.want, out, "requests run concurrently, so the order of answers is not fixed")
			assert.Empty(t, errOut)
			assert.Len(t, sessionsSent(fake, tc.marker), 1, "a request without a session id is not posted again")
		})
	}
}

// TestRunRecordsSessionOnlyFromInitializeResult: an initialize answer without a result starts no session, so the
// session header it carries is not sent with later requests.
func TestRunRecordsSessionOnlyFromInitializeResult(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		w.Header().Set(contentTypeHeader, jsonMIME)
		if m.Method == methodInitialize {
			w.Header().Set(sessionHeader, "s1")
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32600,"message":"not now"}}`, m.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), initializeLine+"\n"+`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	require.Len(t, out, 2)
	assert.Equal(t, []string{""}, sessionsSent(fake, `"id":2,`))
	assert.Empty(t, deletedSessions(fake), "no session was started, so none is deleted")
}

// Authenticate returns ErrSkipRequest for the session DELETE: no DELETE is sent, and nothing is logged, since the
// session is over.
func TestRunSkipsSessionDeleteWhenAuthenticateSkipsIt(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Mcp-Session-Id", "s1")
		answer(w, r)
	})
	b := bridgeFor(srv.URL)
	authenticate := b.Authenticate
	b.Authenticate = func(r *http.Request) error {
		if r.Method == http.MethodDelete {
			return ErrSkipRequest
		}
		return authenticate(r)
	}

	_, errOut := runAll(t, b, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`+"\n")

	assert.Empty(t, deletedSessions(fake))
	assert.Empty(t, errOut)
}

// deletedSessions returns the Mcp-Session-Id of each DELETE the fake endpoint received, in order.
func deletedSessions(fake *fakeEndpoint) []string {
	var sessions []string
	for _, r := range fake.requests() {
		if r.Method == http.MethodDelete {
			sessions = append(sessions, r.Header.Get(sessionHeader))
		}
	}
	return sessions
}

// TestRunDeletesSessionReplacedByInitialize: a second initialize whose answer has a new session ends the session it
// replaces. An answer with the same session keeps it.
func TestRunDeletesSessionReplacedByInitialize(t *testing.T) {
	cases := []struct {
		name   string
		second string // the session of the second initialize's answer
		status int    // the status of the second answer; 0 is 200
		// wantSent is the session the request after the second initialize is sent with.
		wantSent    string
		wantDeleted []string // the replaced session, then the one shutdown ends
	}{
		{name: "new session", second: "s2", wantSent: "s2", wantDeleted: []string{"s1", "s2"}},
		{name: "same session", second: "s1", wantSent: "s1", wantDeleted: []string{"s1"}},
		// A failed answer with a result starts no session, so the current one stays and is not deleted.
		{name: "failed answer keeps session", second: "s2", status: http.StatusInternalServerError, wantSent: "s1", wantDeleted: []string{"s1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			starts := 0
			srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					return
				}
				body, _ := io.ReadAll(r.Body)
				m, _ := parseMessage(body)
				if m.Method == methodInitialize {
					mu.Lock()
					starts++
					n := starts
					mu.Unlock()
					if n == 1 {
						writeInit(w, m, "s1")
						return
					}
					w.Header().Set(contentTypeHeader, jsonMIME)
					w.Header().Set(sessionHeader, tc.second)
					if tc.status != 0 {
						w.WriteHeader(tc.status)
					}
					fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`, m.ID)
					return
				}
				if !m.isRequest() {
					w.WriteHeader(http.StatusAccepted)
					return
				}
				w.Header().Set(contentTypeHeader, jsonMIME)
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
			})
			p := startPiped(t, t.Context(), bridgeFor(srv.URL))

			p.send(t, initializeLine)
			p.waitLines(t, 1)
			p.send(t, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
			p.waitLines(t, 2)
			p.send(t, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`)
			p.waitLines(t, 3)
			require.NoError(t, p.finish(t))

			assert.Equal(t, []string{tc.wantSent}, sessionsSent(fake, `"id":3,`))
			assert.Equal(t, tc.wantDeleted, deletedSessions(fake))
		})
	}
}

// TestRunHoldsRequestUntilNotificationWaitFromItsPost: a notification's gate opens notificationWait after its own POST
// starts, not after the message is read. The second notification waits for a stalled first one, and the request
// waits for the second one's gate, so the request is posted about one wait after the second notification.
func TestRunHoldsRequestUntilNotificationWaitFromItsPost(t *testing.T) {
	const wait = 400 * time.Millisecond
	setNotificationWait(t, wait)
	setShutdownGrace(t, 100*time.Millisecond)
	firstPosted := make(chan struct{})
	secondPosted := make(chan time.Time, 1)
	requested := make(chan time.Time, 1)
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		switch m.Method {
		case "notifications/first":
			close(firstPosted)
			<-r.Context().Done()
		case "notifications/second":
			secondPosted <- time.Now()
			<-r.Context().Done()
		case "tools/list":
			requested <- time.Now()
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","method":"notifications/first"}`)
	waitClosed(t, firstPosted, "the first notification to be posted")
	p.send(t, `{"jsonrpc":"2.0","method":"notifications/second"}`)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 1)
	require.NoError(t, p.finish(t))

	require.Len(t, out, 1)
	assert.Nil(t, parseReply(t, out[0]).Error)
	secondAt := <-secondPosted
	requestedAt := <-requested
	assert.GreaterOrEqual(t, requestedAt.Sub(secondAt), wait/2, "the request did not wait for the second notification's gate")
}

// TestRunWritesNothingWhenHookOutputIsInvalid: when a hook returns a message that is not JSON after a valid one, nothing
// is written for the response, and the request gets one error.
func TestRunWritesNothingWhenHookOutputIsInvalid(t *testing.T) {
	srv, _ := newFakeEndpoint(t, answer)
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(_ context.Context, _ []byte, _ int, msg []byte) ([][]byte, error) {
		if bytes.Contains(msg, []byte(`"result"`)) {
			return [][]byte{msg, []byte("{\n\"jsonrpc\":")}, nil
		}
		return [][]byte{msg}, nil
	})

	out, errOut := runAll(t, b, `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	assert.JSONEq(t, `3`, string(reply.ID))
	require.NotNil(t, reply.Error)
	assert.Equal(t, codeUpstream, reply.Error.Code)
	assert.Contains(t, errOut, "upstream:")
}

// TestRunKeepsReadingWhenHookFailsOnProgress: a hook error on a message other than the response is logged and that
// message is dropped. The response still comes.
func TestRunKeepsReadingWhenHookFailsOnProgress(t *testing.T) {
	const response = `{"jsonrpc":"2.0","id":1,"result":{}}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, eventStreamMIME)
		_, _ = io.WriteString(w, "data: "+progressEvent+"\n\ndata: "+response+"\n\n")
	})
	b := bridgeFor(srv.URL)
	b.Hooks = hookFunc(func(_ context.Context, _ []byte, _ int, msg []byte) ([][]byte, error) {
		if bytes.Contains(msg, []byte("notifications/progress")) {
			return nil, errors.New("progress hook failed")
		}
		return [][]byte{msg}, nil
	})

	out, errOut := runAll(t, b, `{"jsonrpc":"2.0","id":1,"method":"tools/call"}`+"\n")

	assert.Equal(t, []string{response}, out)
	assert.Contains(t, errOut, "progress hook failed")
}

// TestRunDoesNotReadStreamOfNotification: the answer to a notification has no body. An event stream on it is closed
// unread, so nothing is forwarded and the connection is not held open.
func TestRunDoesNotReadStreamOfNotification(t *testing.T) {
	closed := make(chan struct{})
	markClosed := sync.OnceFunc(func() { close(closed) })
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(contentTypeHeader, eventStreamMIME)
		_, _ = io.WriteString(w, "data: "+progressEvent+"\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		markClosed()
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializedLine)
	waitClosed(t, closed, "the bridge to close the stream")
	require.NoError(t, p.finish(t))

	assert.Empty(t, outLines(p.out.String()))
}

// TestRunClosesStaleResponseBeforeReplay: the 404 that ended a session is closed before the replay starts, so its
// connection does not wait for the replay. The body is longer than the 1 MiB the bridge reads, so it stays open until
// the bridge closes it.
func TestRunClosesStaleResponseBeforeReplay(t *testing.T) {
	staleClosed := make(chan struct{})
	markStaleClosed := sync.OnceFunc(func() { close(staleClosed) })
	var sawClosed atomic.Bool
	var mu sync.Mutex
	starts := 0
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		switch {
		case m.Method == methodInitialize:
			mu.Lock()
			starts++
			n := starts
			mu.Unlock()
			if n == 2 {
				select {
				case <-staleClosed:
					sawClosed.Store(true)
				case <-time.After(5 * time.Second):
				}
			}
			writeInit(w, m, fmt.Sprintf("s%d", n))
		case !m.isRequest():
			w.WriteHeader(http.StatusAccepted)
		case r.Header.Get(sessionHeader) == "s1":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, strings.Repeat("x", maxErrorBytes+1))
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			markStaleClosed()
		default:
			w.Header().Set(contentTypeHeader, jsonMIME)
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, m.ID)
		}
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, initializeLine)
	p.waitLines(t, 1)
	p.send(t, initializedLine)
	p.send(t, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	out := p.waitLines(t, 2)
	require.NoError(t, p.finish(t))

	assert.True(t, sawClosed.Load(), "the stale response was closed before the replay")
	require.Len(t, out, 2)
	assert.Nil(t, parseReply(t, out[1]).Error)
}

// answerHooks answers a request whose body names "local" with msgs, and forwards the others.
type answerHooks struct{ msgs [][]byte }

func (answerHooks) Result(_ context.Context, _ []byte, _ int, msg []byte) ([][]byte, error) {
	return [][]byte{msg}, nil
}

func (h answerHooks) Local(_ context.Context, request []byte, _ func([]byte)) ([][]byte, bool) {
	if !bytes.Contains(request, []byte("local")) {
		return nil, false
	}
	return h.msgs, true
}

// TestAnswerLocallyWritesHookMessagesInOrder checks that a request the hooks handle is answered with their messages, one
// line each and in order, and that nothing is written for a request they do not handle.
func TestAnswerLocallyWritesHookMessagesInOrder(t *testing.T) {
	var out, errOut syncBuffer
	c := &conn{b: &Bridge{Out: &out, Err: &errOut, Hooks: answerHooks{msgs: [][]byte{[]byte(`{"n":1}`), []byte(`{"n":2}`)}}}}

	assert.True(t, c.answerLocally(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"local"}`)))
	assert.Equal(t, "{\"n\":1}\n{\"n\":2}\n", out.String())

	assert.False(t, c.answerLocally(t.Context(), []byte(`{"jsonrpc":"2.0","id":2,"method":"remote"}`)))
	assert.Equal(t, "{\"n\":1}\n{\"n\":2}\n", out.String())
}

// TestAnswerLocallyWithoutHooksForwardsEverything checks that a bridge without hooks handles no request.
func TestAnswerLocallyWithoutHooksForwardsEverything(t *testing.T) {
	var out syncBuffer
	c := &conn{b: &Bridge{Out: &out, Err: &syncBuffer{}}}

	assert.False(t, c.answerLocally(t.Context(), []byte(`{"jsonrpc":"2.0","id":1,"method":"local"}`)))
	assert.Empty(t, out.String())
}

// scriptedHooks answers every request with msgs, after notify has been called with notify, when handled is true. Upstream
// messages pass through.
type scriptedHooks struct {
	notify  []byte
	msgs    [][]byte
	handled bool
}

func (h scriptedHooks) Local(_ context.Context, _ []byte, notify func([]byte)) ([][]byte, bool) {
	if h.notify != nil {
		notify(h.notify)
	}
	return h.msgs, h.handled
}

func (scriptedHooks) Result(_ context.Context, _ []byte, _ int, msg []byte) ([][]byte, error) {
	return [][]byte{msg}, nil
}

// TestAnswerLocallyCompactsMessages checks that the messages a hook writes itself, its progress and its answer, are one
// stdout line each, as an upstream answer is.
func TestAnswerLocallyCompactsMessages(t *testing.T) {
	b := bridgeFor("http://unused.test")
	b.Hooks = scriptedHooks{
		notify:  []byte("{\n\"jsonrpc\": \"2.0\",\n\"method\": \"notifications/progress\"}"),
		msgs:    [][]byte{[]byte("{\n\"jsonrpc\": \"2.0\",\n\"id\": 3,\n\"result\": {}}")},
		handled: true,
	}

	out, _ := runAll(t, b, `{"jsonrpc":"2.0","id":3,"method":"tools/call"}`+"\n")

	assert.Equal(t, []string{
		`{"jsonrpc":"2.0","method":"notifications/progress"}`,
		`{"jsonrpc":"2.0","id":3,"result":{}}`,
	}, out)
}

// TestAnswerLocallyWritesNothingWhenNotHandled checks that a request a hook does not handle is forwarded, and that the
// hook's messages for it are not written.
func TestAnswerLocallyWritesNothingWhenNotHandled(t *testing.T) {
	srv, fake := newFakeEndpoint(t, answerWith(http.StatusOK, listTools))
	b := bridgeFor(srv.URL)
	b.Hooks = scriptedHooks{msgs: [][]byte{[]byte(listChangedMsg)}, handled: false}

	out, _ := runAll(t, b, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")

	assert.Equal(t, []string{listTools}, out)
	assert.Len(t, fake.requests(), 1, "the request is forwarded")
}

// TestSignInWritesShareTheBridgeStderr checks that the progress a sign-in writes to its stderr is serialized with the
// bridge's own diagnostics. The stderr here is a plain buffer, so a write that is not serialized is a data race, which
// the race detector reports.
func TestSignInWritesShareTheBridgeStderr(t *testing.T) {
	const progressLines = 200
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if m, _ := parseMessage(body); m.Method == "tools/list" {
			answerWith(http.StatusOK, listChallenge)(w, r)
			return
		}
		// A body that is not JSON is logged by the bridge, once per request.
		answerWith(http.StatusOK, "not json")(w, r)
	})
	signIn := func(_ context.Context, req SignInRequest) (bool, bool, error) {
		for i := range progressLines {
			fmt.Fprintf(req.Stderr, "sign-in progress %d\n", i)
		}
		return false, true, nil
	}
	var stderr bytes.Buffer
	var stdout syncBuffer
	in, pw := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Run(t.Context(), Options{
			In:           in,
			Out:          &stdout,
			Err:          &stderr,
			Endpoint:     srv.URL,
			Config:       &config.Config{Host: testHost},
			Authenticate: func(*http.Request) error { return nil },
			Profile:      "logfood",
			Challenge:    &Challenge{Service: testService, Profile: "logfood", Host: testHost, Label: "GitHub", LoginCommand: loginCommand, SignIn: signIn},
		})
	}()

	// The session is challenged once the list is answered, so the authenticate call waits for that answer.
	fmt.Fprintln(pw, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	require.Eventually(t, func() bool { return strings.Contains(stdout.String(), `"authenticate"`) }, 10*time.Second, time.Millisecond)
	fmt.Fprintln(pw, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"authenticate","arguments":{}}}`)
	for id := 10; id < 30; id++ {
		fmt.Fprintf(pw, `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"echo","arguments":{}}}`+"\n", id)
	}
	require.NoError(t, pw.Close())
	require.NoError(t, <-done)

	var progress int
	for _, line := range strings.Split(stderr.String(), "\n") {
		if strings.HasPrefix(line, "sign-in progress ") {
			progress++
		}
	}
	assert.Equal(t, progressLines, progress, "every progress line is written whole")
}
