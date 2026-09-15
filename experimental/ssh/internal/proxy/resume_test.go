package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResumeRetriesADroppedHandshake(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		if attempts.Add(1) == 1 {
			return
		}
		assert.NoError(t, conn.WriteJSON(controlMessage{}))
		<-ctx.Done()
	}))
	defer server.Close()
	defer cancel()
	pc := newResumableProxyConnection(func(ctx context.Context, req DialRequest) (*websocket.Conn, error) {
		conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+server.URL[4:], nil)
		if resp != nil {
			resp.Body.Close()
		}
		return conn, err
	})
	require.NoError(t, pc.dialReattach(ctx))
	defer func() { _ = pc.closeConnection() }()
	assert.Equal(t, int32(2), attempts.Load())
}

func TestResumeDoesNotRetryARejectedSession(t *testing.T) {
	var attempts int
	pc := newResumableProxyConnection(func(ctx context.Context, req DialRequest) (*websocket.Conn, error) {
		attempts++
		return nil, ErrReattachRejected
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	err := pc.dialReattach(ctx)
	assert.ErrorIs(t, err, ErrReattachRejected)
	assert.NoError(t, ctx.Err())
	assert.Equal(t, 1, attempts)
}

func TestSendBufferWaitForSpace(t *testing.T) {
	for _, cancelWait := range []bool{false, true} {
		t.Run(fmt.Sprintf("cancel=%t", cancelWait), func(t *testing.T) {
			b := newSendBuffer(8)
			require.NoError(t, b.append([]byte("12345678")))
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- b.waitForSpace(ctx, 1) }()
			select {
			case err := <-done:
				t.Fatalf("full window did not block: %v", err)
			case <-time.After(20 * time.Millisecond):
			}
			if cancelWait {
				cancel()
			} else {
				b.ack(1)
			}
			select {
			case err := <-done:
				assert.Equal(t, cancelWait, errors.Is(err, context.Canceled))
			case <-time.After(time.Second):
				t.Fatal("waiting producer did not wake up")
			}
		})
	}
}

func TestParseDialRequestRequiresCorrectedResumeProtocol(t *testing.T) {
	for _, tc := range []struct {
		query      string
		wantResume bool
		wantError  bool
	}{
		{"id=test", false, false},
		{"id=test&delivered=0", false, true},
		{"id=test&resume_version=1&delivered=0", false, true},
		{"id=test&resume_version=2&delivered=0", true, false},
		{"id=test&resume_version=2&delivered=-1", false, true},
		{"id=test&resume_version=2", false, true},
		{"id=test&reattach=1", false, true},
	} {
		t.Run(tc.query, func(t *testing.T) {
			req, err := parseDialRequest(httptest.NewRequest(http.MethodGet, "/?"+tc.query, nil))
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantResume, req.ResumeCapable)
		})
	}
}

func TestResumeCancellationInterruptsHandshake(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	ready := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if !assert.NoError(t, err) {
			return
		}
		defer conn.Close()
		close(ready)
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()
	pc := newResumableProxyConnection(func(ctx context.Context, req DialRequest) (*websocket.Conn, error) {
		conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws"+server.URL[4:], nil)
		if resp != nil {
			resp.Body.Close()
		}
		return conn, err
	})
	done := make(chan error, 1)
	go func() { done <- pc.dialReattach(ctx) }()
	<-ready
	cancel()
	select {
	case err := <-done:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("cancellation did not interrupt the handshake")
	}
}

