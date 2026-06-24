package client

import (
	"context"
	"crypto/md5"
	_ "embed"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/keys"
	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/internal/sshconfig"
	"github.com/databricks/cli/experimental/ssh/internal/vscode"
	sshWorkspace "github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/gorilla/websocket"
)

//go:embed ssh-server-bootstrap.py
var sshServerBootstrapScript string

var errServerMetadata = errors.New("server metadata error")

var connectionNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

const (
	sshServerTaskKey         = "start_ssh_server"
	serverlessEnvironmentKey = "ssh_tunnel_serverless"
	minEnvironmentVersion    = 4
)

// acceleratorProvisioningNotice maps a GPU accelerator type to the upfront notice
// shown while its serverless compute is provisioned. Latencies vary widely by type
// (a single A10 is acquired in minutes; an 8xH100 node is ~10 min at P50 and can
// exceed 30 min at P90), so the wording is tuned per type to set expectations
// accurately. Types absent from this map fall back to a generic message.
var acceleratorProvisioningNotice = map[string]string{
	"GPU_1xA10":  "Provisioning GPU_1xA10 compute. This usually takes a few minutes and may take longer when capacity is constrained.",
	"GPU_8xH100": "Provisioning GPU_8xH100 compute. This typically takes around 10 minutes and can exceed 30 minutes when capacity is constrained.",
}

type ClientOptions struct {
	// Id of the cluster to connect to (for dedicated clusters)
	ClusterID string
	// Connection name (for serverless compute). Used as unique identifier instead of ClusterID.
	ConnectionName string
	// GPU accelerator type (for serverless compute)
	Accelerator string
	// Delay before shutting down the server after the last client disconnects
	ShutdownDelay time.Duration
	// Maximum number of SSH clients
	MaxClients int
	// Indicates that the CLI runs as a ProxyCommand - it should establish ws connection
	// to the cluster and proxy all traffic through stdin/stdout.
	// In the non proxy mode the CLI spawns an ssh client with the ProxyCommand config.
	ProxyMode bool
	// Open remote IDE window with a specific ssh config (empty, 'vscode', or 'cursor')
	IDE string
	// Expected format: "<user_name>,<port>,<cluster_id>".
	// If present, the CLI won't attempt to start the server.
	ServerMetadata string
	// How often the CLI should reconnect to the server with new auth.
	HandoverTimeout time.Duration
	// Max amount of time the server process is allowed to live
	ServerTimeout time.Duration
	// Max amount of time to wait for the SSH server task to reach RUNNING state
	TaskStartupTimeout time.Duration
	// Directory for local SSH tunnel development releases.
	// If not present, the CLI will use github releases with the current version.
	ReleasesDir string
	// Directory for local SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Client public key name located in the ssh-tunnel secrets scope.
	ClientPublicKeyName string
	// Client private key name located in the ssh-tunnel secrets scope.
	ClientPrivateKeyName string
	// If true, the CLI will attempt to start the cluster if it is not running.
	AutoStartCluster bool
	// Optional auth profile name. If present, will be added as --profile flag to the ProxyCommand while spawning ssh client.
	Profile string
	// Additional arguments to pass to the SSH client in the non proxy mode.
	AdditionalArgs []string
	// Optional path to the user known hosts file.
	UserKnownHostsFile string
	// Liteswap header value for traffic routing (dev/test only).
	Liteswap string
	// If true, skip checking and updating IDE settings.
	SkipSettingsCheck bool
	// Environment version for serverless compute.
	EnvironmentVersion int
	// If true, skip confirmation prompts for IDE extension install and IDE settings updates.
	AutoApprove bool
}

func (o *ClientOptions) Validate() error {
	if !o.ProxyMode && o.ClusterID == "" && o.ConnectionName == "" {
		return errors.New("please provide --cluster flag with the cluster ID, or --name flag with the connection name (for serverless compute)")
	}
	if o.Accelerator != "" && o.ConnectionName == "" {
		return errors.New("--accelerator flag can only be used with serverless compute (--name flag)")
	}
	if o.Accelerator != "" && o.Accelerator != "GPU_1xA10" && o.Accelerator != "GPU_8xH100" {
		return fmt.Errorf("invalid accelerator value: %q, expected %q or %q", o.Accelerator, "GPU_1xA10", "GPU_8xH100")
	}
	if o.ConnectionName != "" && !connectionNameRegex.MatchString(o.ConnectionName) {
		return fmt.Errorf("connection name %q must consist of letters, numbers, dashes, and underscores", o.ConnectionName)
	}
	if o.IDE != "" && o.IDE != vscode.VSCodeOption && o.IDE != vscode.CursorOption {
		return fmt.Errorf("invalid IDE value: %q, expected %q or %q", o.IDE, vscode.VSCodeOption, vscode.CursorOption)
	}
	if o.EnvironmentVersion > 0 && o.EnvironmentVersion < minEnvironmentVersion {
		return fmt.Errorf("environment version must be >= %d, got %d", minEnvironmentVersion, o.EnvironmentVersion)
	}
	return nil
}

