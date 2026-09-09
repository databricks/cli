package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/gorilla/websocket"
)

// errSendWindowExhausted means the peer has not acknowledged anything for a full buffer's worth
// of payload. The connection cannot be resumed past this point, because the bytes the peer would
// need replayed are the ones we would have to discard to make room.
var errSendWindowExhausted = errors.New("resume send buffer is full: the peer stopped acknowledging")

// errReplayUnavailable means the peer asked to resume from an offset we no longer hold. It can
// only happen if the peer acknowledged those bytes and then asked for them again, so it is a
// protocol violation rather than a condition to recover from.
var errReplayUnavailable = errors.New("the bytes needed to resume have already been acknowledged and discarded")

// sendBuffer holds the tail of the outgoing payload stream so it can be replayed after an
// unexpected disconnect.
//
// The buffer, not the websocket, is the source of truth for what has been sent: bytes are
// appended before they are written, so a write that fails on a dying connection loses nothing.
// SSH runs its own sequence numbers and MACs over the byte stream (RFC 4253 section 6), so a
// resumed connection has to deliver exactly the bytes the peer missed, once, in order - a
// single lost or duplicated byte disconnects the session with a corrupted MAC.
type sendBuffer struct {
	mu sync.Mutex
	// Total payload bytes appended since the session began.
	sent int64
	// Total payload bytes the peer has confirmed writing to its destination. buf holds
	// exactly the range [acked, sent).
	acked int64
	buf   []byte
	limit int
}

func newSendBuffer(limit int) *sendBuffer {
	return &sendBuffer{limit: limit}
}

// append records payload as sent. It fails once the unacknowledged window would outgrow the
// limit, which means the peer has stopped acknowledging and the stream can no longer be repaired.
func (b *sendBuffer) append(payload []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf)+len(payload) > b.limit {
		return fmt.Errorf("%w: %d unacknowledged bytes, limit %d", errSendWindowExhausted, len(b.buf), b.limit)
	}
	b.buf = append(b.buf, payload...)
	b.sent += int64(len(payload))
	return nil
}

// ack discards everything the peer has confirmed delivering. A stale or duplicated
// acknowledgement is ignored rather than treated as an error: acks are sent periodically and
// may arrive out of order relative to a resume.
func (b *sendBuffer) ack(delivered int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if delivered <= b.acked || delivered > b.sent {
		return
	}
	b.buf = b.buf[delivered-b.acked:]
	b.acked = delivered
}

// replayFrom returns the bytes the peer is missing: everything from the offset it last delivered
// up to what we have sent. The returned slice is a copy, so the caller can write it without
// holding the lock.
func (b *sendBuffer) replayFrom(delivered int64) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if delivered < b.acked {
		return nil, fmt.Errorf("%w: peer asked for offset %d, buffer starts at %d", errReplayUnavailable, delivered, b.acked)
	}
	if delivered > b.sent {
		return nil, fmt.Errorf("peer claims to have delivered %d bytes but only %d were sent", delivered, b.sent)
	}
	missing := b.buf[delivered-b.acked:]
	return append([]byte(nil), missing...), nil
}

// controlMessage is exchanged as a websocket text frame; payload always stays binary. It carries
// the sender's delivered count, both as the periodic acknowledgement and as the first frame of a
// resumed connection, where it tells the peer where to replay from.
//
// Text frames are only ever sent once resume has been negotiated: a server from an older CLI
// treats any non-binary frame as a protocol error and ends the session.
type controlMessage struct {
	Delivered int64 `json:"delivered"`
}

// deliveredCount is how many payload bytes this side has written to its destination, or zero when
// resume was not negotiated.
func (pc *proxyConnection) deliveredCount() int64 {
	if !pc.resumable() {
		return 0
	}
	return pc.resume.delivered.Load()
}

// sendGate throttles the sending loop while a reattach is in progress.
//
// It is deliberately not the write mutex. A reattach has to wait for the peer - the client for its
// dial to be answered, the server for the client to come back - and the write mutex is exactly
// what the other side needs to finish the reattach, so holding it across that wait deadlocks the
// server. Correctness rests on the write mutex plus appending before writing; this only stops the
// sending loop from filling the whole replay window while the connection is down.
type sendGate struct {
	mu     sync.Mutex
	waitCh chan struct{}
}

