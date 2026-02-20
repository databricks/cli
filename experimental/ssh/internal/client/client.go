package client

import (
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/gorilla/websocket"
)

//go:embed ssh-server-bootstrap.py
var sshServerBootstrapScript string

var errServerMetadata = errors.New("server metadata error")

const (
	sshServerTaskKey         = "start_ssh_server"
	serverlessEnvironmentKey = "ssh_tunnel_serverless"

	VSCodeOption  = "vscode"
	VSCodeCommand = "code"
	CursorOption  = "cursor"
	CursorCommand = "cursor"
)

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

	// Only check cluster state for dedicated clusters
	if !opts.IsServerlessMode() {
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

	keyPath, err := keys.GetLocalSSHKeyPath(sessionID, opts.SSHKeysDir)
	if err != nil {
		return fmt.Errorf("failed to get local keys folder: %w", err)
	}

	err = keys.SaveSSHKeyPair(keyPath, privateKeyBytes, publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to save SSH key pair locally: %w", err)
	}
	cmdio.LogString(ctx, "Using SSH key: "+keyPath)
	cmdio.LogString(ctx, fmt.Sprintf("Secrets scope: %s, key name: %s", secretScopeName, opts.ClientPublicKeyName))

	// Check and update IDE settings for serverless mode, where we must set up
	// desired server ports (or socket connection mode) for the connection to go through
	// (as the majority of the localhost ports on the remote side are blocked by iptable rules).
	// Plus the platform (always linux), and extensions (python and jupyter), to make the initial experience smoother.
	if opts.IDE != "" && opts.IsServerlessMode() && !opts.ProxyMode && !opts.SkipSettingsCheck && cmdio.IsPromptSupported(ctx) {
		err = vscode.CheckAndUpdateSettings(ctx, opts.IDE, opts.ConnectionName)
		if err != nil {
			cmdio.LogString(ctx, fmt.Sprintf("Failed to update IDE settings: %v", err))
			cmdio.LogString(ctx, vscode.GetManualInstructions(opts.IDE, opts.ConnectionName))
			cmdio.LogString(ctx, "Use --skip-settings-check to bypass IDE settings verification.")
			shouldProceed, promptErr := cmdio.AskYesOrNo(ctx, "Do you want to proceed with the connection?")
			if promptErr != nil {
				return fmt.Errorf("failed to prompt user: %w", promptErr)
			}
			if !shouldProceed {
				return errors.New("aborted: IDE settings need to be updated manually, user declined to proceed")
			}
		}
	}

	var userName string
	var serverPort int
	var clusterID string

	version := build.GetInfo().Version

	if opts.ServerMetadata == "" {
		cmdio.LogString(ctx, "Checking for ssh-tunnel binaries to upload...")
		if err := UploadTunnelReleases(ctx, client, version, opts.ReleasesDir); err != nil {
			return fmt.Errorf("failed to upload ssh-tunnel binaries: %w", err)
		}
		userName, serverPort, clusterID, err = ensureSSHServerIsRunning(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return fmt.Errorf("failed to ensure that ssh server is running: %w", err)
		}
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

	cmdio.LogString(ctx, "Remote user name: "+userName)
	cmdio.LogString(ctx, fmt.Sprintf("Server port: %d", serverPort))
	if opts.IsServerlessMode() {
		cmdio.LogString(ctx, "Cluster ID (from serverless job): "+clusterID)
	}

	if opts.ProxyMode {
		return runSSHProxy(ctx, client, serverPort, clusterID, opts)
	} else if opts.IDE != "" {
		return runIDE(ctx, client, userName, keyPath, serverPort, clusterID, opts)
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("Additional SSH arguments: %v", opts.AdditionalArgs))
		return spawnSSHClient(ctx, userName, keyPath, serverPort, clusterID, opts)
	}
}