// GenerateDefaultConnectionName creates a deterministic connection name from
// the workspace host and accelerator type. The name includes a hash of the
// workspace host so that different workspaces produce different names,
// avoiding SSH known_hosts conflicts.
func GenerateDefaultConnectionName(host, accelerator string) string {
	h := md5.Sum([]byte(host))
	hashStr := hex.EncodeToString(h[:4])
	if accelerator != "" {
		acc := strings.ToLower(strings.ReplaceAll(accelerator, "_", "-"))
		return fmt.Sprintf("databricks-%s-%s", acc, hashStr)
	}
	return "databricks-cpu-" + hashStr
}

func (o *ClientOptions) IsServerlessMode() bool {
	return o.ClusterID == "" && o.ConnectionName != ""
}

// SessionIdentifier returns the unique identifier for the session.
// For dedicated clusters, this is the cluster ID. For serverless, this is the connection name.
func (o *ClientOptions) SessionIdentifier() string {
	if o.IsServerlessMode() {
		return o.ConnectionName
	}
	return o.ClusterID
}

// FormatMetadata formats the server metadata string for use in ProxyCommand.
// Returns empty string if userName is empty or serverPort is zero.
func FormatMetadata(userName string, serverPort int, clusterID string) string {
	if userName == "" || serverPort == 0 {
		return ""
	}
	if clusterID != "" {
		return fmt.Sprintf("%s,%d,%s", userName, serverPort, clusterID)
	}
	return fmt.Sprintf("%s,%d", userName, serverPort)
}

// ToProxyCommand generates the ProxyCommand string for SSH config.
// This method serializes the ClientOptions into a command-line invocation that will
// be parsed back into ClientOptions when the SSH ProxyCommand is executed.
func (o *ClientOptions) ToProxyCommand() (string, error) {
	executablePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get current executable path: %w", err)
	}

	var proxyCommand string
	if o.IsServerlessMode() {
		proxyCommand = fmt.Sprintf("%q ssh connect --proxy --name=%s --shutdown-delay=%s",
			executablePath, o.ConnectionName, o.ShutdownDelay.String())
		if o.Accelerator != "" {
			proxyCommand += " --accelerator=" + o.Accelerator
		}
	} else {
		proxyCommand = fmt.Sprintf("%q ssh connect --proxy --cluster=%s --auto-start-cluster=%t --shutdown-delay=%s",
			executablePath, o.ClusterID, o.AutoStartCluster, o.ShutdownDelay.String())
	}

	if o.ServerMetadata != "" {
		proxyCommand += " --metadata=" + o.ServerMetadata
	}

	if o.HandoverTimeout > 0 {
		proxyCommand += " --handover-timeout=" + o.HandoverTimeout.String()
	}

	if o.Profile != "" {
		proxyCommand += " --profile=" + o.Profile
	}

	if o.Liteswap != "" {
		proxyCommand += " --liteswap=" + o.Liteswap
	}

	if o.EnvironmentVersion > 0 {
		proxyCommand += " --environment-version=" + strconv.Itoa(o.EnvironmentVersion)
	}

	return proxyCommand, nil
}

