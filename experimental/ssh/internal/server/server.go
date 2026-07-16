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

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/internal/workspace"
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
		Port:      port,
		ClusterID: opts.ClusterID,
	}
	err = workspace.SaveWorkspaceMetadata(ctx, client, opts.Version, opts.SessionID, metadata)
	if err != nil {
		return fmt.Errorf("failed to save metadata to the workspace: %w", err)
	}

	sshdConfigPath, err := prepareSSHDConfig(ctx, client, opts)
	if err != nil {
		return fmt.Errorf("failed to setup SSH configuration: %w", err)
	}

	err = saveJupyterInitScript(ctx, os.ExpandEnv("$HOME/.ipython"))
	if err != nil {
		return fmt.Errorf("failed to save Jupyter init script: %w", err)
	}

	// Best-effort: this only fixes bare python/pip resolution in interactive
	// sessions. The tunnel works without it (the non-interactive `-- <cmd>` path
	// is unaffected), so a write failure on a locked-down home must not abort the
	// server. Mirrors the /run/sshd handling in prepareSSHDConfig.
	//
	// seedEnvActivation and saveJupyterInitScript seed the compute's OS home, which
	// the --ide path consumes (its remote server, terminal, and kernels run as the OS
	// user). seedSessionConfig seeds the workspace home's .config, which the plain
	// interactive session reads after it redirects HOME there (see buildRemoteShellArgs).
	if err := seedEnvActivation(ctx); err != nil {
		log.Warnf(ctx, "Failed to seed environment activation, bare python/pip may resolve to the wrong interpreter in interactive sessions: %v", err)
	}

	if err := seedSessionConfig(ctx, client); err != nil {
		log.Warnf(ctx, "Failed to seed interactive session config in the workspace home, bare python/pip may resolve to the wrong interpreter in interactive sessions: %v", err)
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

// saveJupyterInitScript writes the Databricks IPython startup script under
// ipythonDir (an IPYTHONDIR, i.e. the directory that contains profile_default).
func saveJupyterInitScript(ctx context.Context, ipythonDir string) error {
	ipythonStartupDir := filepath.Join(ipythonDir, "profile_default", "startup")

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
// the rcfile after /etc/bash.bashrc, so re-prepending here wins and bare python/pip
// resolve to $DATABRICKS_VIRTUAL_ENV. The guard makes it a no-op when the variable
// is unset. /etc/bash.bashrc and /etc/profile.d are root-owned and not writable by
// the non-root serverless user, so the rcfile is the only usable hook.
//
// This is appended to the OS home's ~/.bashrc (for the --ide path) and to the rcfile
// the plain interactive session loads via bash --rcfile (see seedSessionConfig and
// buildRemoteShellArgs).
//
// The bootstrap sets DATABRICKS_VIRTUAL_ENV to sys.executable, always an absolute
// path, so dirname yields the interpreter's bin directory (not "." for a bare name).
const envActivationSnippet = envActivationMarker + `
if [ -n "$DATABRICKS_VIRTUAL_ENV" ]; then
	export PATH="$(dirname "$DATABRICKS_VIRTUAL_ENV"):$PATH"
fi
`

// osHomeSourceMarker identifies the block that sources the compute's OS-home ~/.bashrc.
const osHomeSourceMarker = "# added by databricks ssh tunnel (source OS-home bashrc)"

// osHomeSourceSnippet sources the compute's OS-home ~/.bashrc from the interactive
// session, which otherwise never sees it because the session redirects HOME to the
// workspace folder (see buildRemoteShellArgs). The OS-home path is passed live in
// $DATABRICKS_OS_HOME (captured by the client before the redirect) rather than baked
// in, since the rcfile persists on the workspace mount and is reused across clusters
// where the path differs. HOME is restored to the OS home while sourcing so any
// $HOME-relative logic inside that ~/.bashrc resolves against the compute home. This is
// seeded ahead of envActivationSnippet so our PATH fixup runs last and still wins over
// anything the OS-home config prepends.
const osHomeSourceSnippet = osHomeSourceMarker + `
if [ -n "$` + workspace.OSHomeEnvVar + `" ] && [ -f "$` + workspace.OSHomeEnvVar + `/.bashrc" ]; then
	_dbx_ws_home="$HOME"; HOME="$` + workspace.OSHomeEnvVar + `"
	. "$` + workspace.OSHomeEnvVar + `/.bashrc"
	HOME="$_dbx_ws_home"; unset _dbx_ws_home
fi
`

func seedEnvActivation(ctx context.Context) error {
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	return appendGuarded(ctx, filepath.Join(homeDir, ".bashrc"), envActivationMarker, envActivationSnippet)
}

// appendGuarded appends snippet to the bashrc at bashrcPath, preserving any existing
// content. The marker makes it a no-op when the snippet is already present, so it
// never clobbers a user's edits and never duplicates the block when the server
// restarts and re-seeds an rcfile the user may have customized.
func appendGuarded(ctx context.Context, bashrcPath, marker, snippet string) error {
	existing, err := os.ReadFile(bashrcPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to read %s: %w", bashrcPath, err)
	}
	if bytes.Contains(existing, []byte(marker)) {
		log.Info(ctx, "Snippet already present in "+bashrcPath)
		return nil
	}

	// Append so the snippet runs after /etc/bash.bashrc and any existing content;
	// the newline guards against a file that doesn't end in one.
	separator := ""
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		separator = "\n"
	}
	content := string(existing) + separator + snippet
	if err := os.WriteFile(bashrcPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", bashrcPath, err)
	}

	log.Info(ctx, "Seeded snippet in "+bashrcPath)
	return nil
}

// seedSessionConfig seeds the config that a plain interactive SSH session reads once
// it redirects HOME to the user's workspace folder (see buildRemoteShellArgs). It
// writes the bash rcfile (carrying the PATH fixup) and the IPython startup script
// under HOME/.config so the workspace user-folder root stays free of dotfiles. The
// server writes to the workspace FUSE mount, so the files persist across sessions and
// are visible in the workspace file browser.
func seedSessionConfig(ctx context.Context, client *databricks.WorkspaceClient) error {
	// The seed target must match the HOME the client redirects the session to
	// (client.go buildRemoteShellArgs, resolved via its own CurrentUser.Me). Both
	// resolve to /Workspace/Users/<email> off the same identity: the server job is a
	// jobs.submit run with no run_as, so it runs as the submitting client user. If a
	// future change sets run_as to a different identity (e.g. a service principal),
	// this would seed under the server identity's home while the session HOME stays
	// the client's, and the rcfile/IPython config would never be found — seed under
	// the connecting client's home in that case.
	wsHome, err := workspace.UserWorkspaceHome(ctx, client)
	if err != nil {
		return err
	}
	return writeSessionConfig(ctx, wsHome)
}

// writeSessionConfig writes the session config files under wsHome/.config.
func writeSessionConfig(ctx context.Context, wsHome string) error {
	configDir := filepath.Join(wsHome, workspace.SessionConfigDir)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create session config directory %s: %w", configDir, err)
	}

	// This rcfile is the user's general-purpose bashrc for interactive sessions: it
	// persists on the workspace mount and users edit it. So append rather than
	// overwriting, to preserve their customizations across server restarts.
	//
	// Source the compute's OS-home ~/.bashrc first (for the per-compute config placed
	// there), then the PATH fixup last so it wins over anything that sourcing prepended.
	bashrcPath := filepath.Join(wsHome, workspace.SessionBashrc)
	if err := appendGuarded(ctx, bashrcPath, osHomeSourceMarker, osHomeSourceSnippet); err != nil {
		return err
	}
	if err := appendGuarded(ctx, bashrcPath, envActivationMarker, envActivationSnippet); err != nil {
		return err
	}

	if err := saveJupyterInitScript(ctx, filepath.Join(wsHome, workspace.SessionIPythonDir)); err != nil {
		return err
	}

	log.Info(ctx, "Seeded interactive session config in "+configDir)
	return nil
}
