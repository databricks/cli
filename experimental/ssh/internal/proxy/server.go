package proxy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

const serverProcessTerminationTimeout = 10 * time.Second

type createServerCommandFunc func(ctx context.Context) (*exec.Cmd, error)

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
	ctx := log.NewContext(server.ctx, log.GetLogger(server.ctx).With("session", id))
	if conn, exists := server.connections.Get(id); exists && conn != nil {
		server.handleExistingConnection(ctx, w, r, conn)
	} else {
		server.handleNewConnection(ctx, w, r, id)
	}
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

func (server *proxyServer) handleNewConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	conn := newProxyConnection(nil)
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
	serverCmd, err := createServerCommand(cmdCtx)
	if err != nil {
		return fmt.Errorf("failed to create server command: %v", err)
	}
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