func Run(ctx context.Context, client *databricks.WorkspaceClient, opts ClientOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		<-sigCh
		cmdio.LogString(ctx, "Received termination signal, cleaning up...")
		cancel()
	}()

	sessionID := opts.SessionIdentifier()
	if sessionID == "" {
		return errors.New("either --cluster or --name must be provided")
	}

	if !opts.ProxyMode {
		cmdio.LogString(ctx, fmt.Sprintf("Connecting to %s...", sessionID))
	}

	if opts.IDE != "" && !opts.ProxyMode {
		if err := vscode.CheckIDECommand(opts.IDE); err != nil {
			return err
		}
		if err := vscode.CheckIDESSHExtension(ctx, opts.IDE, opts.AutoApprove); err != nil {
			return err
		}
	}

	// Check and update IDE settings for serverless mode, where we must set up
	// desired server ports (or socket connection mode) for the connection to go through
	// (as the majority of the localhost ports on the remote side are blocked by iptable rules).
	// Plus the platform (always linux), and extensions (python and jupyter), to make the initial experience smoother.
	if opts.IDE != "" && opts.IsServerlessMode() && !opts.ProxyMode && !opts.SkipSettingsCheck {
		err := vscode.CheckAndUpdateSettings(ctx, opts.IDE, opts.ConnectionName, opts.AutoApprove)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Failed to update IDE settings: %v", err))
			cmdio.LogString(ctx, vscode.GetManualInstructions(opts.IDE, opts.ConnectionName))
			cmdio.LogString(ctx, "Use --skip-settings-check to bypass IDE settings verification.")
			if opts.AutoApprove {
				return fmt.Errorf("aborted: IDE settings need to be updated manually: %w", err)
			}
			shouldProceed, promptErr := cmdio.AskYesOrNo(ctx, "Do you want to proceed with the connection?")
			if promptErr != nil {
				return fmt.Errorf("failed to prompt user: %w", promptErr)
			}
			if !shouldProceed {
				return errors.New("aborted: IDE settings need to be updated manually, user declined to proceed")
			}
		}
	}

	isReconnect := opts.ServerMetadata != ""
	var serverStartTimeMs int64
	isSuccess := false
	defer func() {
		logSshTunnelEvent(ctx, opts, isSuccess, isReconnect, serverStartTimeMs)
	}()

	// Only check cluster state for dedicated clusters
	if !opts.IsServerlessMode() {
		cmdio.LogString(ctx, "Checking cluster state...")
		err := checkClusterState(ctx, client, opts.ClusterID, opts.AutoStartCluster)
		if err != nil {
			return err
		}
	}

	secretScopeName, err := keys.CreateKeysSecretScope(ctx, client, sessionID)
	if err != nil {
		return fmt.Errorf("failed to create secret scope: %w", err)
	}

	privateKeyBytes, publicKeyBytes, err := keys.CheckAndGenerateSSHKeyPairFromSecrets(ctx, client, secretScopeName, opts.ClientPrivateKeyName, opts.ClientPublicKeyName)
	if err != nil {
		return fmt.Errorf("failed to get or generate SSH key pair from secrets: %w", err)
	}

	keyPath, err := keys.GetLocalSSHKeyPath(ctx, sessionID, opts.SSHKeysDir)
	if err != nil {
		return fmt.Errorf("failed to get local keys folder: %w", err)
	}

	err = keys.SaveSSHKeyPair(keyPath, privateKeyBytes, publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to save SSH key pair locally: %w", err)
	}
	log.Infof(ctx, "Using SSH key: %s", keyPath)
	log.Infof(ctx, "Secrets scope: %s, key name: %s", secretScopeName, opts.ClientPublicKeyName)

	var userName string
	var serverPort int
	var clusterID string

	version := build.GetInfo().Version

	if opts.ServerMetadata == "" {
		cmdio.LogString(ctx, "Uploading binaries...")
		sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
		sp.Update("Uploading binaries...")
		err := UploadTunnelReleases(ctx, client, version, opts.ReleasesDir)
		sp.Close()
		if err != nil {
			return fmt.Errorf("failed to upload ssh-tunnel binaries: %w", err)
		}
		serverStartTime := time.Now()
		userName, serverPort, clusterID, err = ensureSSHServerIsRunning(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return fmt.Errorf("failed to ensure that ssh server is running: %w", err)
		}
		serverStartTimeMs = time.Since(serverStartTime).Milliseconds()
	} else {
		// Metadata format: "<user_name>,<port>,<cluster_id>"
		metadata := strings.Split(opts.ServerMetadata, ",")
		if len(metadata) < 2 {
			return fmt.Errorf("invalid metadata: %s, expected format: <user_name>,<port>[,<cluster_id>]", opts.ServerMetadata)
		}
		userName = metadata[0]
		if userName == "" {
			return fmt.Errorf("remote user name is empty in the metadata: %s", opts.ServerMetadata)
		}
		serverPort, err = strconv.Atoi(metadata[1])
		if err != nil {
			return fmt.Errorf("cannot parse port from metadata: %s, %w", opts.ServerMetadata, err)
		}
		if len(metadata) >= 3 {
			clusterID = metadata[2]
		} else {
			clusterID = opts.ClusterID
		}
	}

	// For serverless mode, we need the cluster ID from metadata for Driver Proxy connections
	if opts.IsServerlessMode() && clusterID == "" {
		return errors.New("cluster ID is required for serverless connections but was not found in metadata")
	}

	log.Infof(ctx, "Remote user name: %s", userName)
	log.Infof(ctx, "Server port: %d", serverPort)
	if opts.IsServerlessMode() {
		log.Infof(ctx, "Cluster ID (from serverless job): %s", clusterID)
	}

	if !opts.ProxyMode {
		cmdio.LogString(ctx, "Connected!")
	}

	isSuccess = true

	if opts.ProxyMode {
		return runSSHProxy(ctx, client, serverPort, clusterID, opts)
	} else if opts.IDE != "" {
		return runIDE(ctx, client, userName, keyPath, serverPort, clusterID, opts)
	} else {
		log.Infof(ctx, "Additional SSH arguments: %v", opts.AdditionalArgs)
		return spawnSSHClient(ctx, client, userName, keyPath, serverPort, clusterID, opts)
	}
}

func runIDE(ctx context.Context, client *databricks.WorkspaceClient, userName, keyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	connectionName := opts.SessionIdentifier()
	if connectionName == "" {
		return errors.New("connection name is required for IDE integration")
	}

	// Get Databricks user name for the workspace path
	currentUser, err := client.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// Ensure SSH config entry exists
	configPath, err := sshconfig.GetMainConfigPath(ctx)
	if err != nil {
		return fmt.Errorf("failed to get SSH config path: %w", err)
	}

	err = ensureSSHConfigEntry(ctx, configPath, connectionName, userName, keyPath, serverPort, clusterID, opts)
	if err != nil {
		return fmt.Errorf("failed to ensure SSH config entry: %w", err)
	}

	return vscode.LaunchIDE(ctx, opts.IDE, connectionName, userName, currentUser.UserName)
}