func (g *sendGate) park() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.waitCh == nil {
		g.waitCh = make(chan struct{})
	}
}

func (g *sendGate) release() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.waitCh != nil {
		close(g.waitCh)
		g.waitCh = nil
	}
}

// wait blocks until the gate is released, and returns immediately when it is already open.
func (g *sendGate) wait(ctx context.Context) error {
	g.mu.Lock()
	waitCh := g.waitCh
	g.mu.Unlock()
	if waitCh == nil {
		return nil
	}
	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// reattach repairs a dropped connection without the SSH session noticing: both ends replay
// whatever the other is missing, so the byte stream continues exactly where it left off.
//
// The receiving loop owns this, because it is the goroutine that must stop reading the dead
// connection. The sending loop is only throttled: its payload is already in the replay buffer.
func (pc *proxyConnection) reattach(ctx context.Context) error {
	pc.resume.gate.park()
	defer pc.resume.gate.release()

	// The server cannot dial - its client is behind the driver proxy - so it waits for an inbound
	// reattach request instead.
	if pc.createWebsocketConnection == nil {
		return pc.awaitReattach(ctx)
	}
	return pc.dialReattach(ctx)
}

// dialReattach reattaches from the client side: redial the session, learn where the server got
// to, and replay what it is missing.
func (pc *proxyConnection) dialReattach(ctx context.Context) error {
	// Held for the whole reattach, dial included, so a handover tick cannot start one while this is
	// in flight. A handover needs the receiving loop to answer it, and the receiving loop is the
	// goroutine running this - the handover would wait out its own timeout and end a session that
	// was about to be repaired. Safe to hold across the dial on this side: only the client
	// initiates handovers, and nothing the reattach waits on needs this lock. The server's side
	// cannot do the same, which is what sendGate is for.
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	budgetCtx, cancel := context.WithTimeout(ctx, proxyResumeBudget)
	defer cancel()

	log.Warnf(ctx, "SSH tunnel connection dropped, reattaching to the session...")
	conn, err := pc.redial(budgetCtx)
	if err != nil {
		return err
	}

	// The server's first frame says how much of our output it wrote, which is where we replay
	// from. It replays what we are missing right after, and those frames wait in the socket
	// until the receiving loop picks the new connection up.
	serverDelivered, err := readResumeHandshake(conn)
	if err != nil {
		conn.Close()
		return err
	}
	if err := pc.replayTo(conn, serverDelivered); err != nil {
		conn.Close()
		return err
	}
	pc.conn.Store(conn)
	log.Warnf(ctx, "SSH tunnel connection reattached, the session continues")
	return nil
}

func (pc *proxyConnection) redial(ctx context.Context) (*websocket.Conn, error) {
	var lastErr error
	for {
		conn, err := pc.createWebsocketConnection(ctx, DialRequest{
			ConnID:        pc.connID,
			Delivered:     pc.resume.delivered.Load(),
			Reattach:      true,
			ResumeCapable: true,
		})
		if err == nil {
			return conn, nil
		}
		lastErr = err
		log.Debugf(ctx, "Reattach dial failed, retrying: %v", err)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("gave up reattaching after %v: %w", proxyResumeBudget, lastErr)
		case <-time.After(proxyResumeRetryBackoff):
		}
	}
}

