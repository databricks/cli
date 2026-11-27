package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
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
	resumeGrace         time.Duration
	onSessionEnd        func(error)
	singleSession       bool
	sessionUsed         bool
	sessionMu           sync.Mutex
}

// ServerOption configures the SSH transport capabilities.
type ServerOption func(*proxyServer)

// WithSessionCompletion reports process startup failures or a completed logical session.
// Established sessions report nil; handovers and failed upgrades do not notify.
func WithSessionCompletion(notify func(error)) ServerOption {
	return func(server *proxyServer) { server.onSessionEnd = notify }
}

// WithSingleSession permits handovers but never starts a second SSH session.
func WithSingleSession() ServerOption {
	return func(server *proxyServer) { server.singleSession = true }
}

func NewProxyServer(ctx context.Context, connections *ConnectionsManager, createServerCommand createServerCommandFunc, options ...ServerOption) *proxyServer {
	server := &proxyServer{
		ctx:                 ctx,
		connections:         connections,
		createServerCommand: createServerCommand,
		resumeGrace:         proxyResumeGrace,
	}
	for _, option := range options {
		option(server)
	}
	return server
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
		if query.Get(ResumeVersionParameter) != strconv.Itoa(ResumeProtocolVersion) {
			return DialRequest{}, fmt.Errorf("session resume requires %s=%d", ResumeVersionParameter, ResumeProtocolVersion)
		}
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
	if query.Has(ResumeVersionParameter) && !req.ResumeCapable {
		return DialRequest{}, errors.New("'resume_version' requires the 'delivered' query parameter")
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
	// Failed upgrades can retry; invoking the process factory consumes this server.
	var command *exec.Cmd
	if server.singleSession {
		server.sessionMu.Lock()
		if server.sessionUsed {
			server.sessionMu.Unlock()
			http.Error(w, "SSH session already used", http.StatusServiceUnavailable)
			return
		}
		server.sessionUsed = true
		server.sessionMu.Unlock()
		defer func() {
			if command == nil {
				server.sessionMu.Lock()
				server.sessionUsed = false
				server.sessionMu.Unlock()
			}
		}()
	}

	// The server never dials, so it passes no connection factory: reattaching, for it, means
	// waiting for the client to come back.
	var conn *proxyConnection
	if req.ResumeCapable {
		conn = newResumableProxyConnection(nil)
		conn.resume.grace = server.resumeGrace
	} else {
		conn = newProxyConnection(nil)
	}
	if !server.connections.TryAdd(id, conn) {
		log.Info(ctx, "Maximum clients reached, rejecting connection")
		http.Error(w, "Maximum clients reached", http.StatusServiceUnavailable)
		return
	}
	var err error
	defer func() {
		server.connections.Remove(id)
		if command != nil && server.onSessionEnd != nil {
			var startErr error
			if command.Process == nil {
				startErr = err
			}
			server.onSessionEnd(startErr)
		}
	}()

	log.Infof(ctx, "Starting proxy server for new connection, count: %d", server.connections.Count())
	err = runServerProxy(ctx, conn, func(ctx context.Context) *exec.Cmd {
		command = server.createServerCommand(ctx)
		return command
	}, w, r)
	if err != nil {
		log.Errorf(ctx, "Proxy server error: %v", err)
	} else {
		log.Info(ctx, "Proxy server finished")
	}
}

func runServerProxy(ctx context.Context, proxy *proxyConnection, createServerCommand createServerCommandFunc, w http.ResponseWriter, r *http.Request) error {
	err := proxy.accept(w, r)
	if err != nil {
		return fmt.Errorf("failed to upgrade to websockets: %w", err)
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
		return fmt.Errorf("failed to get stdin pipe: %w", err)
	}
	defer sshdStdin.Close()

	sshdStdout, err := serverCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	defer sshdStdout.Close()

	err = serverCmd.Start()
	if err != nil {
		return fmt.Errorf("failed to start SSHD process: %w", err)
	}
	defer releaseServerCommand(ctx, serverCmd)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		// Let the sending loop drain stdout and receive its final acknowledgment
		// before closing the websocket, even if the process has already exited.
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
	err := conn.close()
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