func runIDE(ctx context.Context, client *databricks.WorkspaceClient, userName, keyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	if opts.IDE != VSCodeOption && opts.IDE != CursorOption {
		return fmt.Errorf("invalid IDE value: %s, expected '%s' or '%s'", opts.IDE, VSCodeOption, CursorOption)
	}

	connectionName := opts.SessionIdentifier()
	if connectionName == "" {
		return errors.New("connection name is required for IDE integration")
	}

	// Get Databricks user name for the workspace path
	currentUser, err := client.CurrentUser.Me(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	databricksUserName := currentUser.UserName

	// Ensure SSH config entry exists
	configPath, err := sshconfig.GetMainConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get SSH config path: %w", err)
	}

	err = ensureSSHConfigEntry(ctx, configPath, connectionName, userName, keyPath, serverPort, clusterID, opts)
	if err != nil {
		return fmt.Errorf("failed to ensure SSH config entry: %w", err)
	}

	ideCommand := VSCodeCommand
	if opts.IDE == CursorOption {
		ideCommand = CursorCommand
	}

	// Construct the remote SSH URI
	// Format: ssh-remote+<server_user_name>@<connection_name> /Workspace/Users/<databricks_user_name>/
	remoteURI := fmt.Sprintf("ssh-remote+%s@%s", userName, connectionName)
	remotePath := fmt.Sprintf("/Workspace/Users/%s/", databricksUserName)

	cmdio.LogString(ctx, fmt.Sprintf("Launching %s with remote URI: %s and path: %s", opts.IDE, remoteURI, remotePath))

	ideCmd := exec.CommandContext(ctx, ideCommand, "--remote", remoteURI, remotePath)
	ideCmd.Stdout = os.Stdout
	ideCmd.Stderr = os.Stderr

	return ideCmd.Run()
}

func ensureSSHConfigEntry(ctx context.Context, configPath, hostName, userName, keyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	// Ensure the Include directive exists in the main SSH config
	err := sshconfig.EnsureIncludeDirective(configPath)
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

	cmdio.LogString(ctx, fmt.Sprintf("Updated SSH config entry for '%s'", hostName))
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

	workspaceID, err := client.CurrentWorkspaceID(ctx)
	if err != nil {
		return 0, "", "", err
	}
	metadataURL := fmt.Sprintf("%s/driver-proxy-api/o/%d/%s/%d/metadata", client.Config.Host, workspaceID, effectiveClusterID, wsMetadata.Port)
	log.Debugf(ctx, "Metadata URL: %s", metadataURL)
	req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
	if err != nil {
		return 0, "", "", err
	}
	if liteswap != "" {
		req.Header.Set("x-databricks-traffic-id", "testenv://liteswap/"+liteswap)
	}
	if err := client.Config.Authenticate(req); err != nil {
		return 0, "", "", err
	}
	resp, err := http.DefaultClient.Do(req)
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

func submitSSHTunnelJob(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) error {
	sessionID := opts.SessionIdentifier()
	contentDir, err := sshWorkspace.GetWorkspaceContentDir(ctx, client, version, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	err = client.Workspace.MkdirsByPath(ctx, contentDir)
	if err != nil {
		return fmt.Errorf("failed to create directory in the remote workspace: %w", err)
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
		return fmt.Errorf("failed to create ssh-tunnel notebook: %w", err)
	}

	baseParams := map[string]string{
		"version":                 version,
		"secretScopeName":         secretScopeName,
		"authorizedKeySecretName": opts.ClientPublicKeyName,
		"shutdownDelay":           opts.ShutdownDelay.String(),
		"maxClients":              strconv.Itoa(opts.MaxClients),
		"sessionId":               sessionID,
	}

	cmdio.LogString(ctx, "Submitting a job to start the ssh server...")

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
			cmdio.LogString(ctx, "Using accelerator: "+opts.Accelerator)
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
					EnvironmentVersion: "3",
				},
			},
		}
	}

	waiter, err := client.Jobs.Submit(ctx, submitRequest)
	if err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	cmdio.LogString(ctx, fmt.Sprintf("Job submitted successfully with run ID: %d", waiter.RunId))

	return waitForJobToStart(ctx, client, waiter.RunId, opts.TaskStartupTimeout)
}