func TestResumeReplaysFullWindowsInBothDirections(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Second)
	defer cancel()
	serverProxy := newResumableProxyConnection(nil)
	var accepted atomic.Bool
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if accepted.CompareAndSwap(false, true) {
			assert.NoError(t, serverProxy.accept(w, r))
			return
		}
		req, err := parseDialRequest(r)
		if assert.NoError(t, err) {
			if req.Reattach {
				_ = serverProxy.acceptReattach(ctx, w, r, req.Delivered)
			} else {
				_ = serverProxy.acceptHandover(ctx, w, r)
			}
		}
	}))
	server.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateHijacked {
			assert.NoError(t, conn.(*net.TCPConn).SetWriteBuffer(4096))
		}
	}
	server.Start()
	defer server.Close()
	defer cancel()
	sessionCtx := ctx
	clientProxy := newResumableProxyConnection(func(ctx context.Context, req DialRequest) (*websocket.Conn, error) {
		url := fmt.Sprintf("ws%s?id=test&resume_version=2&delivered=%d", server.URL[4:], req.Delivered)
		if req.Reattach {
			url += "&reattach=1"
		}
		dialer := websocket.Dialer{NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := (&net.Dialer{}).DialContext(ctx, network, addr)
			if err == nil {
				err = conn.(*net.TCPConn).SetWriteBuffer(4096)
			}
			return conn, err
		}}
		conn, resp, err := dialer.DialContext(ctx, url, nil)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil {
			context.AfterFunc(sessionCtx, func() { conn.Close() })
		}
		return conn, err
	})
	require.NoError(t, clientProxy.connect(ctx))
	<-serverProxy.ready
	defer func() { _ = serverProxy.closeConnection() }()
	defer func() { _ = clientProxy.closeConnection() }()

	// Both writes failed before reaching the peer. Neither direction has room for
	// new payload, and neither can finish replay unless the peer resumes reading.
	toClient := bytes.Repeat([]byte("s"), proxyResumeBufferLimit)
	toServer := bytes.Repeat([]byte("c"), proxyResumeBufferLimit)
	require.NoError(t, serverProxy.resume.sendBuf.append(toClient))
	require.NoError(t, clientProxy.resume.sendBuf.append(toServer))
	require.NoError(t, clientProxy.closeConnection())

	clientInput, clientWriter := io.Pipe()
	serverInput, serverWriter := io.Pipe()
	defer clientWriter.Close()
	defer serverWriter.Close()
	clientOutput := newTestBuffer(t)
	serverOutput := newTestBuffer(t)
	clientOutput.OnWrite = make(chan []byte, 4096)
	serverOutput.OnWrite = make(chan []byte, 4096)
	done := make(chan error, 2)
	go func() { done <- serverProxy.start(ctx, serverInput, serverOutput) }()
	go func() { done <- clientProxy.start(ctx, clientInput, clientOutput) }()
	require.Eventually(t, func() bool {
		return clientOutput.Contains(toClient) && serverOutput.Contains(toServer)
	}, 2*time.Second, time.Millisecond, "both peers must read while replaying")

	// Resume must remain available after a transfer larger than the replay window,
	// including an auth handover and another reset while data is in flight.
	moreClient := bytes.Repeat([]byte("download"), proxyResumeBufferLimit)
	moreServer := bytes.Repeat([]byte("upload!!"), proxyResumeBufferLimit)
	writes := make(chan error, 2)
	go func() { _, err := serverWriter.Write(moreClient); writes <- err }()
	go func() { _, err := clientWriter.Write(moreServer); writes <- err }()
	require.NoError(t, clientProxy.initiateHandover(ctx))
	require.NoError(t, clientProxy.conn.Load().UnderlyingConn().Close())
	for range 2 {
		select {
		case err := <-writes:
			require.NoError(t, err)
		case <-ctx.Done():
			t.Fatal("transfer stalled after reattach")
		}
	}
	toClient = append(toClient, moreClient...)
	toServer = append(toServer, moreServer...)
	require.Eventually(t, func() bool {
		return clientOutput.Contains(toClient) && serverOutput.Contains(toServer)
	}, 2*time.Second, time.Millisecond, "large transfers must remain byte-exact after reconnecting")
	cancel()
	for range 2 {
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("proxy did not stop")
		}
	}
	assert.Len(t, clientOutput.String(), len(toClient))
	assert.Len(t, serverOutput.String(), len(toServer))
}

func TestSendBufferReplaysWhatThePeerMissed(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("hello ")))
	require.NoError(t, b.append([]byte("world")))

	// The peer wrote only the first 6 bytes before the connection broke.
	missing, err := b.replayFrom(6)
	require.NoError(t, err)
	assert.Equal(t, "world", string(missing))
}

func TestSendBufferReplayFromCurrentOffsetIsEmpty(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("hello")))

	missing, err := b.replayFrom(5)
	require.NoError(t, err)
	assert.Empty(t, missing)
}

func TestSendBufferAckDiscardsThePrefix(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("aaaabbbb")))
	b.ack(4)

	// Everything from the acknowledged offset is still replayable.
	missing, err := b.replayFrom(4)
	require.NoError(t, err)
	assert.Equal(t, "bbbb", string(missing))

	// The discarded prefix is not.
	_, err = b.replayFrom(3)
	assert.ErrorIs(t, err, errReplayUnavailable)
}

func TestSendBufferAckFreesTheWindow(t *testing.T) {
	b := newSendBuffer(8)
	require.NoError(t, b.append([]byte("12345678")))
	require.ErrorIs(t, b.append([]byte("9")), errSendWindowExhausted)

	b.ack(8)
	assert.NoError(t, b.append([]byte("9")))
}

func TestSendBufferIgnoresStaleAndImpossibleAcks(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abcdef")))
	b.ack(4)

	// A stale ack must not rewind the buffer, and an ack beyond what we sent must not
	// discard bytes the peer cannot have received.
	b.ack(2)
	b.ack(99)

	missing, err := b.replayFrom(4)
	require.NoError(t, err)
	assert.Equal(t, "ef", string(missing))
}

func TestSendBufferRejectsAnImpossibleReplayOffset(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abc")))

	_, err := b.replayFrom(4)
	require.Error(t, err)
	assert.NotErrorIs(t, err, errReplayUnavailable)
}

func TestSendBufferReplayIsACopy(t *testing.T) {
	b := newSendBuffer(64)
	require.NoError(t, b.append([]byte("abcdef")))

	missing, err := b.replayFrom(0)
	require.NoError(t, err)
	missing[0] = 'z'

	again, err := b.replayFrom(0)
	require.NoError(t, err)
	assert.Equal(t, "abcdef", string(again), "mutating a replay must not corrupt the buffer")
}