func ensureSSHConfigEntry(ctx context.Context, configPath, hostName, userName, keyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	// Ensure the Include directive exists in the main SSH config
	err := sshconfig.EnsureIncludeDirective(ctx, configPath)
	if err != nil {
		return err
	}

	// Generate ProxyCommand with server metadata
	optsWithMetadata := opts
	optsWithMetadata.ServerMetadata = FormatMetadata(userName, serverPort, clusterID)

	proxyCommand, err := optsWithMetadata.ToProxyCommand()
	if err != nil {
		return fmt.Errorf("failed to generate ProxyCommand: %w", err)
	}

	hostConfig := sshconfig.GenerateHostConfig(hostName, userName, keyPath, proxyCommand)

	_, err = sshconfig.CreateOrUpdateHostConfig(ctx, hostName, hostConfig, true)
	if err != nil {
		return err
	}

	log.Infof(ctx, "Updated SSH config entry for '%s'", hostName)
	return nil
}

// getServerMetadata retrieves the server metadata from the workspace and validates it via Driver Proxy.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
// For dedicated clusters, clusterID should be the same as sessionID.
// For serverless, clusterID is read from the workspace metadata.
func getServerMetadata(ctx context.Context, client *databricks.WorkspaceClient, sessionID, clusterID, version, liteswap string) (int, string, string, error) {
	wsMetadata, err := sshWorkspace.GetWorkspaceMetadata(ctx, client, version, sessionID)
	if err != nil {
		return 0, "", "", errors.Join(errServerMetadata, err)
	}
	log.Debugf(ctx, "Workspace metadata: %+v", wsMetadata)

	// For serverless mode, the cluster ID comes from the metadata
	effectiveClusterID := clusterID
	if wsMetadata.ClusterID != "" {
		effectiveClusterID = wsMetadata.ClusterID
	}

	if effectiveClusterID == "" {
		return 0, "", "", errors.Join(errServerMetadata, errors.New("cluster ID not available in metadata"))
	}

	req, err := newDriverProxyRequest(ctx, client, effectiveClusterID, wsMetadata.Port, "metadata", liteswap)
	if err != nil {
		return 0, "", "", err
	}
	log.Debugf(ctx, "Metadata URL: %s", req.URL)
	httpClient := &http.Client{Transport: client.Config.HTTPTransport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, "", "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", "", err
	}
	log.Debugf(ctx, "Metadata response: %s", string(bodyBytes))
	log.Debugf(ctx, "Metadata response status code: %d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return 0, "", "", errors.Join(errServerMetadata, fmt.Errorf("server is not ok, status code %d", resp.StatusCode))
	}

	return wsMetadata.Port, string(bodyBytes), effectiveClusterID, nil
}

// newDriverProxyRequest builds an authenticated GET request to one of the SSH server's
// HTTP endpoints behind the workspace driver proxy.
func newDriverProxyRequest(ctx context.Context, client *databricks.WorkspaceClient, clusterID string, port int, endpoint, liteswap string) (*http.Request, error) {
	workspaceID, err := auth.ResolveWorkspaceID(ctx, client)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/driver-proxy-api/o/%s/%s/%d/%s", client.Config.Host, workspaceID, clusterID, port, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if liteswap != "" {
		req.Header.Set("x-databricks-traffic-id", "testenv://liteswap/"+liteswap)
	}
	if err := client.Config.Authenticate(req); err != nil {
		return nil, err
	}
	return req, nil
}

// fetchServerErrorLogs fetches recent warning/error log lines from the running SSH
// server's /logs endpoint. It is best-effort: any failure (including older server
// versions that don't serve /logs) yields an empty string.
func fetchServerErrorLogs(ctx context.Context, client *databricks.WorkspaceClient, clusterID string, serverPort int, liteswap string) string {
	req, err := newDriverProxyRequest(ctx, client, clusterID, serverPort, "logs", liteswap)
	if err != nil {
		log.Debugf(ctx, "Failed to build server logs request: %v", err)
		return ""
	}
	httpClient := &http.Client{Transport: client.Config.HTTPTransport}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Debugf(ctx, "Failed to fetch server logs: %v", err)
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Debugf(ctx, "Server logs endpoint returned status %d", resp.StatusCode)
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debugf(ctx, "Failed to read server logs response: %v", err)
		return ""
	}
	return strings.TrimSpace(string(body))
}

