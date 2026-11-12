package proxy

import (
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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testToken = "Bearer test-token"

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

// runAll runs b to completion with input as stdin and returns the stdout lines and stderr.
func runAll(t *testing.T, b *Bridge, input string) ([]string, string) {
	t.Helper()
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

func TestRunSendsNothingForAcceptedNotification(t *testing.T) {
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	out, errOut := runAll(t, bridgeFor(srv.URL), `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")

	assert.Empty(t, out)
	assert.Empty(t, errOut)
}

func TestRunForwardsSessionAndProtocolHeaders(t *testing.T) {
	srv, fake := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			return
		}
		body, _ := io.ReadAll(r.Body)
		m, _ := parseMessage(body)
		if m.Method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "s1")
		}
		answer(w, r)
	})
	p := startPiped(t, t.Context(), bridgeFor(srv.URL))

	p.send(t, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
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
		{name: "HTTP 500", status: http.StatusInternalServerError, body: "boom", wantError: "HTTP 500"},
		{name: "HTTP 401 without JSON", status: http.StatusUnauthorized, body: "unauthorized", wantError: "HTTP 401"},
		{name: "HTTP 401 with JSON-RPC body", status: http.StatusUnauthorized, body: passthrough, wantPassing: passthrough},
		{name: "no response", status: http.StatusAccepted, wantError: "no response"},
		{name: "invalid JSON", status: http.StatusOK, body: "<html>", wantError: "invalid JSON"},
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
			assert.Contains(t, errOut, "upstream")
		})
	}
}

func TestRunAnswersNetworkFailureWithJSONRPCError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	out, _ := runAll(t, bridgeFor(url), `{"jsonrpc":"2.0","id":7,"method":"tools/list"}`+"\n")

	require.Len(t, out, 1)
	reply := parseReply(t, out[0])
	assert.JSONEq(t, `7`, string(reply.ID))
	require.NotNil(t, reply.Error)
	assert.Equal(t, codeUpstream, reply.Error.Code)
	assert.Contains(t, reply.Error.Message, "request 7")
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

func TestRunForwardsLargeMessages(t *testing.T) {
	const size = 8 << 20
	line := `{"jsonrpc":"2.0","id":1,"method":"echo","params":{"text":"` + strings.Repeat("a", size) + `"}}`
	srv, _ := newFakeEndpoint(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(w, r.Body)
	})

	out, _ := runAll(t, bridgeFor(srv.URL), line+"\n")

	assert.Equal(t, []string{line}, out)
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
