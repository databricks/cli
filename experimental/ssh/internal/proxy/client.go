package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/databricks/cli/libs/log"
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

func RunClientProxy(ctx context.Context, src io.ReadCloser, dst io.Writer, requestHandoverTick func() <-chan time.Time, createConn createWebsocketConnectionFunc) error {
	proxy := newProxyConnection(createConn)
	log.Infof(ctx, "Establishing SSH proxy connection...")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := proxy.connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to proxy: %w", err)
	}
	defer proxy.close()
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
					return gCtx.Err()
				case <-requestHandoverTick():
					if err := proxy.initiateHandover(gCtx); err != nil {
						return err
					}
				}
			}
		})
		g.Go(func() error {
			// When proxy.start returns (EOF from ssh, or the server closing the connection),
			// cancel so the handover goroutine stops too and g.Wait can return.
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