// submitSSHTunnelJob submits the bootstrap job and waits for the SSH server task to start.
// It returns the job run ID (when known) so callers can fetch and surface the run's error
// details if the server never comes up.
func submitSSHTunnelJob(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) (int64, error) {
	sessionID := opts.SessionIdentifier()
	contentDir, err := sshWorkspace.GetWorkspaceContentDir(ctx, client, version, sessionID)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	err = client.Workspace.MkdirsByPath(ctx, contentDir)
	if err != nil {
		return 0, fmt.Errorf("failed to create directory in the remote workspace: %w", err)
	}

	sshTunnelJobName := "ssh-server-bootstrap-" + sessionID
	jobNotebookPath := filepath.ToSlash(filepath.Join(contentDir, "ssh-server-bootstrap"))
	notebookContent := "# Databricks notebook source\n" + sshServerBootstrapScript
	encodedContent := base64.StdEncoding.EncodeToString([]byte(notebookContent))

	err = client.Workspace.Import(ctx, workspace.Import{
		Path:      jobNotebookPath,
		Format:    workspace.ImportFormatSource,
		Content:   encodedContent,
		Language:  workspace.LanguagePython,
		Overwrite: true,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to create ssh-tunnel notebook: %w", err)
	}

	baseParams := map[string]string{
		"version":                 version,
		"secretScopeName":         secretScopeName,
		"authorizedKeySecretName": opts.ClientPublicKeyName,
		"shutdownDelay":           opts.ShutdownDelay.String(),
		"maxClients":              strconv.Itoa(opts.MaxClients),
		"sessionId":               sessionID,
		"serverless":              strconv.FormatBool(opts.IsServerlessMode()),
	}

	log.Infof(ctx, "Submitting a job to start the ssh server...")

	task := jobs.SubmitTask{
		TaskKey: sshServerTaskKey,
		NotebookTask: &jobs.NotebookTask{
			NotebookPath:   jobNotebookPath,
			BaseParameters: baseParams,
		},
		TimeoutSeconds: int(opts.ServerTimeout.Seconds()),
	}

	if opts.IsServerlessMode() {
		task.EnvironmentKey = serverlessEnvironmentKey
		if opts.Accelerator != "" {
			log.Infof(ctx, "Using accelerator: %s", opts.Accelerator)
			task.Compute = &jobs.Compute{
				HardwareAccelerator: compute.HardwareAcceleratorType(opts.Accelerator),
			}
		}
	} else {
		task.ExistingClusterId = opts.ClusterID
	}

	submitRequest := jobs.SubmitRun{
		RunName:        sshTunnelJobName,
		TimeoutSeconds: int(opts.ServerTimeout.Seconds()),
		Tasks:          []jobs.SubmitTask{task},
	}

	if opts.IsServerlessMode() {
		submitRequest.Environments = []jobs.JobEnvironment{
			{
				EnvironmentKey: serverlessEnvironmentKey,
				Spec: &compute.Environment{
					EnvironmentVersion: strconv.Itoa(max(opts.EnvironmentVersion, minEnvironmentVersion)),
				},
			},
		}
	}

	waiter, err := client.Jobs.Submit(ctx, submitRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to submit job: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Job submitted successfully with run ID: %d", waiter.RunId))

	// Return the run ID even on error so callers can fetch the run's failure details.
	return waiter.RunId, waitForJobToStart(ctx, client, waiter.RunId, opts)
}

// shellSingleQuote wraps s in single quotes for safe inclusion in a shell
// command, escaping any embedded single quotes.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// buildRemoteShellArgs returns the ssh arguments that follow the hostname.
//
// For the interactive case (no remote command given), it forces PTY allocation
// and launches a login bash, because the default login shell on Databricks
// compute images is /bin/sh. If bash is unavailable it falls back to $SHELL or
// /bin/sh so the connection never breaks. When wsHome is set, the shell first
// changes into the user's workspace home folder; if that directory is missing
// the cd is ignored and the shell still launches from $HOME.
//
// For the non-interactive case (e.g. `databricks ssh connect ... -- ls -la`),
// the user's command is returned verbatim so behavior is unchanged.
//
// Note: this returns the remote command only. PTY allocation (-t) is added to
// the ssh options *before* the destination by the caller; -t placed after the
// host would be parsed as part of the remote command, not as ssh's flag.
func buildRemoteShellArgs(opts ClientOptions, wsHome string) []string {
	if len(opts.AdditionalArgs) > 0 {
		return opts.AdditionalArgs
	}
	cmd := `command -v bash >/dev/null 2>&1 && exec bash -l || exec "${SHELL:-/bin/sh}" -l`
	if wsHome != "" {
		cmd = "cd " + shellSingleQuote(wsHome) + " 2>/dev/null; " + cmd
	}
	return []string{cmd}
}

// buildSSHArgs assembles the argument list for the ssh client. Options come
// first, then the destination host, then the remote command (if any). PTY
// allocation (-t) for the interactive case is added before the host: ssh stops
// parsing options at the destination, so a -t placed after the host would be
// treated as part of the remote command rather than as ssh's force-PTY flag.
func buildSSHArgs(userName, privateKeyPath, proxyCommand, hostName, wsHome string, opts ClientOptions) []string {
	sshArgs := []string{
		"-l", userName,
		"-i", privateKeyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=360",
		"-o", "ProxyCommand=" + proxyCommand,
	}
	if opts.UserKnownHostsFile != "" {
		sshArgs = append(sshArgs, "-o", "UserKnownHostsFile="+opts.UserKnownHostsFile)
	}
	if len(opts.AdditionalArgs) == 0 {
		sshArgs = append(sshArgs, "-t")
	}
	sshArgs = append(sshArgs, hostName)
	sshArgs = append(sshArgs, buildRemoteShellArgs(opts, wsHome)...)
	return sshArgs
}

func spawnSSHClient(ctx context.Context, client *databricks.WorkspaceClient, userName, privateKeyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	// Create a copy with metadata for the ProxyCommand
	optsWithMetadata := opts
	optsWithMetadata.ServerMetadata = FormatMetadata(userName, serverPort, clusterID)

	proxyCommand, err := optsWithMetadata.ToProxyCommand()
	if err != nil {
		return fmt.Errorf("failed to generate ProxyCommand: %w", err)
	}

	hostName := opts.SessionIdentifier()

	// For an interactive session (no remote command supplied), land the shell in
	// the user's workspace home folder (/Workspace/Users/<email>) instead of the
	// OS home. Only needed for an interactive session; skip the lookup otherwise.
	var wsHome string
	if len(opts.AdditionalArgs) == 0 {
		if currentUser, err := client.CurrentUser.Me(ctx, iam.MeRequest{}); err != nil {
			log.Warnf(ctx, "Failed to resolve current user for workspace home directory: %v", err)
		} else {
			wsHome = "/Workspace/Users/" + currentUser.UserName
		}
	}

	sshArgs := buildSSHArgs(userName, privateKeyPath, proxyCommand, hostName, wsHome, opts)

	log.Debugf(ctx, "Launching SSH client: ssh %s", strings.Join(sshArgs, " "))
	sshCmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	// Tee ssh's stderr so the user still sees it while we retain the tail to inspect after exit.
	// A host-key-verification failure is reported only on stderr, so we need a copy to detect it.
	stderrTail := &tailWriter{maxBytes: hostKeyStderrTailBytes}
	sshCmd.Stderr = io.MultiWriter(os.Stderr, stderrTail)

	err = sshCmd.Run()
	// ssh reserves exit code 255 for its own connection-level failures (a remote command's exit
	// code is passed through as-is, 0-254). The server keeps running after a failed connection
	// attempt, so its error (e.g. sshd missing from the container image) is only visible in its
	// own logs — fetch them from the /logs endpoint and show them instead of leaving the user
	// with ssh's opaque "Connection closed" message.
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 255 {
		if hint := hostKeyChangedHint(stderrTail.String(), hostName, opts.UserKnownHostsFile); hint != "" {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, hint))
		} else if logs := fetchServerErrorLogs(ctx, client, clusterID, serverPort, opts.Liteswap); logs != "" {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, "The SSH connection closed unexpectedly. Recent SSH server errors:"))
			cmdio.LogString(ctx, truncateTail(logs, maxRunFailureTraceBytes))
		} else {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, "The SSH connection closed unexpectedly. If it dropped right after connecting, "+
				"the cluster's container image is likely missing an OpenSSH server: ensure 'openssh-server' "+
				"is installed (it provides /usr/sbin/sshd), then check the SSH server job run logs."))
		}
	}
	return err
}

