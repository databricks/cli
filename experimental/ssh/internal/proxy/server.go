package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

const serverProcessTerminationTimeout = 10 * time.Second

type createServerCommandFunc func(ctx context.Context) *exec.Cmd

type proxyServer struct {
	ctx                 context.Context
	connections         *ConnectionsManager
	createServerCommand createServerCommandFunc
}

func NewProxyServer(ctx context.Context, connections *ConnectionsManager, createServerCommand createServerCommandFunc) *proxyServer {
	return &proxyServer{
		ctx:                 ctx,
		connections:         connections,
		createServerCommand: createServerCommand,
	}
}

func (server *proxyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}
	req, err := parseDialRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := log.NewContext(server.ctx, log.GetLogger(server.ctx).With("session", id))
	conn, exists := server.connections.Get(id)
	switch {
	case exists && conn != nil && req.Reattach:
		server.handleReattach(ctx, w, r, conn, req)
	case exists && conn != nil:
		server.handleExistingConnection(ctx, w, r, conn)
	case req.Reattach:
		// The session is gone: its grace period expired, or the server restarted. Say so instead
		// of starting a fresh one. A new sshd would answer the client's replayed bytes with a
		// fresh SSH handshake, and ssh would fail on a corrupted MAC rather than a clear error.
		log.Info(ctx, "Reattach requested for a session that no longer exists")
		http.Error(w, "Session no longer exists", http.StatusGone)
	default:
		server.handleNewConnection(ctx, w, r, id, req)
	}
}

// parseDialRequest reads the resume protocol's query parameters. Both are absent for a client
// that does not speak it, which leaves the session non-resumable on this side too.
func parseDialRequest(r *http.Request) (DialRequest, error) {
	query := r.URL.Query()
	req := DialRequest{
		ConnID:   query.Get("id"),
		Reattach: query.Get("reattach") == "1",
	}
	if raw := query.Get("delivered"); raw != "" {
		delivered, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || delivered < 0 {
			return DialRequest{}, fmt.Errorf("invalid 'delivered' query parameter: %q", raw)
		}
		req.Delivered = delivered
		req.ResumeCapable = true
	}
	if req.Reattach && !req.ResumeCapable {
		return DialRequest{}, errors.New("'reattach' requires the 'delivered' query parameter")
	}
	return req, nil
}

func (server *proxyServer) handleReattach(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *proxyConnection, req DialRequest) {
	if !conn.resumable() {
		log.Info(ctx, "Reattach requested for a session that was not started as resumable")
		http.Error(w, "Session is not resumable", http.StatusConflict)
		return
	}
	log.Info(ctx, "Client reattaching to a dropped connection")
	if err := conn.acceptReattach(ctx, w, r, req.Delivered); err != nil {
		log.Errorf(ctx, "Failed to accept the reattach: %v", err)
		return
	}
	log.Info(ctx, "Reattach accepted")
}

func (server *proxyServer) handleExistingConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *proxyConnection) {
	log.Info(ctx, "Client already connected, accepting handover")
	err := conn.acceptHandover(ctx, w, r)
	if err != nil {
		log.Errorf(ctx, "Failed to accept handover: %v", err)
		http.Error(w, "Handover failed", http.StatusInternalServerError)
	} else {
		log.Info(ctx, "Handover accepted")
	}
}

func (server *proxyServer) handleNewConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, id string, req DialRequest) {
	// The server never dials, so it passes no connection factory: reattaching, for it, means
	// waiting for the client to come back.
	var conn *proxyConnection
	if req.ResumeCapable {
		conn = newResumableProxyConnection(nil, proxyResumeBufferLimit)
	} else {
		conn = newProxyConnection(nil)
	}
	if !server.connections.TryAdd(id, conn) {
		log.Info(ctx, "Maximum clients reached, rejecting connection")
		http.Error(w, "Maximum clients reached", http.StatusServiceUnavailable)
		return
	}
	defer server.connections.Remove(id)

	log.Infof(ctx, "Starting proxy server for new connection, count: %d", server.connections.Count())
	err := runServerProxy(ctx, conn, server.createServerCommand, w, r)
	if err != nil {
		log.Errorf(ctx, "Proxy server error: %v", err)
	} else {
		log.Info(ctx, "Proxy server finished")
	}
}

func runServerProxy(ctx context.Context, proxy *proxyConnection, createServerCommand createServerCommandFunc, w http.ResponseWriter, r *http.Request) error {
	err := proxy.accept(w, r)
	if err != nil {
		return fmt.Errorf("failed to upgrade to websockets: %v", err)
	}
	defer closeProxyConnection(ctx, proxy)
	log.Infof(ctx, "New connection accepted")

	cmdCtx, cancelServerCommand := context.WithCancel(ctx)
	defer cancelServerCommand()
	serverCmd := createServerCommand(cmdCtx)
	// Fail-safe that ensures we aren't blocked waiting for a stuck sshd process to terminate (in releaseServerCommand)
	serverCmd.WaitDelay = serverProcessTerminationTimeout
	serverCmd.Stderr = os.Stderr

	sshdStdin, err := serverCmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin pipe: %v", err)
	}

	sshdStdout, err := serverCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %v", err)
	}

	err = serverCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start SSHD process: %v", err)
	}
	defer releaseServerCommand(ctx, serverCmd)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer closeProxyConnection(ctx, proxy)
		// Waiting on the underlying Process, not the command itself.
		// Command.Wait needs to be called to release all resources,
		// but it's only safe to do so after we've finished reading from stdout,
		// so we do it in releaseSSHDResources after the proxy is closed.
		state, err := serverCmd.Process.Wait()
		log.Infof(ctx, "SSHD process exited with state: %v, %v", state, err)
		return err
	})

	g.Go(func() error {
		defer cancelServerCommand()
		return proxy.start(gCtx, sshdStdout, sshdStdin)
	})

	return g.Wait()
}

func closeProxyConnection(ctx context.Context, conn *proxyConnection) {
	err := conn.close(ctx)
	if err != nil {
		log.Errorf(ctx, "Failed to close websocket: %v", err)
	}
}

func releaseServerCommand(ctx context.Context, sshdCmd *exec.Cmd) {
	log.Infof(ctx, "Releasing SSHD command resources")
	err := sshdCmd.Wait()
	if err != nil {
		log.Errorf(ctx, "Failed to wait for SSHD command: %v", err)
	}
}
