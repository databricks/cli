package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

// clientHandshakeTimeout bounds how long the client waits for the first byte from the SSH server
// after the proxy websocket is established. A healthy sshd sends its identification string
// immediately (RFC 4253 §4.2), so if nothing arrives the server most likely failed to launch
// sshd — e.g. the cluster's container image has no OpenSSH server. The server can hold the
// websocket open in that state, leaving the proxy loops blocked forever, so we bail out instead
// of letting the ssh client hang until its ConnectTimeout. It is a var so tests can shorten it.
var clientHandshakeTimeout = 30 * time.Second

var errHandshakeTimeout = errors.New("no response from the SSH server: the cluster's container image may be missing an OpenSSH server (sshd) — ensure 'openssh-server' is installed and check the SSH server job run logs")

// firstByteWriter signals (once) the first time any data is written through it, then forwards
// transparently. The client uses it to detect that the SSH server has started responding.
type firstByteWriter struct {
	w         io.Writer
	signaled  atomic.Bool
	firstByte chan struct{}
}

func (f *firstByteWriter) Write(p []byte) (int, error) {
	if len(p) > 0 && f.signaled.CompareAndSwap(false, true) {
		close(f.firstByte)
	}
	return f.w.Write(p)
}

// logPongs wraps a connection factory so every connection it creates — the initial one and each
// one a handover creates — logs the pongs coming back for our keepalive pings. Debug visibility only:
// the receiving loop stays the only judge of whether a connection is alive.
func logPongs(ctx context.Context, createConn createWebsocketConnectionFunc) createWebsocketConnectionFunc {
	return func(connCtx context.Context, req DialRequest) (*websocket.Conn, error) {
		conn, err := createConn(connCtx, req)
		if err != nil {
			return nil, err
		}
		conn.SetPongHandler(func(string) error {
			log.Debugf(ctx, "Received websocket keepalive pong")
			return nil
		})
		return conn, nil
	}
}

// RunClientProxy proxies the SSH byte stream over a websocket to the tunnel server.
//
// resumable turns on the resume protocol, which lets a session survive an unexpected disconnect.
// It must only be set when the server is known to speak it: an older server answers a reattach
// request by starting a fresh sshd, and replaying into that corrupts the SSH stream instead of
// repairing it.
func RunClientProxy(ctx context.Context, src io.ReadCloser, dst io.Writer, requestHandoverTick func() <-chan time.Time, keepaliveInterval time.Duration, resumable bool, createConn createWebsocketConnectionFunc) error {
	var proxy *proxyConnection
	wrappedConn := logPongs(ctx, createConn)
	if resumable {
		proxy = newResumableProxyConnection(wrappedConn, proxyResumeBufferLimit)
	} else {
		proxy = newProxyConnection(wrappedConn)
	}
	log.Infof(ctx, "Establishing SSH proxy connection...")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := proxy.connect(ctx); err != nil {
		return errors.Join(ErrConnectFailed, fmt.Errorf("failed to connect to proxy: %w", err))
	}
	defer proxy.close(ctx)
	log.Infof(ctx, "SSH proxy connection established")

	wrappedDst := &firstByteWriter{w: dst, firstByte: make(chan struct{})}

	// Run the proxy loops in the background. We don't wait on them directly: if the server holds
	// the websocket open without ever launching sshd, both loops can block forever (the sending
	// loop on os.Stdin, the receiving loop on ReadMessage), so g.Wait would never return.
	done := make(chan error, 1)
	go func() {
		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error {
			for {
				select {
				case <-gCtx.Done():
					// Return nil, not gCtx.Err(): this helper loop does not decide the session's
					// outcome, proxy.start does. proxy.start's goroutine below cancels the context
					// (its deferred cancel) before the errgroup records proxy.start's return value,
					// so a gCtx.Err() here can be recorded as the group's first error and mask that
					// real error — which normalizeProxyError would then swallow into a silent nil.
					return nil
				case <-requestHandoverTick():
					if err := proxy.initiateHandover(gCtx); err != nil {
						// A handover that never got past its dial leaves the current connection
						// untouched and still carrying traffic, so ending the session over it
						// would throw away a working tunnel - the failure mode customers see as
						// a drop every handover interval. The next tick tries again. Deferring
						// the auth refresh is safe: the driver proxy authenticates a websocket
						// at upgrade time, so a live connection is not re-checked. Logged at
						// debug because nothing changed for the user, and this would otherwise
						// write into their interactive terminal.
						if errors.Is(err, errHandoverDialFailed) {
							log.Debugf(gCtx, "Could not open a replacement connection for the auth handover, staying on the current one: %v", err)
							continue
						}
						return errors.Join(ErrHandoverFailed, err)
					}
				}
			}
		})
		g.Go(func() error {
			// Keep the websocket carrying traffic while the SSH session is idle. Both proxy loops
			// are data-driven, so an idle session puts no frames on the connection at all and the
			// server side reaps the stream it then considers dead (websocket close 4000).
			ticker := time.NewTicker(keepaliveInterval)
			defer ticker.Stop()
			for {
				select {
				case <-gCtx.Done():
					// nil, not gCtx.Err(): see the handover loop above — a helper stopping must
					// not mask proxy.start's error as the errgroup's first error.
					return nil
				case <-ticker.C:
					// A ping that ticks during a handover goes to the connection being replaced and
					// may simply fail. Harmless: a handover establishes a fresh connection, which
					// resets the peer's idle clock anyway, and the next tick uses the new one.
					if err := proxy.sendPing(); err != nil {
						// Not fatal, but not harmless either: gorilla puts the connection into a
						// permanent write-error state after any failed write, so nothing more can
						// be sent on it. Reads are unaffected and may still be delivering output
						// the user is waiting on, so the session is left to end the way it would
						// anyway — the next write fails and the sending loop reports it.
						log.Warnf(gCtx, "Failed to send websocket keepalive ping, the connection can no longer send: %v", err)
					} else {
						// The driver proxy does not return pongs (verified end to end), so this
						// line is the only evidence in a customer's log that pings were flowing.
						log.Debugf(gCtx, "Sent websocket keepalive ping")
					}
				}
			}
		})
		g.Go(func() error {
			// When proxy.start returns (EOF from ssh, or the server closing the connection),
			// cancel so the handover and keepalive goroutines stop too and g.Wait can return.
			defer cancel()
			return proxy.start(gCtx, src, wrappedDst)
		})
		done <- g.Wait()
	}()

	select {
	case err := <-done:
		// Session ended before the handshake even started (e.g. the server closed the connection).
		return normalizeProxyError(err)
	case <-wrappedDst.firstByte:
		// The server is responding; the handshake is underway. Wait for the session to finish.
		return normalizeProxyError(<-done)
	case <-time.After(clientHandshakeTimeout):
		// cancel() (deferred) unblocks what it can; the process exits and reclaims any goroutine
		// still stuck on os.Stdin. ssh then fails fast instead of hanging on its ConnectTimeout.
		return errHandshakeTimeout
	case <-ctx.Done():
		return nil
	}
}

// normalizeProxyError treats a clean finish or a context cancellation (our own exit signal, or the
// user interrupting) as success; anything else is a real proxy error.
func normalizeProxyError(err error) error {
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}
