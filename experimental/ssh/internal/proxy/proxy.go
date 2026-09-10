package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// Sentinels for how a proxy session ended, so callers can attribute it for telemetry without
// matching on error text. Joined onto the error at the site that detected it.
var (
	// ErrConnectFailed marks a failure to establish the initial proxy websocket.
	ErrConnectFailed = errors.New("proxy websocket could not be established")
	// ErrWebsocketDropped marks an established proxy websocket that stopped carrying traffic.
	ErrWebsocketDropped = errors.New("proxy websocket dropped")
	// ErrHandoverFailed marks a handover that ended the session. A handover that only failed
	// to dial its replacement does not end the session and is not reported with this.
	ErrHandoverFailed = errors.New("proxy handover failed")
)

var (
	errProxyEOF             = errors.New("proxy EOF error")
	errSendingLoopStopped   = errors.New("sending loop stopped")
	errReceivingLoopStopped = errors.New("receiving loop stopped")
	// Marks a handover that failed while opening its replacement connection, before any
	// connection state changed. The current connection is still the one both proxy loops
	// use, so the session can carry on with it instead of ending.
	errHandoverDialFailed = errors.New("handover dial failed")
	// Marks a write that failed on a resumable connection. The payload is already buffered for
	// replay, so the sending loop treats it as a pause rather than the end of the session.
	errSendFailedResumable = errors.New("send failed on a resumable connection")
)

// proxyResumeGrace is how long the server holds a dropped session - sshd, its client slot and its
// buffered output - waiting for the client to come back. Longer than proxyResumeBudget so the
// server never reaps a client that is still trying. It is a var so tests can shorten it.
var proxyResumeGrace = 90 * time.Second

const (
	// Same as gorilla/websocket default read/write buffer sizes. Bigger payloads will be split into multiple ws frames.
	proxyBufferSize = 4 * 1024
	// Timeout for the full handover process, when initiated by the client.
	proxyHandoverInitTimeout = 30 * time.Second
	// Timeout for the handover process, when accepted by the server.
	proxyHandoverAcceptTimeout = 25 * time.Second
	// Bounds how long a keepalive ping may hold the websocket's write lock. A stalled or half-open
	// connection parks a write until the kernel gives up retransmitting (~15 minutes with Linux
	// defaults), and close() and the sending loop need that same lock, so the ping caps its wait.
	proxyPingWriteTimeout = 5 * time.Second

	// How long the client keeps trying to reattach to its session after the connection drops.
	// It has to stay clear of ssh's own ceiling: ServerAliveInterval 30 (see
	// sshconfig.ServerAliveIntervalSeconds) times OpenSSH's default ServerAliveCountMax of 3
	// means ssh gives up on an unresponsive tunnel after about 90 seconds, and a resume that
	// outlasts that repairs a session ssh has already abandoned.
	proxyResumeBudget = 60 * time.Second
	// Backoff between resume dials. The first attempt is immediate: a reset often clears at once.
	proxyResumeRetryBackoff = 2 * time.Second
	// Bounds the wait for the peer's first frame on a reattached connection, which carries the
	// offset to replay from. The connection is new, but the peer may be wedged.
	proxyResumeHandshakeTimeout = 10 * time.Second
	// Cap on payload held for replay, per direction. The unacknowledged window is bounded by
	// the TCP send buffer plus whatever the driver proxy holds, and acks land every
	// proxyAckThreshold bytes, so this is never approached in practice. Reaching it means the
	// peer stopped acknowledging, i.e. the connection is already beyond saving.
	proxyResumeBufferLimit = 1 << 20
	// How much payload may be delivered before we tell the peer about it, so it can release
	// its replay buffer. Small enough to keep the window far below proxyResumeBufferLimit.
	proxyAckThreshold = 64 << 10
)

