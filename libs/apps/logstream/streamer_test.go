package logstream

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogStreamerTailBufferFlushes(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage() // search token

		for i := 1; i <= 3; i++ {
			require.NoError(t, sendEntry(conn, float64(i), fmt.Sprintf("msg%d", i)))
		}
		time.Sleep(50 * time.Millisecond)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "test",
		tail:     2,
		follow:   false,
		prefetch: 25 * time.Millisecond,
		writer:   buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2, "expected only last two log lines")
	assert.Contains(t, lines[0], "msg2")
	assert.Contains(t, lines[1], "msg3")
}

func TestLogStreamerTailFlushErrorPropagates(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()

		require.NoError(t, sendEntry(conn, 1, "msg1"))
		require.NoError(t, sendEntry(conn, 2, "msg2"))

		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	writerErr := errors.New("simulated write failure")
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "test",
		tail:     2,
		follow:   false,
		prefetch: 0,
		writer:   &failWriter{err: writerErr},
	}

	err := streamer.Run(context.Background())
	require.Error(t, err)
	assert.Equal(t, writerErr, err)
}

func TestLogStreamerTrimsCRLFInStructuredEntries(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		require.NoError(t, sendEntry(conn, 123, "line with crlf\r\n"))
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "token",
		writer: buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	output := buf.String()
	assert.Contains(t, output, "line with crlf")
	assert.NotContains(t, output, "\r")
}