func runSSHProxy(ctx context.Context, client *databricks.WorkspaceClient, serverPort int, clusterID string, opts ClientOptions) error {
	createConn := func(ctx context.Context, connID string) (*websocket.Conn, error) {
		return createWebsocketConnection(ctx, client, connID, clusterID, serverPort, opts.Liteswap)
	}
	requestHandoverTick := func() <-chan time.Time {
		return time.After(opts.HandoverTimeout)
	}
	return proxy.RunClientProxy(ctx, os.Stdin, os.Stdout, requestHandoverTick, createConn)
}

func checkClusterState(ctx context.Context, client *databricks.WorkspaceClient, clusterID string, autoStart bool) error {
	sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
	defer sp.Close()
	if autoStart {
		sp.Update("Waiting for compute to start...")
		err := client.Clusters.EnsureClusterIsRunning(ctx, clusterID)
		if err != nil {
			return fmt.Errorf("failed to ensure that the cluster is running: %w", err)
		}
	} else {
		sp.Update("Checking cluster state...")
		cluster, err := client.Clusters.GetByClusterId(ctx, clusterID)
		if err != nil {
			return fmt.Errorf("failed to get cluster info: %w", err)
		}
		if cluster.State != compute.StateRunning {
			return fmt.Errorf("cluster %s is not running, current state: %s. Use --auto-start-cluster to start it automatically", clusterID, cluster.State)
		}
	}
	return nil
}