// resumeState is the per-connection bookkeeping a resumable transport needs. It is nil unless
// both ends negotiated resume, in which case the proxy behaves exactly as it did before.
type resumeState struct {
	// Outgoing payload that may still have to be replayed. Not used after degradation but retained
	// to avoid race conditions where concurrent goroutines may check resumable() and then access the buffer.
	sendBuf *sendBuffer
	// Total payload bytes written to the destination. The peer replays from this offset, so it
	// only advances after a successful write.
	delivered atomic.Int64
	// The delivered count we last told the peer about, so an ack is only sent once the number
	// has actually moved.
	acked atomic.Int64
	// Carries the replacement connection to a parked receiving loop. Only the server uses it:
	// it cannot dial, so it waits here for the client's inbound reattach request. Buffered so a
	// client that reattaches before this side has noticed the drop is picked up rather than missed.
	resumed chan *websocket.Conn
	// Signalled by the receiving loop once it has stopped delivering, so a reattach can report a
	// delivered count that cannot move under it. Server side only, and buffered for the same
	// reason as resumed.
	parked chan struct{}
	// Throttles the sending loop while a reattach is in progress.
	gate sendGate
	// Set to true when the replay buffer fills, degrading the connection to non-resumable
	// so a healthy session survives. Further reattach attempts stop advertising resumability.
	degraded atomic.Bool
}

func newResumeState(bufferLimit int) *resumeState {
	return &resumeState{
		sendBuf: newSendBuffer(bufferLimit),
		resumed: make(chan *websocket.Conn, 1),
		parked:  make(chan struct{}, 1),
	}
}

// handoverCoordination holds the context and channels used to coordinate a single handover operation
// between the receiving loop and the handover initiator (initiateHandover or acceptHandover).
type handoverCoordination struct {
	// Context with timeout for the entire handover operation.
	// Shared between the handover initiator and the receiving loop.
	ctx context.Context
	// Used by the receiving loop to signal about the closure of the current connection to the handover initiator.
	// After signalling, the receiving loop will block until connSwapped channel is signaled.
	connClosed chan error
	// Used by the handover initiator to signal the receiving loop that it's safe to start reading from the new connection.
	connSwapped chan struct{}
}

