package proxy_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResumeBackpressure covers a live peer whose acknowledgments arrive after the
// replay window fills, as happens during IDE startup through the driver proxy.
func TestResumeBackpressure(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	const window = 1 << 20
	payload := bytes.Repeat([]byte("0123456789abcdef"), window/2)
	received := make(chan []byte, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		stop := context.AfterFunc(ctx, func() { conn.Close() })
		defer stop()
		if !assert.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("SSH-2.0-test\r\n"))) {
			return
		}
		var data []byte
		defer func() { received <- data }()
		acked := 0
		for len(data) < len(payload) {
			mt, frame, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if mt == websocket.TextMessage {
				var ack struct {
					Delivered int `json:"delivered"`
				}
				if !assert.NoError(t, json.Unmarshal(frame, &ack)) {
					return
				}
				assert.Equal(t, len("SSH-2.0-test\r\n"), ack.Delivered)
				continue
			}
			if !assert.Equal(t, websocket.BinaryMessage, mt) {
				return
			}
			data = append(data, frame...)
			if len(data)-acked >= window {
				// Give the sender time to try to exceed its window before acknowledging.
				select {
				case <-time.After(50 * time.Millisecond):
				case <-ctx.Done():
					return
				}
				if err := conn.WriteJSON(map[string]int{"delivered": len(data)}); err != nil {
					return
				}
				acked = len(data)
			}
		}
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "finished"))
	}))
	defer server.Close()
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	go func() { _, _ = writer.Write(payload) }()
	err := proxy.RunClientProxy(ctx, input, io.Discard, func() <-chan time.Time { return nil }, time.Hour, true,
		func(ctx context.Context, req proxy.DialRequest) (*websocket.Conn, error) {
			conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+server.URL[4:], nil)
			if resp != nil {
				resp.Body.Close()
			}
			return conn, err
		})
	assert.NoError(t, err)
	cancel()
	select {
	case data := <-received:
		require.Len(t, data, len(payload), "the live peer received a truncated stream")
		assert.Equal(t, sha256.Sum256(payload), sha256.Sum256(data))
	case <-time.After(time.Second):
		t.Fatal("the peer did not finish")
	}
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}

func (b *lockedBuffer) bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return bytes.Clone(b.b.Bytes())
}

// TestResumeHandoverWithPendingAck exercises data already in flight when the
// handover owns the write lock. Reading the close frame must not wait on an ack.
func TestResumeHandoverWithPendingAck(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	initial := make(chan *websocket.Conn, 1)
	ready := make(chan struct{})
	payload := bytes.Repeat([]byte("x"), 64<<10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		stop := context.AfterFunc(ctx, func() { conn.Close() })
		defer stop()
		select {
		case old := <-initial:
			if !assert.NoError(t, old.WriteMessage(websocket.BinaryMessage, payload)) {
				return
			}
			if !assert.NoError(t, old.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "handover"))) {
				return
			}
			if !assert.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("after"))) {
				return
			}
		default:
			if !assert.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("before"))) {
				return
			}
			initial <- conn
			close(ready)
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	input, writer := io.Pipe()
	defer input.Close()
	defer writer.Close()
	var output lockedBuffer
	ticks := make(chan time.Time, 1)
	done := make(chan error, 1)
	go func() {
		done <- proxy.RunClientProxy(ctx, input, &output, func() <-chan time.Time { return ticks }, time.Hour, true,
			func(ctx context.Context, req proxy.DialRequest) (*websocket.Conn, error) {
				conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+server.URL[4:], nil)
				if resp != nil {
					resp.Body.Close()
				}
				return conn, err
			})
	}()
	<-ready
	ticks <- time.Now()
	want := append(append([]byte("before"), payload...), []byte("after")...)
	assert.Eventually(t, func() bool { return bytes.Equal(want, output.bytes()) }, 2*time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("client did not stop after cancellation")
	}
}
