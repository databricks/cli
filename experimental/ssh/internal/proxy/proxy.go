package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

var errProxyEOF = errors.New("proxy EOF error")

const (
	proxyBufferSize                  = 4 * 1024 // Same as gorilla/websocket default read/write buffer sizes. Bigger payloads will be split into multiple ws frames.
	proxyHandoverInitTimeout         = 30 * time.Second
	proxyHandoverAcceptTimeout       = 25 * time.Second
	proxyHandoverAckCloseConnTimeout = 15 * time.Second
)

type proxyConnection struct {
	connID                    string
	conn                      atomic.Value // *websocket.Conn
	connChanged               sync.Cond
	createWebsocketConnection createWebsocketConnectionFunc

	handoverMutex           sync.Mutex
	isHandover              atomic.Bool
	currentConnectionClosed chan error
}

type createWebsocketConnectionFunc func(ctx context.Context, connID string) (*websocket.Conn, error)

func newProxyConnection(createConn createWebsocketConnectionFunc) *proxyConnection {
	return &proxyConnection{
		connID:                    uuid.NewString(),
		connChanged:               sync.Cond{L: &sync.Mutex{}},
		currentConnectionClosed:   make(chan error),
		createWebsocketConnection: createConn,
	}
}

func (pc *proxyConnection) start(ctx context.Context, src io.Reader, dst io.Writer) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return pc.runSendingLoop(gCtx, src)
	})
	g.Go(func() error {
		return pc.runReceivingLoop(gCtx, dst)
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
	pc.connChanged.Broadcast()
	return nil
}

func (pc *proxyConnection) accept(w http.ResponseWriter, r *http.Request) error {
	conn, err := pc.acceptWebsocketConnection(w, r)
	if err != nil {
		return err
	}
	pc.conn.Store(conn)
	pc.connChanged.Broadcast()
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
	conn := pc.conn.Load().(*websocket.Conn)
	return conn.WriteMessage(mt, data)
}

func (pc *proxyConnection) runReceivingLoop(ctx context.Context, dst io.Writer) error {
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		conn := pc.conn.Load().(*websocket.Conn)
		mt, data, err := conn.ReadMessage()
		if err != nil {
			// During handover a normal closure is expected, but any other error must stop the read loop (and eventually terminate the ssh session).
			if pc.isHandover.Load() {
				var closeConnSignal error
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					closeConnSignal = fmt.Errorf("failed to read from websocket during handover: %w", err)
				}
				if err := pc.signalClosedConnection(closeConnSignal); err != nil {
					return err
				}
				// Next time we read, we want to read from the new connection.
				// While we wait for the handover to complete, the new connection might be getting incoming messages.
				// They will be buffered by the TCP stack and will be read by us after the handover is complete.
				pc.waitForNewConnection(conn)
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

func (pc *proxyConnection) signalClosedConnection(err error) error {
	select {
	case pc.currentConnectionClosed <- err:
		return err
	case <-time.After(proxyHandoverAckCloseConnTimeout):
		return fmt.Errorf("timeout waiting for acknowledgement of old connection closed message: %w", err)
	}
}

func (pc *proxyConnection) waitForNewConnection(conn *websocket.Conn) {
	pc.connChanged.L.Lock()
	defer pc.connChanged.L.Unlock()
	for pc.conn.Load() == conn {
		pc.connChanged.Wait()
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

func (pc *proxyConnection) initiateHandover(ctx context.Context) error {
	// Blocks proxying any outgoing messages during the entire handover
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	// When handover flag is set, the receiving loop handles a close message from the current connection
	// as a signal to finish the handover and switch to the new connection.
	pc.isHandover.Store(true)
	defer pc.isHandover.Store(false)

	ctx, cancel := context.WithTimeout(ctx, proxyHandoverInitTimeout)
	defer cancel()

	// Create a new websocket connection by sending an /ssh?id=<connID> request to the server.
	// When server realises it's an ID of an existing connection, it will start AcceptHandover process.
	newConn, err := pc.createWebsocketConnection(ctx, pc.connID)
	if err != nil {
		return fmt.Errorf("failed to create new websocket connection: %w", err)
	}

	// Wait for the server to close the old connection
	select {
	case err := <-pc.currentConnectionClosed:
		if err != nil {
			newConn.Close()
			return fmt.Errorf("connection handover failed: %w", err)
		}
	case <-ctx.Done():
		newConn.Close()
		return ctx.Err()
	}

	pc.conn.Store(newConn)
	pc.connChanged.Broadcast()

	return nil
}

func (pc *proxyConnection) acceptHandover(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Blocks proxying any outgoing messages during the entire handover
	pc.handoverMutex.Lock()
	defer pc.handoverMutex.Unlock()

	// When handover flag is set, the receiving loop handles a close message from the current connection
	// as a signal to finish the handover and switch to the new connection.
	pc.isHandover.Store(true)
	defer pc.isHandover.Store(false)

	ctx, cancel := context.WithTimeout(ctx, proxyHandoverAcceptTimeout)
	defer cancel()

	newConn, err := pc.acceptWebsocketConnection(w, r)
	if err != nil {
		return fmt.Errorf("failed to accept new websocket connection: %w", err)
	}

	// Signal the client to complete handover by closing the old connection.
	// Not using pc.sendMessage here, because it's blocked by the handover mutex.
	currentConn := pc.conn.Load().(*websocket.Conn)
	err = currentConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "handover"))
	if err != nil {
		newConn.Close()
		return fmt.Errorf("failed to send close message to the current connection: %w", err)
	}

	// Wait for the client to acknowledge the closure of the old connection.
	// On the client its done automatically by the websocket library with the default close handler.
	// On the server we then receive a close error in the RunReceivingLoop and signal about it to the handoverOldConnClosed channel.
	select {
	case err := <-pc.currentConnectionClosed:
		if err != nil {
			newConn.Close()
			return fmt.Errorf("connection handover failed: %w", err)
		}
	case <-ctx.Done():
		newConn.Close()
		return ctx.Err()
	}

	pc.conn.Store(newConn)
	pc.connChanged.Broadcast()

	return nil
}

func isNormalClosure(err error) bool {
	return websocket.IsCloseError(err, websocket.CloseNormalClosure) || errors.Is(err, errProxyEOF)
}