func spawnSSHClient(ctx context.Context, userName, privateKeyPath string, serverPort int, clusterID string, opts ClientOptions) error {
	// Create a copy with metadata for the ProxyCommand
	optsWithMetadata := opts
	optsWithMetadata.ServerMetadata = FormatMetadata(userName, serverPort, clusterID)

	proxyCommand, err := optsWithMetadata.ToProxyCommand()
	if err != nil {
		return fmt.Errorf("failed to generate ProxyCommand: %w", err)
	}

	hostName := opts.SessionIdentifier()

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
	sshArgs = append(sshArgs, hostName)
	sshArgs = append(sshArgs, opts.AdditionalArgs...)

	log.Debugf(ctx, "Launching SSH client: ssh %s", strings.Join(sshArgs, " "))
	sshCmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
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
	if autoStart {
		cmdio.LogString(ctx, "Ensuring the cluster is running: "+clusterID)
		err := client.Clusters.EnsureClusterIsRunning(ctx, clusterID)
		if err != nil {
			return fmt.Errorf("failed to ensure that the cluster is running: %w", err)
		}
	} else {
		cmdio.LogString(ctx, "Checking cluster state: "+clusterID)
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
func waitForJobToStart(ctx context.Context, client *databricks.WorkspaceClient, runID int64, taskStartupTimeout time.Duration) error {
	cmdio.LogString(ctx, "Waiting for the SSH server task to start...")
	var prevState jobs.RunLifecycleStateV2State

	_, err := retries.Poll(ctx, taskStartupTimeout, func() (*jobs.RunTask, *retries.Err) {
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

		// Print status if it changed
		if currentState != prevState {
			cmdio.LogString(ctx, fmt.Sprintf("Task status: %s", currentState))
			prevState = currentState
		}

		// Check if task is running
		if currentState == jobs.RunLifecycleStateV2StateRunning {
			cmdio.LogString(ctx, "SSH server task is now running, proceeding to connect...")
			return sshTask, nil
		}

		// Check for terminal failure states
		if currentState == jobs.RunLifecycleStateV2StateTerminated {
			return nil, retries.Halt(errors.New("task terminated before reaching running state"))
		}

		// Continue polling for other states
		return nil, retries.Continues(fmt.Sprintf("waiting for task to start (current state: %s)", currentState))
	})

	return err
}

func ensureSSHServerIsRunning(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) (string, int, string, error) {
	sessionID := opts.SessionIdentifier()
	// For dedicated clusters, use clusterID; for serverless, it will be read from metadata
	clusterID := opts.ClusterID

	serverPort, userName, effectiveClusterID, err := getServerMetadata(ctx, client, sessionID, clusterID, version, opts.Liteswap)
	if errors.Is(err, errServerMetadata) {
		cmdio.LogString(ctx, "SSH server is not running, starting it now...")

		err := submitSSHTunnelJob(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return "", 0, "", fmt.Errorf("failed to submit and start ssh server job: %w", err)
		}

		cmdio.LogString(ctx, "Waiting for the ssh server to start...")
		maxRetries := 30
		for retries := range maxRetries {
			if ctx.Err() != nil {
				return "", 0, "", ctx.Err()
			}
			serverPort, userName, effectiveClusterID, err = getServerMetadata(ctx, client, sessionID, clusterID, version, opts.Liteswap)
			if err == nil {
				cmdio.LogString(ctx, "Health check successful, starting ssh WebSocket connection...")
				break
			} else if retries < maxRetries-1 {
				time.Sleep(2 * time.Second)
			} else {
				return "", 0, "", fmt.Errorf("failed to start the ssh server: %w", err)
			}
		}
	} else if err != nil {
		return "", 0, "", err
	}

	return userName, serverPort, effectiveClusterID, nil
}
