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
)

//go:embed jupyter-init.py
var jupyterInitScript string

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
	SecretScopeName string
	// The name of a secret containing the server's private key value
	ServerPrivateKeyName string
	// The name of a secret containing the server's public key value
	ServerPublicKeyName string
	// The name of a secret containing the client's public key (authorized key)
	AuthorizedKeySecretName string
	// The default port to listen on (for /ssh and /metadata requests from the clients)
	DefaultPort int
	// If the default port is taken, the server will try to listen on the first free port in the DefaultPort + PortRange range
	PortRange int
}

func Run(ctx context.Context, client *databricks.WorkspaceClient, opts ServerOptions) error {
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

	createServerCommand := func(ctx context.Context) *exec.Cmd {
		return createSSHDProcess(ctx, sshdConfigPath)
	}
	connections := proxy.NewConnectionsManager(opts.MaxClients, opts.ShutdownDelay)
	http.Handle("/ssh", proxy.NewProxyServer(ctx, connections, createServerCommand))
	http.HandleFunc("/metadata", serveMetadata)
	go handleTimeout(ctx, connections.TimedOut, opts.ShutdownDelay)

	return http.ListenAndServe(listenAddr, nil)
}

func serveMetadata(w http.ResponseWriter, r *http.Request) {
	currentUser, err := user.Current()
	if err != nil {
		http.Error(w, "Failed to get current user", http.StatusInternalServerError)
		return
	}
	_, err = io.WriteString(w, currentUser.Username)
	if err != nil {
		http.Error(w, "Failed to write current user", http.StatusInternalServerError)
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
