package server

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"time"

	"github.com/databricks/cli/cmd/ssh/internal/proxy"
	"github.com/databricks/cli/cmd/ssh/internal/workspace"
	"github.com/databricks/cli/libs/env"
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
	// The cluster ID that the client started this server on (required for Driver Proxy connections)
	ClusterID string
	// SessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
	// Used for metadata storage path. Defaults to ClusterID if not set.
	SessionID string
	// Serverless indicates whether the server is running on serverless compute.
	Serverless bool
	// UsagePolicyID the job was submitted with. Persisted to metadata.json so reconnects
	// can tell which usage policy the running server was started under.
	UsagePolicyID string
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
	ctx, logBuf := captureWarnLogs(ctx)

	port, err := findAvailablePort(opts.DefaultPort, opts.PortRange)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}

	listenAddr := fmt.Sprintf("0.0.0.0:%d", port)
	log.Info(ctx, "Starting server on "+listenAddr)

	// Save metadata including ClusterID (required for Driver Proxy connections in serverless mode)
	metadata := &workspace.WorkspaceMetadata{
		Port:          port,
		ClusterID:     opts.ClusterID,
		UsagePolicyID: opts.UsagePolicyID,
	}
	err = workspace.SaveWorkspaceMetadata(ctx, client, opts.Version, opts.SessionID, metadata)
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

	// Best-effort: this only fixes bare python/pip resolution in interactive
	// sessions. The tunnel works without it (the non-interactive `-- <cmd>` path
	// is unaffected), so a write failure on a locked-down home must not abort the
	// server. Mirrors the /run/sshd handling in prepareSSHDConfig.
	if err := seedEnvActivation(ctx); err != nil {
		log.Warnf(ctx, "Failed to seed environment activation, bare python/pip may resolve to the wrong interpreter in interactive sessions: %v", err)
	}

	createServerCommand := func(ctx context.Context) *exec.Cmd {
		return createSSHDProcess(ctx, sshdConfigPath)
	}
	connections := proxy.NewConnectionsManager(opts.MaxClients, opts.ShutdownDelay)
	http.Handle("/ssh", proxy.NewProxyServer(ctx, connections, createServerCommand))
	http.HandleFunc("/metadata", serveMetadata)
	http.HandleFunc("/logs", logBuf.serveHTTP)

	http.Handle("/driver-proxy-http/ssh", proxy.NewProxyServer(ctx, connections, createServerCommand))
	http.HandleFunc("/driver-proxy-http/metadata", serveMetadata)
	http.HandleFunc("/driver-proxy-http/logs", logBuf.serveHTTP)

	go handleTimeout(ctx, connections.TimedOut, opts.ShutdownDelay)

	return http.ListenAndServe(listenAddr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Normalize double slashes from the driver proxy (e.g. //metadata -> /metadata)
		r.URL.Path = path.Clean(r.URL.Path)
		http.DefaultServeMux.ServeHTTP(w, r)
	}))
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

// envActivationMarker identifies the block seedEnvActivation appends to ~/.bashrc,
// so a server restart within the same home directory doesn't append it twice.
const envActivationMarker = "# added by databricks ssh tunnel (DECO-27499)"

// envActivationSnippet re-prepends the environment interpreter's bin directory to
// PATH for interactive SSH sessions. The client runs an interactive, non-login
// shell (bash -i), which sources /etc/bash.bashrc; on serverless that file runs
// activate_root_python_environment.sh, which prepends the cluster-libraries python
// ahead of the environment interpreter that sshd forwards via SetEnv. bash sources
// ~/.bashrc after /etc/bash.bashrc, so re-prepending here wins and bare python/pip
// resolve to $DATABRICKS_VIRTUAL_ENV. The guard makes it a no-op when the variable
// is unset. /etc/bash.bashrc and /etc/profile.d are root-owned and not writable by
// the non-root serverless user, so ~/.bashrc is the only usable hook.
//
// The bootstrap sets DATABRICKS_VIRTUAL_ENV to sys.executable, always an absolute
// path, so dirname yields the interpreter's bin directory (not "." for a bare name).
const envActivationSnippet = envActivationMarker + `
if [ -n "$DATABRICKS_VIRTUAL_ENV" ]; then
	export PATH="$(dirname "$DATABRICKS_VIRTUAL_ENV"):$PATH"
fi
`

func seedEnvActivation(ctx context.Context) error {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	bashrcPath := filepath.Join(homeDir, ".bashrc")

	existing, err := os.ReadFile(bashrcPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to read %s: %w", bashrcPath, err)
	}
	if bytes.Contains(existing, []byte(envActivationMarker)) {
		log.Info(ctx, "Environment activation already present in "+bashrcPath)
		return nil
	}

	// Append so the snippet runs after /etc/bash.bashrc and any existing ~/.bashrc
	// content; the newline guards against a file that doesn't end in one.
	separator := ""
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		separator = "\n"
	}
	content := string(existing) + separator + envActivationSnippet
	err = os.WriteFile(bashrcPath, []byte(content), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write %s: %w", bashrcPath, err)
	}

	log.Info(ctx, "Seeded environment activation in "+bashrcPath)
	return nil
}