// waitForJobToStart polls the task status until the SSH server task is in RUNNING state or terminates.
// Returns an error if the task fails to start or if polling times out.
func waitForJobToStart(ctx context.Context, client *databricks.WorkspaceClient, runID int64, opts ClientOptions) error {
	waitingMessage := "Waiting for compute to start..."
	if opts.Accelerator != "" {
		// GPU capacity is acquired on demand and the wait varies a lot by accelerator
		// type; without this notice users assume a long PENDING wait means the service
		// is down. Latencies differ enough between types that a single message would be
		// misleading, so phrase the heads-up per accelerator with a generic fallback.
		notice, ok := acceleratorProvisioningNotice[opts.Accelerator]
		if !ok {
			notice = fmt.Sprintf("Provisioning %s compute. This can take several minutes and may take longer when capacity is constrained.", opts.Accelerator)
		}
		cmdio.LogString(ctx, notice)
		waitingMessage = fmt.Sprintf("Provisioning %s compute...", opts.Accelerator)
	}

	sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
	defer sp.Close()
	sp.Update(waitingMessage)
	var prevState jobs.RunLifecycleStateV2State

	_, err := retries.Poll(ctx, opts.TaskStartupTimeout, func() (*jobs.RunTask, *retries.Err) {
		run, err := client.Jobs.GetRun(ctx, jobs.GetRunRequest{
			RunId: runID,
		})
		if err != nil {
			return nil, retries.Halt(fmt.Errorf("failed to get job run status: %w", err))
		}

		// Find the SSH server task
		var sshTask *jobs.RunTask
		for i := range run.Tasks {
			if run.Tasks[i].TaskKey == sshServerTaskKey {
				sshTask = &run.Tasks[i]
				break
			}
		}

		if sshTask == nil {
			return nil, retries.Halt(fmt.Errorf("SSH server task '%s' not found in job run", sshServerTaskKey))
		}

		if sshTask.Status == nil {
			return nil, retries.Halt(errors.New("task status is nil"))
		}

		currentState := sshTask.Status.State

		// Update spinner if state changed
		if currentState != prevState {
			sp.Update(fmt.Sprintf("%s (task: %s)", waitingMessage, currentState))
			prevState = currentState
		}

		// Check if task is running
		if currentState == jobs.RunLifecycleStateV2StateRunning {
			return sshTask, nil
		}

		// Check for terminal failure states. Surface the run's actual error (e.g. a notebook
		// traceback or "Could not reach driver") instead of a generic message.
		if currentState == jobs.RunLifecycleStateV2StateTerminated {
			return nil, retries.Halt(fmt.Errorf("ssh server bootstrap job failed:\n%s", describeRunFailure(ctx, client, runID)))
		}

		// Continue polling for other states
		return nil, retries.Continues(fmt.Sprintf("waiting for task to start (current state: %s)", currentState))
	})

	return err
}

// maxRunFailureTraceBytes bounds how much of a failed run's error trace we print to the
// terminal; the full output is always available via the run page URL.
const maxRunFailureTraceBytes = 2000

// describeRunFailure fetches a failed bootstrap run's error details and formats them for the
// terminal. It is best-effort: any API error is folded into the returned text rather than
// propagated, so callers can always embed the result in their own error.
func describeRunFailure(ctx context.Context, client *databricks.WorkspaceClient, runID int64) string {
	if runID == 0 {
		return "  (no job run ID available)"
	}

	run, err := client.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return fmt.Sprintf("  could not fetch job run %d: %v", runID, err)
	}

	var b strings.Builder

	// Locate the SSH server task to read its termination reason and per-task run output.
	var sshTask *jobs.RunTask
	for i := range run.Tasks {
		if run.Tasks[i].TaskKey == sshServerTaskKey {
			sshTask = &run.Tasks[i]
			break
		}
	}

	if sshTask != nil && sshTask.Status != nil && sshTask.Status.TerminationDetails != nil {
		if msg := strings.TrimSpace(sshTask.Status.TerminationDetails.Message); msg != "" {
			fmt.Fprintf(&b, "  %s\n", msg)
		}
	}

	// The notebook error/traceback carries the real cause (e.g. a Python exception).
	outputRunID := runID
	if sshTask != nil && sshTask.RunId != 0 {
		outputRunID = sshTask.RunId
	}
	if output, err := client.Jobs.GetRunOutput(ctx, jobs.GetRunOutputRequest{RunId: outputRunID}); err == nil && output != nil {
		e := strings.TrimSpace(output.Error)
		trace := strings.TrimSpace(output.ErrorTrace)
		// Notebook tracebacks end with the same message as Error; skip Error then so the
		// server-log tail the bootstrap embeds in the message isn't printed twice.
		if e != "" && !strings.Contains(trace, e) {
			fmt.Fprintf(&b, "  %s\n", truncateTail(e, maxRunFailureTraceBytes))
		}
		if trace != "" {
			fmt.Fprintf(&b, "%s\n", truncateTail(trace, maxRunFailureTraceBytes))
		}
	}

	if run.RunPageUrl != "" {
		fmt.Fprintf(&b, "  See the full job logs: %s", run.RunPageUrl)
	}

	if b.Len() == 0 {
		return fmt.Sprintf("  job run %d failed; see run details in the workspace", runID)
	}
	return strings.TrimRight(b.String(), "\n")
}

// runFailureIfTerminated reports whether the bootstrap run has reached a terminal state (so the
// SSH server will never come up), returning a formatted failure description when it has.
func runFailureIfTerminated(ctx context.Context, client *databricks.WorkspaceClient, runID int64) (string, bool) {
	if runID == 0 {
		return "", false
	}
	run, err := client.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
	if err != nil {
		return "", false
	}
	for i := range run.Tasks {
		if run.Tasks[i].TaskKey != sshServerTaskKey {
			continue
		}
		if run.Tasks[i].Status != nil && run.Tasks[i].Status.State == jobs.RunLifecycleStateV2StateTerminated {
			return describeRunFailure(ctx, client, runID), true
		}
		return "", false
	}
	return "", false
}