func (c *handoverCoordination) signalConnectionClosed(err error) error {
	select {
	case c.connClosed <- err:
		return err
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *handoverCoordination) waitForConnectionToClose() error {
	select {
	case err := <-c.connClosed:
		return err
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *handoverCoordination) signalConnectionSwapped() error {
	select {
	case c.connSwapped <- struct{}{}:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *handoverCoordination) waitForConnectionToSwap() error {
	select {
	case <-c.connSwapped:
		return nil
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

// proxyConnection is the main struct that manages the websocket connection and the handover process.
// It works both on the client and the server side (see internal/client and internal/server packages).
// It has 3 goroutines:
// - Sending loop: reads from src and sends to the current connection.
// - Receiving loop: reads from the current connection and writes to dst.
// - Main: starts the other two (start method) and initiates or accepts handover (initiateHandover or acceptHandover).
type proxyConnection struct {
	// Each connection has a unique ID.
	connID string
	// Function to create a new websocket connection. Tests can override this to use a test websocket connection.
	createWebsocketConnection createWebsocketConnectionFunc
	// Atomic that keeps the currently active connection.
	// Can be swapped during handover.
	conn atomic.Pointer[websocket.Conn]
	// Prevents multiple handover processes from running concurrently.
	// Blocks proxying any outgoing messages during the entire handover in the sending loop.
	handoverMutex sync.Mutex
	// Atomic that holds the current handover coordination channels, or nil if no handover is in progress.
	handoverState atomic.Pointer[handoverCoordination]
	// Channel that is closed when the initial connection is established (or failed).
	// Prevents race conditions where handover is accepted before the initial connection is ready.
	ready chan struct{}
	// Byte accounting for reattaching to this session after an unexpected disconnect, or nil
	// when resume was not negotiated. Immutable after construction.
	resume *resumeState
}

// DialRequest describes the connection a client is asking the server for.
type DialRequest struct {
	// Identifies the session. A server that already has a connection under this ID treats the
	// dial as a handover or a reattach rather than a new session.
	ConnID string
	// Whether this client speaks the resume protocol. When set, the dial carries the delivered
	// offset below, and that parameter's presence is how the server learns it must buffer its
	// own output for replay too.
	ResumeCapable bool
	// How many payload bytes this side has written to its destination. Sent on every dial, not
	// just a reattach, so the offset is always current when a drop does happen.
	Delivered int64
	// Asks the server to reattach this connection to an existing session whose previous
	// connection dropped, and to replay what was lost. Stated explicitly rather than inferred
	// from the server's own view of the connection, because the client often notices the drop
	// first and would otherwise race the server into treating a reattach as a handover.
	Reattach bool
}

type createWebsocketConnectionFunc func(ctx context.Context, req DialRequest) (*websocket.Conn, error)

func newProxyConnection(createConn createWebsocketConnectionFunc) *proxyConnection {
	return &proxyConnection{
		connID:                    uuid.NewString(),
		createWebsocketConnection: createConn,
		ready:                     make(chan struct{}),
	}
}

// newResumableProxyConnection is newProxyConnection with the byte accounting that lets the
// session survive an unexpected disconnect. Both ends must agree: a server that does not speak
// the protocol tears the session down on the first dropped connection regardless, and a client
// must not attempt a resume against one (it would replay into a freshly spawned sshd).
// bufferLimit sets the per-direction replay buffer cap; use proxyResumeBufferLimit for production.
func newResumableProxyConnection(createConn createWebsocketConnectionFunc, bufferLimit int) *proxyConnection {
	pc := newProxyConnection(createConn)
	pc.resume = newResumeState(bufferLimit)
	return pc
}

// resumable reports whether this connection can reattach to its session after a drop.
// Once the replay buffer fills, the connection is degraded to non-resumable and this returns false.
//
// Degrade-during-reattach safety: degraded is set inside sendMessage while holding handoverMutex.
// resumable checks whether the connection is still resumable. Race-safe: degraded is atomic.Bool
// and write-once (set true, never cleared), so concurrent reads from multiple goroutines
// (sendMessage, runReceivingLoop, runSendingLoop) are safe.
//
// On the client side, dialReattach holds handoverMutex for its entire lifetime, serializing
// degradation with any in-flight reattach. On the server side, acceptReattach holds handoverMutex
// around replayTo, and sendGate blocks the sending loop before sendMessage can degrade, so
// degradation cannot interleave with replay.
// Once degraded is true, resumable() returns false, preventing:
// - runReceivingLoop from attempting reattaches (the if pc.resumable() check stops it)
// - replayTo from being called by a new reattach, so sendBuf won't be accessed
// This prevents silent data corruption from an inconsistent replay offset.
func (pc *proxyConnection) resumable() bool {
	if pc.resume == nil {
		return false
	}
	return !pc.resume.degraded.Load()
}

func (pc *proxyConnection) start(ctx context.Context, src io.ReadCloser, dst io.Writer) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		err := pc.runSendingLoop(gCtx, src)
		// Always return a non nil error to cancel the errgroup context
		return errors.Join(err, errSendingLoopStopped)
	})
	g.Go(func() error {
		err := pc.runReceivingLoop(gCtx, dst)
		// Always return a non nil error to cancel the errgroup context
		return errors.Join(err, errReceivingLoopStopped)
	})
	g.Go(func() error {
		// Wait for the context to be cancelled. There can be multiple reasons:
		// - Sending loop finished (e.g. EOF from source)
		// - Receiving loop finished (e.g. connection closed)
		// - Parent context cancelled
		// Both loops can still be stuck on conn.ReadMessage or src.Read and won't notice context cancellation,
		// so we close the connection and the source (sshd stdout pipe or ssh client stdio) to unblock them.
		<-gCtx.Done()
		return errors.Join(pc.close(gCtx), pc.closeConnection(), pc.closeSource(src))
	})
	err := g.Wait()
	if err == nil || isNormalClosure(err) {
		return nil
	}
	return err
}

func (pc *proxyConnection) connect(ctx context.Context) error {
	defer close(pc.ready)
	// Nothing has been delivered yet, so the initial dial reports offset zero. Sending it at all
	// is what tells a resume-capable server that this client speaks the protocol.
	conn, err := pc.createWebsocketConnection(ctx, DialRequest{ConnID: pc.connID, ResumeCapable: pc.resumable()})
	if err != nil {
		return err
	}
	pc.conn.Store(conn)
	return nil
}

func (pc *proxyConnection) accept(w http.ResponseWriter, r *http.Request) error {
	defer close(pc.ready)
	conn, err := pc.acceptWebsocketConnection(w, r)
	if err != nil {
		return err
	}
	pc.conn.Store(conn)
	return nil
}

func (pc *proxyConnection) acceptWebsocketConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	upgrader := websocket.Upgrader{}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade to websockets: %w", err)
	}
	return conn, nil
}

