package server

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"golang.org/x/sync/errgroup"
)

//go:embed jupyter-init.py
var jupyterInitScript string

const sshdProcessTerminationTimeout = 10 * time.Second

type sshHandler struct {
	ctx            context.Context
	Connections    *connectionsManager
	SSHDConfigPath string
}

func newSSHHandler(ctx context.Context, connections *connectionsManager, sshDConfigPath string) *sshHandler {
	return &sshHandler{
		ctx:            ctx,
		Connections:    connections,
		SSHDConfigPath: sshDConfigPath,
	}
}

func (handler *sshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
		return
	}
	ctx := log.NewContext(handler.ctx, log.GetLogger(handler.ctx).With("session", id))
	if conn, exists := handler.Connections.Get(id); exists && conn != nil {
		handler.handleExistingConnection(ctx, w, r, conn)
	} else {
		handler.handleNewConnection(ctx, w, r, id)
	}
}

func (handler *sshHandler) handleExistingConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, conn *proxy.ProxyConnection) {
	log.Info(ctx, "Client already connected, accepting handover")
	err := conn.AcceptHandover(ctx, w, r)
	if err != nil {
		log.Errorf(ctx, "Failed to accept handover: %v", err)
		http.Error(w, "Handover failed", http.StatusInternalServerError)
	} else {
		log.Info(ctx, "Handover accepted")
	}
}

func (handler *sshHandler) handleNewConnection(ctx context.Context, w http.ResponseWriter, r *http.Request, id string) {
	log.Info(ctx, "Accepting new connection")
	conn := proxy.NewProxyConnection(nil)
	err := conn.Accept(w, r)
	if err != nil {
		log.Errorf(ctx, "Failed to upgrade to websockets: %v", err)
		return
	}

	if !handler.Connections.TryAdd(id, conn) {
		log.Info(ctx, "Maximum clients reached, rejecting connection")
		err := conn.Close()
		if err != nil {
			log.Errorf(ctx, "Failed to close websocket: %v", err)
		}
		return
	}
	defer handler.closeProxyConnection(ctx, conn, id)

	log.Infof(ctx, "New connection accepted, count: %d, starting SSHD process", handler.Connections.Count())
	sshdCtx, cancelSSHDProcess := context.WithCancel(ctx)
	defer cancelSSHDProcess()
	sshdCmd := createSSHDProcess(sshdCtx, handler.SSHDConfigPath)
	// Fail-safe that ensures we aren't blocked waiting for a stuck sshd process to terminate (in releaseSSHDResources)
	sshdCmd.WaitDelay = sshdProcessTerminationTimeout
	sshdCmd.Stderr = os.Stderr

	sshdStdin, err := sshdCmd.StdinPipe()
	if err != nil {
		log.Errorf(ctx, "Failed to get stdin pipe: %v", err)
		return
	}

	sshdStdout, err := sshdCmd.StdoutPipe()
	if err != nil {
		log.Errorf(ctx, "Failed to get stdout pipe: %v", err)
		return
	}

	err = sshdCmd.Start()
	if err != nil {
		log.Errorf(ctx, "Failed to start SSHD process: %v", err)
		return
	}
	defer releaseSSHDResources(ctx, sshdCmd)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		defer handler.closeProxyConnection(ctx, conn, id)
		// Waiting on the underlying Process, not the command itself.
		// Command.Wait needs to be called to release all resources,
		// but it's only safe to do so after we've finished reading from stdout,
		// so we do it in releaseSSHDResources after the proxy is closed.
		state, err := sshdCmd.Process.Wait()
		log.Infof(ctx, "SSHD process exited with state: %v, %v", state, err)
		return err
	})

	g.Go(func() error {
		defer cancelSSHDProcess()
		return conn.Start(gCtx, sshdStdout, sshdStdin)
	})

	err = g.Wait()
	if err != nil && !proxy.IsNormalClosure(err) {
		log.Errorf(ctx, "SSHD handler error: %v", err)
	} else {
		log.Info(ctx, "SSHD handler finished")
	}
}