// truncateTail returns the last maxBytes of s, marking the cut when truncated.
func truncateTail(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	return "  ...\n" + s[len(s)-maxBytes:]
}

// hostKeyStderrTailBytes bounds how much of ssh's stderr we retain to detect a host-key failure.
// The host-key warning block ssh prints is well under this, so the tail always captures it.
const hostKeyStderrTailBytes = 4096

// tailWriter retains the last maxBytes written to it, so we can inspect an external command's
// recent stderr without buffering an unbounded amount.
type tailWriter struct {
	maxBytes int
	buf      []byte
}

func (w *tailWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	if len(w.buf) > w.maxBytes {
		w.buf = w.buf[len(w.buf)-w.maxBytes:]
	}
	return len(p), nil
}

func (w *tailWriter) String() string {
	return string(w.buf)
}

// hostKeyChangedHint returns advice for clearing a stale known_hosts entry when ssh's stderr
// shows a host-key-verification failure, or "" if the failure was something else. A cluster that
// has been recreated keeps the same connection name but gets a new host key, so the old entry no
// longer matches and ssh aborts the connection.
func hostKeyChangedHint(stderr, hostName, knownHostsFile string) string {
	// "Host key verification failed." is OpenSSH's fixed message for this case; matching it is the
	// only signal ssh gives (the "don't branch on err.Error()" rule is about Go errors, not the
	// output of an external program).
	if !strings.Contains(stderr, "Host key verification failed") {
		return ""
	}
	cmd := "ssh-keygen -R " + hostName
	if knownHostsFile != "" {
		// ssh-keygen -R defaults to ~/.ssh/known_hosts, so name the custom file explicitly.
		cmd += " -f " + knownHostsFile
	}
	return "The host key for " + hostName + " has changed. " +
		"Remove the stale entry and reconnect:\n  " + cmd
}

func ensureSSHServerIsRunning(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) (string, int, string, error) {
	sessionID := opts.SessionIdentifier()
	// For dedicated clusters, use clusterID; for serverless, it will be read from metadata
	clusterID := opts.ClusterID

	serverPort, userName, effectiveClusterID, err := getServerMetadata(ctx, client, sessionID, clusterID, version, opts.Liteswap)
	if errors.Is(err, errServerMetadata) {
		cmdio.LogString(ctx, "Starting SSH server...")

		runID, err := submitSSHTunnelJob(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return "", 0, "", fmt.Errorf("failed to submit and start ssh server job: %w", err)
		}

		sp := cmdio.NewSpinner(ctx, cmdio.WithElapsedTime())
		defer sp.Close()
		sp.Update("Waiting for the SSH server to start...")
		maxRetries := 30
		for retries := range maxRetries {
			if ctx.Err() != nil {
				return "", 0, "", ctx.Err()
			}
			serverPort, userName, effectiveClusterID, err = getServerMetadata(ctx, client, sessionID, clusterID, version, opts.Liteswap)
			if err == nil {
				cmdio.LogString(ctx, "Health check successful, starting ssh WebSocket connection...")
				break
			}
			// The metadata never appears if the bootstrap job dies after reaching RUNNING.
			// Surface the job's actual error instead of waiting out the full timeout with a
			// generic "metadata.json doesn't exist" message.
			if failure, terminated := runFailureIfTerminated(ctx, client, runID); terminated {
				return "", 0, "", fmt.Errorf("ssh server bootstrap job failed:\n%s", failure)
			}
			if retries < maxRetries-1 {
				time.Sleep(2 * time.Second)
			} else {
				return "", 0, "", fmt.Errorf("failed to start the ssh server: %w\n%s", err, describeRunFailure(ctx, client, runID))
			}
		}
	} else if err != nil {
		return "", 0, "", err
	}

	return userName, serverPort, effectiveClusterID, nil
}

func logSshTunnelEvent(ctx context.Context, opts ClientOptions, isSuccess, isReconnect bool, serverStartTimeMs int64) {
	computeType := protos.SshTunnelComputeTypeDedicated
	if opts.IsServerlessMode() {
		computeType = protos.SshTunnelComputeTypeServerless
	}

	var clientMode protos.SshTunnelClientMode
	switch {
	case opts.ProxyMode:
		clientMode = protos.SshTunnelClientModeProxy
	case opts.IDE != "":
		clientMode = protos.SshTunnelClientModeIDE
	default:
		clientMode = protos.SshTunnelClientModeSSH
	}

	telemetry.Log(ctx, protos.DatabricksCliLog{
		SshTunnelEvent: &protos.SshTunnelEvent{
			ComputeType:       computeType,
			AcceleratorType:   opts.Accelerator,
			IdeType:           opts.IDE,
			ClientMode:        clientMode,
			IsReconnect:       isReconnect,
			AutoStartCluster:  opts.AutoStartCluster,
			ServerStartTimeMs: serverStartTimeMs,
			IsSuccess:         isSuccess,
		},
	})
}