func (pc *proxyConnection) runSendingLoop(ctx context.Context, src io.Reader) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		b := make([]byte, proxyBufferSize)
		n, readErr := src.Read(b)
		if n > 0 {
			// Wait out any reattach in progress, so a connection that is down does not fill the
			// whole replay window before it comes back. src stays blocked on the OS side
			// meanwhile, which is the backpressure we want.
			if pc.resumable() {
				if err := pc.resume.gate.wait(ctx); err != nil {
					return err
				}
			}
			// This will block during handover - we stop sending anything except the close message.
			// Meanwhile the "src" (sshd server stdout or ssh client stdin) will be buffered/blocked on the OS side until we start reading from it again.
			err := pc.sendMessage(ctx, websocket.BinaryMessage, b[:n])
			switch {
			case errors.Is(err, errSendFailedResumable):
				// Buffered for replay, and sendMessage has closed the connection so the
				// receiving loop starts the reattach. Carry on reading src: its bytes accumulate
				// in the replay buffer, and the gate above holds the next write until the
				// connection is back. Falls through to readErr rather than continuing the loop,
				// so a read that returned data together with an error still reports it.
				log.Debugf(ctx, "Send failed on a resumable connection, waiting for the reattach: %v", err)
			case err != nil:
				return errors.Join(ErrWebsocketDropped, fmt.Errorf("failed to send message: %w", err))
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return errors.Join(errProxyEOF, readErr)
			} else {
				return fmt.Errorf("failed to read from source: %w", readErr)
			}
		}
	}
}

func (pc *proxyConnection) sendMessage(ctx context.Context, mt int, data []byte) error {
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()
	// Record the payload before writing it, and under the same lock a resume swaps the
	// connection with: that way the buffer always holds every byte the peer may still be
	// missing, and a resume can never replay a range the sending loop is still appending to.
	if pc.resumable() && mt == websocket.BinaryMessage {
		if err := pc.resume.sendBuf.append(data); err != nil {
			// The replay buffer has filled. This is not a dead peer - the window is reached by
			// ordinary in-flight data when a continuous burst saturates the transport. Degrade to
			// non-resumable to keep the session alive: a subsequent real drop will end it.
			if errors.Is(err, errSendWindowExhausted) {
				pc.resume.degraded.Store(true)
				log.Warnf(ctx, "SSH tunnel replay buffer filled: downgrading to non-resumable to keep the session alive")
				// Buffer retained but unused after degradation to avoid race conditions. Continue with
				// the write below instead of returning an error. Once resumable() checks degraded and
				// returns false, no more appends or acks will occur, so the ~1 MiB of retained memory
				// is acceptable for the session's lifetime.
			} else {
				return err
			}
		}
	}
	conn := pc.conn.Load()
	err := conn.WriteMessage(mt, data)
	if err != nil && pc.resumable() {
		// This failure costs no data whatever the message type: a binary payload was buffered
		// above, and control/ack messages are regenerated after the resume. gorilla latches a
		// permanent write error after any failed write, so this connection can never send again -
		// close it to fail the receiving loop's read now and drive the reattach, rather than let
		// the sending loop fill the whole window first. This must cover a failed ack (a text
		// control frame) too, not only a binary payload: when traffic is one-way from the server
		// the receiving side never writes a binary frame, so a poisoned connection would otherwise
		// only ever surface as a failed ack. Left as a bare log, the read loop kept running while
		// the peer stopped getting acks, its replay buffer filled to the limit, and the session
		// ended instead of reattaching. A failed close message reaches here only during teardown,
		// where closing the connection is what happens next anyway.
		conn.Close()
		return errors.Join(errSendFailedResumable, err)
	}
	return err
}

// sendControlMessage tells the peer how much payload we have written to our destination, so it
// can release that much of its replay buffer.
func (pc *proxyConnection) sendControlMessage(ctx context.Context, delivered int64) error {
	payload, err := json.Marshal(controlMessage{Delivered: delivered})
	if err != nil {
		return err
	}
	return pc.sendMessage(ctx, websocket.TextMessage, payload)
}

// ackDelivered reports our delivered count to the peer once it has moved far enough to be worth
// a frame. A failure is not fatal: the ack is only an optimisation that keeps the peer's replay
// buffer small, and a genuinely broken connection is reported by the loops themselves.
func (pc *proxyConnection) ackDelivered(ctx context.Context) {
	delivered := pc.resume.delivered.Load()
	if delivered-pc.resume.acked.Load() < proxyAckThreshold {
		return
	}
	if err := pc.sendControlMessage(ctx, delivered); err != nil {
		log.Debugf(ctx, "Failed to acknowledge %d delivered bytes: %v", delivered, err)
		return
	}
	pc.resume.acked.Store(delivered)
}