// awaitReattach reattaches from the server side by waiting for the client to come back. The
// session - sshd, the client slot, and the buffered output - is held for the grace period.
//
// It first announces that it has stopped delivering, which is what makes its delivered count
// stable for acceptReattach to report. Both channels are buffered, so a client that reattaches
// before this side has even noticed the drop is picked up rather than missed.
func (pc *proxyConnection) awaitReattach(ctx context.Context) error {
	log.Infof(ctx, "Connection dropped, holding the session for up to %v for the client to reattach", proxyResumeGrace)
	select {
	case pc.resume.parked <- struct{}{}:
	default:
		// A previous park is still queued, which means acceptReattach has not consumed it yet.
		// Nothing to add: it is about to read a delivered count that is already stable.
	}
	select {
	case <-pc.resume.resumed:
		// acceptReattach has already replayed and installed the new connection.
		log.Info(ctx, "Client reattached to the session")
		return nil
	case <-time.After(proxyResumeGrace):
		return fmt.Errorf("the client did not reattach within %v", proxyResumeGrace)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// awaitParked waits until the receiving loop has stopped delivering to sshd, so this side's
// delivered count cannot move while a reattach reports it.
//
// Without this the client can reattach while the server is still draining data buffered on the
// dying connection: the greeting would carry a stale offset, the client would replay from it, and
// the server would write those bytes to sshd twice. SSH would then fail on a corrupted MAC - the
// exact failure the replay accounting exists to prevent.
func (pc *proxyConnection) awaitParked(ctx context.Context) error {
	select {
	case <-pc.resume.parked:
		return nil
	case <-time.After(proxyResumeHandshakeTimeout):
		return fmt.Errorf("the receiving loop did not stop delivering within %v", proxyResumeHandshakeTimeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// acceptReattach installs a client's replacement connection on the server side: announce where we
// got to, replay what the client is missing, and hand the connection to the parked receiving loop.
//
// Called from the HTTP handler goroutine, so it takes the write lock the loops use.
func (pc *proxyConnection) acceptReattach(ctx context.Context, w http.ResponseWriter, r *http.Request, clientDelivered int64) error {
	select {
	case <-pc.ready:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Retire the dying connection before anything else. The receiving loop may still be draining
	// data buffered on it - a reset that only tore down the client's leg leaves this side's read
	// succeeding for a while, then hanging - and every byte it delivers moves the offset the
	// greeting below is about to report.
	previous := pc.conn.Load()
	if previous != nil {
		previous.Close()
	}
	if err := pc.awaitParked(ctx); err != nil {
		return err
	}

	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	conn, err := pc.acceptWebsocketConnection(w, r)
	if err != nil {
		return fmt.Errorf("failed to accept the reattached connection: %w", err)
	}
	// The offset first, then the payload: the client reads exactly one control frame before it
	// hands the connection to its own receiving loop.
	if err := pc.sendResumeHandshake(conn); err != nil {
		conn.Close()
		return err
	}
	if err := pc.replayTo(conn, clientDelivered); err != nil {
		conn.Close()
		return err
	}
	pc.conn.Store(conn)

	select {
	case pc.resume.resumed <- conn:
	default:
		// The loop already gave up, or a previous signal is still queued. Either way the session
		// is past saving; the loop's own grace timer reports it.
		log.Warnf(ctx, "Reattach signal could not be delivered to the receiving loop")
	}
	return nil
}

// replayTo sends the peer everything it is missing, given how much it says it delivered.
func (pc *proxyConnection) replayTo(conn *websocket.Conn, peerDelivered int64) error {
	missing, err := pc.resume.sendBuf.replayFrom(peerDelivered)
	if err != nil {
		return err
	}
	if len(missing) == 0 {
		return nil
	}
	return conn.WriteMessage(websocket.BinaryMessage, missing)
}

// sendResumeHandshake announces our delivered count as the first frame of a reattached
// connection, so the peer knows where to replay from. Written directly rather than through
// sendMessage: the connection is not installed yet, and this must not be recorded as payload.
func (pc *proxyConnection) sendResumeHandshake(conn *websocket.Conn) error {
	payload, err := json.Marshal(controlMessage{Delivered: pc.resume.delivered.Load()})
	if err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

// readResumeHandshake reads the delivered count the peer sends as the first frame of a reattached
// connection. The deadline bounds the wait: the connection is new, but the peer may be wedged.
func readResumeHandshake(conn *websocket.Conn) (int64, error) {
	if err := conn.SetReadDeadline(time.Now().Add(proxyResumeHandshakeTimeout)); err != nil {
		return 0, err
	}
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	mt, data, err := conn.ReadMessage()
	if err != nil {
		return 0, fmt.Errorf("failed to read the reattach handshake: %w", err)
	}
	if mt != websocket.TextMessage {
		return 0, fmt.Errorf("expected a reattach handshake control frame, got websocket message type %d", mt)
	}
	var msg controlMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return 0, fmt.Errorf("failed to decode the reattach handshake: %w", err)
	}
	return msg.Delivered, nil
}