func TestLogStreamerDialErrorIncludesResponseBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"FORBIDDEN","message":"token invalid"}`))
	}))
	defer server.Close()

	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "test",
		writer: &bytes.Buffer{},
	}

	err := streamer.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 403 Forbidden")
	assert.Contains(t, err.Error(), "token invalid")
}

func TestLogStreamerRetriesOnDialFailure(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		require.NoError(t, sendEntry(conn, float64(id), fmt.Sprintf("msg%d", id)))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer:   &flakyDialer{failures: 1, inner: &websocket.Dialer{}},
		url:      toWebSocketURL(server.URL),
		tail:     0,
		follow:   true,
		prefetch: 0,
		writer:   buf,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	require.ErrorIs(t, streamer.Run(ctx), context.DeadlineExceeded)
	assert.Contains(t, buf.String(), "msg1")
}

func TestLogStreamerSendsSearchTerm(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, msg, err := conn.ReadMessage()
		require.NoError(t, err)
		assert.Equal(t, "ERROR", string(msg))
		require.NoError(t, sendEntry(conn, 1, "boom"))
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "test",
		search: "ERROR",
		writer: buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	assert.Contains(t, buf.String(), "boom")
}

func TestLogStreamerFiltersSources(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		require.NoError(t, sendEntry(conn, 1, "app"))
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, mustJSON(wsEntry{Source: "SYSTEM", Timestamp: 2, Message: "sys"})))
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	sources := map[string]struct{}{"APP": {}}

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer:  &websocket.Dialer{},
		url:     toWebSocketURL(server.URL),
		token:   "test",
		sources: sources,
		writer:  buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	output := strings.TrimSpace(buf.String())
	assert.Contains(t, output, "app")
	assert.NotContains(t, output, "sys")
}

func TestFormatLogEntryColorizesWhenEnabled(t *testing.T) {
	original := color.NoColor
	color.NoColor = false
	defer func() { color.NoColor = original }()

	entry := &wsEntry{Source: "app", Timestamp: 1, Message: "hello\n"}
	streamer := &logStreamer{colorize: true}
	colored := streamer.formatLogEntry(entry)
	assert.Contains(t, colored, "\x1b[")
	assert.Contains(t, colored, fmt.Sprintf("[%s]", color.HiBlueString("APP")))

	streamer.colorize = false
	plain := streamer.formatLogEntry(entry)
	assert.NotContains(t, plain, "\x1b[")
	assert.Contains(t, plain, "[APP]")
}

func mustJSON(entry wsEntry) []byte {
	raw, err := json.Marshal(entry)
	if err != nil {
		panic(err)
	}
	return raw
}

func TestTailWithoutPrefetchRespectsTailSize(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		for i := 1; i <= 4; i++ {
			require.NoError(t, sendEntry(conn, float64(i), fmt.Sprintf("line%d", i)))
		}
		time.Sleep(20 * time.Millisecond)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "token",
		tail:     2,
		prefetch: 0,
		writer:   buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "line3")
	assert.Contains(t, lines[1], "line4")
}

func TestCloseErrorPropagatesWhenAbnormal(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4403, "auth failed"), time.Now().Add(time.Second))
	})
	defer server.Close()

	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "token",
	}

	err := streamer.Run(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "log stream closed with code 4403")
	assert.Contains(t, err.Error(), "auth failed")
}

type failWriter struct {
	err error
}

func (f *failWriter) Write([]byte) (int, error) {
	return 0, f.err
}

type flakyDialer struct {
	failures int32
	inner    Dialer
}

func (f *flakyDialer) DialContext(ctx context.Context, urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error) {
	if atomic.LoadInt32(&f.failures) > 0 {
		atomic.AddInt32(&f.failures, -1)
		return nil, nil, errors.New("transient dial failure")
	}
	return f.inner.DialContext(ctx, urlStr, requestHeader)
}

func newTestLogServer(t *testing.T, handler func(int, *websocket.Conn)) *httptest.Server {
	upgrader := websocket.Upgrader{}
	var connCount atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := int(connCount.Add(1))
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		go handler(id, conn)
	}))

	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})
	return server
}

func toWebSocketURL(raw string) string {
	return strings.Replace(raw, "http", "ws", 1)
}

func sendEntry(conn *websocket.Conn, ts float64, message string) error {
	payload, err := json.Marshal(wsEntry{
		Source:    "APP",
		Timestamp: ts,
		Message:   message,
	})
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func TestLogStreamerTailFlushesWithoutFollow(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		for i := 1; i <= 4; i++ {
			require.NoError(t, sendEntry(conn, float64(i), fmt.Sprintf("line%d", i)))
		}
		time.Sleep(250 * time.Millisecond)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	writer := newNotifyBuffer()
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "token",
		tail:     2,
		follow:   false,
		prefetch: 50 * time.Millisecond,
		writer:   writer,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streamer.Run(ctx)
	}()

	require.Eventually(t, writer.hasWrite, 150*time.Millisecond, 10*time.Millisecond, "expected tail logs to flush before the server closed the socket")
	require.NoError(t, <-done)
	lines := strings.Split(strings.TrimSpace(writer.String()), "\n")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "line3")
	assert.Contains(t, lines[1], "line4")
}

func TestLogStreamerFollowTailWithoutPrefetchEmitsRequestedLines(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		for i := 1; i <= 4; i++ {
			require.NoError(t, sendEntry(conn, float64(i), fmt.Sprintf("line%d", i)))
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	writer := newNotifyBuffer()
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "token",
		tail:     2,
		follow:   true,
		prefetch: 0,
		writer:   writer,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streamer.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return strings.Contains(writer.String(), "line4")
	}, time.Second, 10*time.Millisecond, "expected to see full tail even when prefetching is disabled")
	snapshot := writer.String()
	cancel()

	err := <-done
	require.ErrorIs(t, err, context.Canceled)
	lines := strings.Split(strings.TrimSpace(snapshot), "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	tail := lines[len(lines)-2:]
	assert.Contains(t, tail[0], "line3")
	assert.Contains(t, tail[1], "line4")
}

func TestLogStreamerFollowTailDoesNotReplayAfterReconnect(t *testing.T) {
	t.Parallel()

	stopCtx, stop := context.WithCancel(context.Background())
	defer stop()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		if id == 1 {
			for i := 1; i <= 4; i++ {
				require.NoError(t, sendEntry(conn, float64(i), fmt.Sprintf("line%d", i)))
			}
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
			return
		}

		require.NoError(t, sendEntry(conn, 5, "line5"))
		require.NoError(t, sendEntry(conn, 6, "line6"))
		<-stopCtx.Done()
	})
	defer server.Close()

	writer := newNotifyBuffer()
	streamer := &logStreamer{
		dialer:   &websocket.Dialer{},
		url:      toWebSocketURL(server.URL),
		token:    "token",
		tail:     2,
		follow:   true,
		prefetch: 0,
		writer:   writer,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streamer.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return strings.Contains(writer.String(), "line6")
	}, time.Second, 10*time.Millisecond, "expected logs from the second connection")

	cancel()
	stop()

	err := <-done
	require.ErrorIs(t, err, context.Canceled)

	output := writer.String()
	assert.Equal(t, 1, strings.Count(output, "line3"), "line3 emitted more than once")
	assert.Equal(t, 1, strings.Count(output, "line4"), "line4 emitted more than once")
	assert.Contains(t, output, "line5")
	assert.Contains(t, output, "line6")
}

func TestLogStreamerRefreshesTokenAfterAuthClose(t *testing.T) {
	t.Parallel()

	var connCount atomic.Int32
	upgrader := websocket.Upgrader{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := int(connCount.Add(1))
		auth := r.Header.Get("Authorization")
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		go func() {
			defer conn.Close()
			_, _, _ = conn.ReadMessage()
			if id == 1 {
				assert.Equal(t, "Bearer expired", auth)
				_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(4403, "auth failed"), time.Now().Add(time.Second))
				return
			}

			assert.Equal(t, "Bearer fresh", auth)
			require.NoError(t, sendEntry(conn, 1, "refreshed"))
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
		}()
	}))
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	var refreshes atomic.Int32
	tokenProvider := func(ctx context.Context) (string, error) {
		if refreshes.Load() > 0 {
			return "", errors.New("token refreshed multiple times")
		}
		refreshes.Add(1)
		return "fresh", nil
	}

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer:        &websocket.Dialer{},
		url:           toWebSocketURL(server.URL),
		token:         "expired",
		tokenProvider: tokenProvider,
		follow:        true,
		writer:        buf,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streamer.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), "refreshed")
	}, time.Second, 10*time.Millisecond, "expected logs after token refresh")

	cancel()

	err := <-done
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, int32(1), refreshes.Load(), "expected single token refresh")
}

func TestLogStreamerEmitsPlainTextFrames(t *testing.T) {
	t.Parallel()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte("plain text line")))
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	})
	defer server.Close()

	buf := &bytes.Buffer{}
	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "token",
		writer: buf,
	}

	require.NoError(t, streamer.Run(context.Background()))
	assert.Contains(t, buf.String(), "plain text line")
}

func TestLogStreamerTimeoutStopsQuietFollowStream(t *testing.T) {
	t.Parallel()

	stopCtx, stop := context.WithCancel(context.Background())
	defer stop()

	server := newTestLogServer(t, func(id int, conn *websocket.Conn) {
		defer conn.Close()
		_, _, _ = conn.ReadMessage()
		<-stopCtx.Done()
	})
	defer server.Close()

	streamer := &logStreamer{
		dialer: &websocket.Dialer{},
		url:    toWebSocketURL(server.URL),
		token:  "token",
		follow: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- streamer.Run(ctx)
	}()

	<-ctx.Done()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.DeadlineExceeded, "streamer should exit when context times out")
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("streamer did not exit within 200ms of context deadline")
	}
}

type notifyBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
	ch  chan struct{}
}

func newNotifyBuffer() *notifyBuffer {
	return &notifyBuffer{ch: make(chan struct{}, 1)}
}

func (n *notifyBuffer) Write(p []byte) (int, error) {
	n.mu.Lock()
	written, err := n.buf.Write(p)
	n.mu.Unlock()
	if err == nil {
		select {
		case n.ch <- struct{}{}:
		default:
		}
	}
	return written, err
}

func (n *notifyBuffer) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.buf.String()
}

func (n *notifyBuffer) hasWrite() bool {
	select {
	case <-n.ch:
		return true
	default:
		return false
	}
}
