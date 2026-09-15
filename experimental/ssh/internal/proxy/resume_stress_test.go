//go:build !windows

package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sink collects everything the proxy writes to its destination. Unlike testBuffer it has no
// bounded channel, which a multi-megabyte transfer would fill and then block the receiving loop
// on - the stall would be the test's, not the proxy's.
type sink struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *sink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *sink) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Len()
}

func (s *sink) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.buf.Bytes()...)
}

// runResumableClientTo runs RunClientProxy against url with resume negotiated, writing what it
// receives to dst. It returns the writer standing in for ssh's stdin and the session's outcome.
func runResumableClientTo(t *testing.T, url string, dst io.Writer) (io.WriteCloser, <-chan error) {
	ctx := cmdio.MockDiscard(t.Context())
	wsURL := "ws" + url[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		u := fmt.Sprintf("%s?id=%s&resume_version=%d&delivered=%d", wsURL, dial.ConnID, ResumeProtocolVersion, dial.Delivered)
		if dial.Reattach {
			u += "&reattach=1"
		}
		conn, resp, err := websocket.DefaultDialer.DialContext(ctx, u, nil) // nolint:bodyclose
		if resp != nil {
			resp.Body.Close()
		}
		// Mirrors what createWebsocketConnection does in internal/client: a 4xx answer to a
		// reattach is a permanent refusal, so the redial loop stops instead of spending its whole
		// budget. Without it a test that refuses a reattach waits out proxyResumeBudget.
		if err != nil && dial.Reattach && resp != nil && resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return nil, errors.Join(ErrReattachRejected, err)
		}
		return conn, err
	}
	src, srcWriter := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, dst, neverTick, time.Hour, true, createConn)
	}()
	return srcWriter, done
}

// runPlainClientTo is runResumableClientTo without resume, the path an older server gets.
func runPlainClientTo(t *testing.T, url string, dst io.Writer) (io.WriteCloser, <-chan error) {
	ctx := cmdio.MockDiscard(t.Context())
	wsURL := "ws" + url[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL+"?id="+dial.ConnID, nil) // nolint:bodyclose
		return conn, err
	}
	src, srcWriter := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, dst, neverTick, time.Hour, false, createConn)
	}()
	return srcWriter, done
}

// firstDifference reports where two byte streams diverge, so a failure names an offset instead of
// dumping megabytes. Returns -1 when want is a prefix of got.
func firstDifference(got, want []byte) int {
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			return i
		}
	}
	if len(got) < len(want) {
		return len(got)
	}
	return -1
}

// context reports the bytes around an offset, for a failure message.
func context16(b []byte, at int) string {
	return string(b[max(at-16, 0):min(at+16, len(b))])
}

// resetClientLegs is resetClients without its assertions: a handover closes connections behind the
// relay's back, so a leg that is already gone cannot be reset and is not a failure.
func (r *tcpRelay) resetClientLegs() {
	r.mu.Lock()
	conns := r.clientConns
	r.clientConns = nil
	r.mu.Unlock()
	for _, conn := range conns {
		if conn.SetLinger(0) == nil {
			conn.Close()
		}
	}
}

// One reset at a quiet moment is the easy case. Customers report drops several times a session, on
// networks that flap, so resets have to be survivable wherever they land - including while a replay
// is still being written. A single lost or duplicated byte fails SSH's MAC (RFC 4253 section 6) and
// disconnects the session the resume exists to save, so the whole stream is compared, not sampled.
func TestResumeStaysByteExactUnderAResetStorm(t *testing.T) {
	server := createTestServer(t, 4, time.Hour)
	defer server.Close()
	relay := newTCPRelay(t, server.Listener.Addr().String())

	out := &sink{}
	srcWriter, done := runResumableClientTo(t, relay.URL(), out)

	const chunks = 512
	const chunkSize = 8 * 1024
	var sent []byte
	for i := range chunks {
		sent = append(sent, bytes.Repeat([]byte{byte('A' + i%26)}, chunkSize)...)
	}

	storm := make(chan struct{})
	stormDone := make(chan struct{})
	var resets atomic.Int64
	go func() {
		defer close(stormDone)
		for {
			select {
			case <-storm:
				return
			case <-time.After(7 * time.Millisecond):
				relay.resetClientLegs()
				resets.Add(1)
			}
		}
	}()
	stop := func() {
		close(storm)
		<-stormDone
	}

	writeDone := make(chan error, 1)
	go func() {
		// Paced, so the resets land inside the transfer rather than after it. A terminal or an scp
		// stream is not delivered as one instantaneous burst either.
		for i := range chunks {
			if _, err := srcWriter.Write(sent[i*chunkSize : (i+1)*chunkSize]); err != nil {
				writeDone <- err
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		writeDone <- nil
	}()

	deadline := time.After(60 * time.Second)
	for out.Len() < len(sent) {
		select {
		case err := <-done:
			stop()
			t.Fatalf("session ended after %d resets with %d/%d bytes echoed: %v", resets.Load(), out.Len(), len(sent), err)
		case <-deadline:
			stop()
			t.Fatalf("transfer stalled after %d resets with %d/%d bytes echoed", resets.Load(), out.Len(), len(sent))
		case <-time.After(20 * time.Millisecond):
		}
	}
	stop()
	require.NoError(t, <-writeDone)

	got := out.Bytes()
	if diff := firstDifference(got, sent); diff >= 0 {
		t.Fatalf("the echoed stream diverges from what was sent at offset %d (after %d resets): got %q, want %q",
			diff, resets.Load(), context16(got, diff), context16(sent, diff))
	}
	assert.Len(t, got, len(sent), "the echoed stream must not carry extra bytes")
}

// Every reset spawns a reattach, and a session on a flapping network gets hundreds of them.
// Goroutines that outlive their reattach would accumulate for the life of the session.
func TestRepeatedResetsDoNotLeakGoroutines(t *testing.T) {
	server := createTestServer(t, 4, time.Hour)
	defer server.Close()
	relay := newTCPRelay(t, server.Listener.Addr().String())

	out := &sink{}
	srcWriter, done := runResumableClientTo(t, relay.URL(), out)

	// Settle the session first, so the baseline covers only what the resets add.
	_, err := srcWriter.Write([]byte("warmup\n"))
	require.NoError(t, err)
	require.Eventually(t, func() bool { return out.Len() >= len("warmup\n") }, 10*time.Second, 10*time.Millisecond)
	time.Sleep(500 * time.Millisecond)
	baseline := runtime.NumGoroutine()

	const rounds = 30
	for i := range rounds {
		relay.resetClientLegs()
		line := []byte("after reset\n")
		want := out.Len() + len(line)
		_, err := srcWriter.Write(line)
		require.NoError(t, err)
		require.Eventually(t, func() bool { return out.Len() >= want }, 20*time.Second, 10*time.Millisecond,
			"round %d did not survive its reset", i)
	}

	select {
	case err := <-done:
		t.Fatalf("session ended during the resets: %v", err)
	default:
	}

	// Reattach goroutines retire asynchronously, so give them a moment.
	time.Sleep(time.Second)
	assert.Less(t, runtime.NumGoroutine()-baseline, rounds,
		"goroutine count grew from %d to %d across %d resets, which is close to one per reset",
		baseline, runtime.NumGoroutine(), rounds)
}