func (handler *sshHandler) closeProxyConnection(ctx context.Context, conn *proxy.ProxyConnection, id string) {
	if _, ok := handler.Connections.Get(id); !ok {
		return
	}
	handler.Connections.Remove(id)
	log.Infof(ctx, "Closing client connection, current count: %d", handler.Connections.Count())
	err := conn.Close()
	if err != nil {
		log.Errorf(ctx, "Failed to close websocket: %v", err)
	}
}

func releaseSSHDResources(ctx context.Context, sshdCmd *exec.Cmd) {
	log.Infof(ctx, "Releasing SSHD command resources")
	err := sshdCmd.Wait()
	if err != nil {
		log.Errorf(ctx, "Failed to wait for SSHD command: %v", err)
	}
}

func handleTimeout(ctx context.Context, timedOutChannel chan bool, shutdownDelay time.Duration) {
	<-timedOutChannel
	log.Info(ctx, fmt.Sprintf("No SSH clients for %v, shutting down...", shutdownDelay))
	os.Exit(0)
}

func findAvailablePort(startPort, maxAttempts int) (int, error) {
	for i := range maxAttempts {
		port := startPort + i
		addr := fmt.Sprintf(":%d", port)

		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found after %d attempts starting from port %d", maxAttempts, startPort)
}

func saveJupyterInitScript(ctx context.Context) error {
	ipythonStartupDir := os.ExpandEnv("$HOME/.ipython/profile_default/startup")

	err := os.MkdirAll(ipythonStartupDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create IPython startup directory %s: %w", ipythonStartupDir, err)
	}

	initScriptPath := filepath.Join(ipythonStartupDir, "init_script.py")
	err = os.WriteFile(initScriptPath, []byte(jupyterInitScript), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write Jupyter init script to %s: %w", initScriptPath, err)
	}

	log.Info(ctx, "Saved Jupyter init script to: "+initScriptPath)
	return nil
}

type ServerOptions struct {
	// The version that the client started this server with
	Version string
	// Maximum of concurrent ssh/ws connections
	MaxClients int
	// Delay before shutting down the server when there are no active connections
	ShutdownDelay time.Duration
	// The cluster ID that the client started this server on
	ClusterID string
	// The directory to store sshd configuration
	ConfigDir string
	// The name of the secrets scope to use for client and server keys
	SecretsScope string
	// The name of a secret containing the client's public key value
	ClientPublicKeyName string
	// The name of a secret containing the server's private key value
	ServerPrivateKeyName string
	// The name of a secret containing the server's public key value
	ServerPublicKeyName string
	// The default port to listen on (for /ssh and /metadata requests from the clients)
	DefaultPort int
	// If the default port is taken, the server will try to listen on the first free port in the DefaultPort + PortRange range
	PortRange int
}

func RunServer(ctx context.Context, client *databricks.WorkspaceClient, opts ServerOptions) error {
	port, err := findAvailablePort(opts.DefaultPort, opts.PortRange)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Info(ctx, "Starting server on "+listenAddr)

	err = workspace.SaveWorkspaceMetadata(ctx, client, opts.Version, opts.ClusterID, port)
	if err != nil {
		return fmt.Errorf("failed to save metadata to the workspace: %w", err)
	}

	sshdConfigPath, err := prepareSSHDConfig(ctx, client, opts)
	if err != nil {
		return fmt.Errorf("failed to setup SSH configuration: %w", err)
	}

	err = saveJupyterInitScript(ctx)
	if err != nil {
		return fmt.Errorf("failed to save Jupyter init script: %w", err)
	}

	connections := newConnectionsManager(opts.MaxClients, opts.ShutdownDelay)
	http.Handle("/ssh", newSSHHandler(ctx, connections, sshdConfigPath))
	go handleTimeout(ctx, connections.TimedOut, opts.ShutdownDelay)

	http.HandleFunc("/metadata", func(w http.ResponseWriter, r *http.Request) {
		currentUser, err := user.Current()
		if err != nil {
			http.Error(w, "Failed to get current user", http.StatusInternalServerError)
			return
		}
		_, err = io.WriteString(w, currentUser.Username)
		if err != nil {
			http.Error(w, "Failed to write current user", http.StatusInternalServerError)
		}
	})

	return http.ListenAndServe(listenAddr, nil)
}
