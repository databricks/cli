package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var (
	errProxyEOF             = errors.New("proxy EOF error")
	errSendingLoopStopped   = errors.New("sending loop stopped")
	errReceivingLoopStopped = errors.New("receiving loop stopped")
)

const (
	// Same as gorilla/websocket default read/write buffer sizes. Bigger payloads will be split into multiple ws frames.
	proxyBufferSize = 4 * 1024
	// Timeout for the full handover process, when initiated by the client.
	proxyHandoverInitTimeout = 30 * time.Second
	// Timeout for the handover process, when accepted by the server.
	proxyHandoverAcceptTimeout = 25 * time.Second
)

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
}

type createWebsocketConnectionFunc func(ctx context.Context, connID string) (*websocket.Conn, error)

func newProxyConnection(createConn createWebsocketConnectionFunc) *proxyConnection {
	return &proxyConnection{
		connID:                    uuid.NewString(),
		createWebsocketConnection: createConn,
	}
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
		return errors.Join(pc.close(), pc.closeSource(src))
	})
	err := g.Wait()
	if err == nil || isNormalClosure(err) {
		return nil
	}
	return err
}

func (pc *proxyConnection) connect(ctx context.Context) error {
	conn, err := pc.createWebsocketConnection(ctx, pc.connID)
	if err != nil {
		return err
	}
	pc.conn.Store(conn)
	return nil
}

func (pc *proxyConnection) accept(w http.ResponseWriter, r *http.Request) error {
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
			// This will block during handover - we stop sending anything except the close message.
			// Meanwhile the "src" (sshd server stdout or ssh client stdin) will be buffered/blocked on the OS side until we start reading from it again.
			err := pc.sendMessage(websocket.BinaryMessage, b[:n])
			if err != nil {
				return fmt.Errorf("failed to send message: %w", err)
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

func (pc *proxyConnection) sendMessage(mt int, data []byte) error {
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()
	conn := pc.conn.Load()
	return conn.WriteMessage(mt, data)
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
					closeConnSignal = fmt.Errorf("failed to read from websocket during handover: %w", err)
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
				} else {
					return fmt.Errorf("failed to read from websocket: %w", err)
				}
			}
		}

		if mt != websocket.BinaryMessage {
			return errors.New("received non-binary websocket message")
		}
		if _, err := dst.Write(data); err != nil {
			return fmt.Errorf("failed to copy to writer: %w", err)
		}
	}
}

func (pc *proxyConnection) close() error {
	// Keep in mind that pc.sendMessage blocks during handover
	err := pc.sendMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		if isNormalClosure(err) || errors.Is(err, websocket.ErrCloseSent) {
			return nil
		} else {
			return fmt.Errorf("failed to send close message: %w", err)
		}
	}
	return nil
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
	newConn, err := pc.createWebsocketConnection(handoverCtx, pc.connID)
	if err != nil {
		return fmt.Errorf("failed to create new websocket connection: %w", err)
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