// sendPing writes a keepalive ping on the current connection. Unlike sendMessage it takes neither
// the handover mutex nor an unbounded wait: gorilla permits WriteControl concurrently with the data
// writes, and its deadline bounds how long a stalled socket holds the connection's write lock, which
// close() and the sending loop also need.
func (pc *proxyConnection) sendPing() error {
	conn := pc.conn.Load()
	return conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(proxyPingWriteTimeout))
}

func (pc *proxyConnection) runReceivingLoop(ctx context.Context, dst io.Writer) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn := pc.conn.Load()
		mt, data, err := conn.ReadMessage()
		if err != nil {
			// During handover a normal closure is expected, but any other error must stop the read loop (and eventually terminate the ssh session).
			if handover := pc.handoverState.Load(); handover != nil {
				var closeConnSignal error
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					closeConnSignal = errors.Join(ErrWebsocketDropped, fmt.Errorf("failed to read from websocket during handover: %w", err))
				}
				// Signal the current connection is closed to the handover initiator (initiateHandover or acceptHandover).
				if err := handover.signalConnectionClosed(closeConnSignal); err != nil {
					return err
				}
				// Wait for the handover initiator to swap the connection.
				// While we wait for the handover to complete, the new connection might be getting incoming messages.
				// They will be buffered by the TCP stack and will be read by us after the handover is complete.
				if err := handover.waitForConnectionToSwap(); err != nil {
					return err
				}
				// Continue with the receiving loop, pc.conn is now the new connection.
				continue
			} else {
				if errors.Is(err, io.EOF) || websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					return errors.Join(errProxyEOF, err)
				}
				// A read that fails once our own context is cancelled is the teardown, not a drop:
				// start's context watcher closes the connection to unblock this very read, and
				// only after the context is done, so cancellation is always visible here first.
				// Neither branch below fits - a reattach would warn the user about a drop on every
				// clean exit and could not succeed anyway (its redial budget comes from this same
				// context), and ErrWebsocketDropped would bill an ordinary exit to a tunnel failure.
				if ctx.Err() != nil {
					return ctx.Err()
				}
				// An unexpected drop. With resume negotiated the session state on both ends
				// outlives the connection, so reattach instead of ending the session.
				if pc.resumable() {
					if resumeErr := pc.reattach(ctx); resumeErr != nil {
						return errors.Join(ErrWebsocketDropped, fmt.Errorf("failed to reattach after the connection dropped: %w", resumeErr))
					}
					continue
				}
				return errors.Join(ErrWebsocketDropped, fmt.Errorf("failed to read from websocket: %w", err))
			}
		}

		if mt == websocket.TextMessage && pc.resumable() {
			var msg controlMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				return fmt.Errorf("failed to decode control message: %w", err)
			}
			pc.resume.sendBuf.ack(msg.Delivered)
			continue
		}
		if mt != websocket.BinaryMessage {
			return errors.New("received non-binary websocket message")
		}
		if _, err := dst.Write(data); err != nil {
			return fmt.Errorf("failed to copy to writer: %w", err)
		}
		if pc.resumable() {
			// Only count what actually reached the destination: this is the offset the peer
			// replays from, so counting an unwritten byte would silently lose it.
			pc.resume.delivered.Add(int64(len(data)))
			pc.ackDelivered(ctx)
		}
	}
}

func (pc *proxyConnection) close(ctx context.Context) error {
	// Keep in mind that pc.sendMessage blocks during handover
	err := pc.sendMessage(ctx, websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		if isNormalClosure(err) || errors.Is(err, websocket.ErrCloseSent) {
			return nil
		} else {
			return fmt.Errorf("failed to send close message: %w", err)
		}
	}
	return nil
}

// closeConnection closes the underlying websocket. The close message pc.close sends only ends the
// session if the peer is still there to react to it by closing the connection, and it does not even
// go out once a failed write has put the connection into gorilla's permanent write-error state (one
// timed-out keepalive ping is enough). Without this the receiving loop stays blocked in ReadMessage
// and the session hangs instead of exiting.
func (pc *proxyConnection) closeConnection() error {
	err := pc.conn.Load().Close()
	if errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}

func (pc *proxyConnection) closeSource(src io.ReadCloser) error {
	err := src.Close()
	if err != nil && (errors.Is(err, os.ErrClosed) || errors.Is(err, io.ErrClosedPipe)) {
		return nil
	}
	return err
}

func (pc *proxyConnection) initiateHandover(ctx context.Context) error {
	// Blocks proxying any outgoing messages during the entire handover
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	handoverCtx, cancel := context.WithTimeout(ctx, proxyHandoverInitTimeout)
	defer cancel()
	handoverState := &handoverCoordination{
		ctx:         handoverCtx,
		connClosed:  make(chan error),
		connSwapped: make(chan struct{}),
	}
	// Existence of the handoverState indicates to the receiving loop that we are in the middle of a handover process,
	// and should treat close messages as a signal to finish the handover instead of erroring out.
	pc.handoverState.Store(handoverState)
	defer pc.handoverState.Store(nil)

	// Create a new websocket connection by sending an /ssh?id=<connID> request to the server.
	// When server realises it's an ID of an existing connection, it will start AcceptHandover process.
	newConn, err := pc.createWebsocketConnection(handoverCtx, DialRequest{
		ConnID:        pc.connID,
		ResumeCapable: pc.resumable(),
		// A handover replaces a connection that still works, so the close-frame barrier keeps
		// the byte stream intact and nothing needs replaying. The offset still travels, so the
		// server keeps buffering for the drop that may come later.
		Delivered: pc.deliveredCount(),
	})
	if err != nil {
		// Nothing has been swapped yet: pc.conn is still live and the receiving loop is still
		// reading it. Tag the error so the caller can keep the session on it - see
		// errHandoverDialFailed. Retrying the dial here instead would be unsafe: a dial can
		// fail after the server already accepted it and began its side of the handover, and a
		// second dial would then race the first one's acceptHandover for the same connection.
		return errors.Join(errHandoverDialFailed, fmt.Errorf("failed to create new websocket connection: %w", err))
	}

	// Wait for the server to close the old connection
	// (it does so when it receives an /ssh request with known connection ID and starts AcceptHandover process).
	// Receiving loop will signal about closed connection to the coord.connClosed channel.
	if err := handoverState.waitForConnectionToClose(); err != nil {
		newConn.Close()
		return err
	}

	pc.conn.Store(newConn)

	// Let the receiving loop know that the current connection is swapped and it's safe to start reading from it.
	if err := handoverState.signalConnectionSwapped(); err != nil {
		newConn.Close()
		return err
	}
	return nil
}

func (pc *proxyConnection) acceptHandover(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Blocks proxying any outgoing messages during the entire handover
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	// Wait for the initial connection to be ready
	select {
	case <-pc.ready:
	case <-ctx.Done():
		return ctx.Err()
	}

	handoverCtx, cancel := context.WithTimeout(ctx, proxyHandoverAcceptTimeout)
	defer cancel()
	handoverState := &handoverCoordination{
		ctx:         handoverCtx,
		connClosed:  make(chan error),
		connSwapped: make(chan struct{}),
	}
	// Existence of the handoverState indicates to the receiving loop that we are in the middle of a handover process,
	// and should treat close messages as a signal to finish the handover instead of erroring out.
	pc.handoverState.Store(handoverState)
	defer pc.handoverState.Store(nil)

	newConn, err := pc.acceptWebsocketConnection(w, r)
	if err != nil {
		return fmt.Errorf("failed to accept new websocket connection: %w", err)
	}

	// Signal the client to complete handover by closing the old connection.
	// Not using pc.sendMessage here, because it's blocked by the handover mutex.
	currentConn := pc.conn.Load()
	if currentConn == nil {
		newConn.Close()
		return errors.New("initial connection not established")
	}
	err = currentConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "handover"))
	if err != nil {
		newConn.Close()
		return fmt.Errorf("failed to send close message to the current connection: %w", err)
	}

	// Wait for the client to acknowledge the closure of the old connection.
	// On the client its done automatically by the websocket library with the default close handler.
	// On the server we then receive a close error in the RunReceivingLoop and signal about it to the coord.connClosed channel.
	if err := handoverState.waitForConnectionToClose(); err != nil {
		newConn.Close()
		return err
	}

	pc.conn.Store(newConn)

	// Let the receiving loop know that the current connection is swapped and it's safe to start reading from it.
	if err := handoverState.signalConnectionSwapped(); err != nil {
		newConn.Close()
		return err
	}

	return nil
}

func isNormalClosure(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure) || errors.Is(err, errProxyEOF)
}
