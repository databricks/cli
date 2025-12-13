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
	"github.com/databricks/cli/experimental/ssh/internal/setup"
	sshWorkspace "github.com/databricks/cli/experimental/ssh/internal/workspace"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/gorilla/websocket"
)

//go:embed ssh-server-bootstrap.py
var sshServerBootstrapScript string

var errServerMetadata = errors.New("server metadata error")

type ClientOptions struct {
	// Id of the cluster to connect to
	ClusterID string
	// Delay before shutting down the server after the last client disconnects
	ShutdownDelay time.Duration
	// Maximum number of SSH clients
	MaxClients int
	// Indicates that the CLI runs as a ProxyCommand - it should establish ws connection
	// to the cluster and proxy all traffic through stdin/stdout.
	// In the non proxy mode the CLI spawns an ssh client with the ProxyCommand config.
	ProxyMode bool
	// Expected format: "<user_name>,<port>".
	// If present, the CLI won't attempt to start the server.
	ServerMetadata string
	// How often the CLI should reconnect to the server with new auth.
	HandoverTimeout time.Duration
	// Max amount of time the server process is allowed to live
	ServerTimeout time.Duration
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

	err := checkClusterState(ctx, client, opts.ClusterID, opts.AutoStartCluster)
	if err != nil {
		return err
	}

	secretScopeName, err := keys.CreateKeysSecretScope(ctx, client, opts.ClusterID)
	if err != nil {
		return fmt.Errorf("failed to create secret scope: %w", err)
	}

	privateKeyBytes, publicKeyBytes, err := keys.CheckAndGenerateSSHKeyPairFromSecrets(ctx, client, opts.ClusterID, secretScopeName, opts.ClientPrivateKeyName, opts.ClientPublicKeyName)
	if err != nil {
		return fmt.Errorf("failed to get or generate SSH key pair from secrets: %w", err)
	}

	keyPath, err := keys.GetLocalSSHKeyPath(opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return fmt.Errorf("failed to get local keys folder: %w", err)
	}

	err = keys.SaveSSHKeyPair(keyPath, privateKeyBytes, publicKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to save SSH key pair locally: %w", err)
	}
	cmdio.LogString(ctx, "Using SSH key: "+keyPath)
	cmdio.LogString(ctx, fmt.Sprintf("Secrets scope: %s, key name: %s", secretScopeName, opts.ClientPublicKeyName))

	var userName string
	var serverPort int

	version := build.GetInfo().Version

	if opts.ServerMetadata == "" {
		cmdio.LogString(ctx, "Checking for ssh-tunnel binaries to upload...")
		if err := UploadTunnelReleases(ctx, client, version, opts.ReleasesDir); err != nil {
			return fmt.Errorf("failed to upload ssh-tunnel binaries: %w", err)
		}
		userName, serverPort, err = ensureSSHServerIsRunning(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return fmt.Errorf("failed to ensure that ssh server is running: %w", err)
		}
	} else {
		metadata := strings.Split(opts.ServerMetadata, ",")
		if len(metadata) != 2 {
			return fmt.Errorf("invalid metadata: %s, expected format: <user_name>,<port>", opts.ServerMetadata)
		}
		userName = metadata[0]
		if userName == "" {
			return fmt.Errorf("remote user name is empty in the metadata: %s", opts.ServerMetadata)
		}
		serverPort, err = strconv.Atoi(metadata[1])
		if err != nil {
			return fmt.Errorf("cannot parse port from metadata: %s, %w", opts.ServerMetadata, err)
		}
	}

	cmdio.LogString(ctx, "Remote user name: "+userName)
	cmdio.LogString(ctx, fmt.Sprintf("Server port: %d", serverPort))

	if opts.ProxyMode {
		return runSSHProxy(ctx, client, serverPort, opts)
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("Additional SSH arguments: %v", opts.AdditionalArgs))
		return spawnSSHClient(ctx, userName, keyPath, serverPort, opts)
	}
}

func getServerMetadata(ctx context.Context, client *databricks.WorkspaceClient, clusterID, version string) (int, string, error) {
	serverPort, err := sshWorkspace.GetWorkspaceMetadata(ctx, client, version, clusterID)
	if err != nil {
		return 0, "", errors.Join(errServerMetadata, err)
	}
	workspaceID, err := client.CurrentWorkspaceID(ctx)
	if err != nil {
		return 0, "", err
	}
	metadataURL := fmt.Sprintf("%s/driver-proxy-api/o/%d/%s/%d/metadata", client.Config.Host, workspaceID, clusterID, serverPort)
	req, err := http.NewRequestWithContext(ctx, "GET", metadataURL, nil)
	if err != nil {
		return 0, "", err
	}
	if err := client.Config.Authenticate(req); err != nil {
		return 0, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", errors.Join(errServerMetadata, fmt.Errorf("server is not ok, status code %d", resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", err
	}
	return serverPort, string(bodyBytes), nil
}

func submitSSHTunnelJob(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) (int64, error) {
	contentDir, err := sshWorkspace.GetWorkspaceContentDir(ctx, client, version, opts.ClusterID)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	err = client.Workspace.MkdirsByPath(ctx, contentDir)
	if err != nil {
		return 0, fmt.Errorf("failed to create directory in the remote workspace: %w", err)
	}

	sshTunnelJobName := "ssh-server-bootstrap-" + opts.ClusterID
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

	submitRun := jobs.SubmitRun{
		RunName:        sshTunnelJobName,
		TimeoutSeconds: int(opts.ServerTimeout.Seconds()),
		Tasks: []jobs.SubmitTask{
			{
				TaskKey: "start_ssh_server",
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: jobNotebookPath,
					BaseParameters: map[string]string{
						"version":                 version,
						"secretScopeName":         secretScopeName,
						"authorizedKeySecretName": opts.ClientPublicKeyName,
						"shutdownDelay":           opts.ShutdownDelay.String(),
						"maxClients":              strconv.Itoa(opts.MaxClients),
					},
				},
				TimeoutSeconds:    int(opts.ServerTimeout.Seconds()),
				ExistingClusterId: opts.ClusterID,
			},
		},
	}

	cmdio.LogString(ctx, "Submitting a job to start the ssh server...")
	runResult, err := client.Jobs.Submit(ctx, submitRun)
	if err != nil {
		return 0, fmt.Errorf("failed to submit job: %w", err)
	}

	return runResult.Response.RunId, nil
}

func spawnSSHClient(ctx context.Context, userName, privateKeyPath string, serverPort int, opts ClientOptions) error {
	proxyCommand, err := setup.GenerateProxyCommand(opts.ClusterID, opts.AutoStartCluster, opts.ShutdownDelay, opts.Profile, userName, serverPort, opts.HandoverTimeout)
	if err != nil {
		return fmt.Errorf("failed to generate ProxyCommand: %w", err)
	}

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
	sshArgs = append(sshArgs, opts.ClusterID)
	sshArgs = append(sshArgs, opts.AdditionalArgs...)

	cmdio.LogString(ctx, "Launching SSH client: ssh "+strings.Join(sshArgs, " "))

	sshCmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func runSSHProxy(ctx context.Context, client *databricks.WorkspaceClient, serverPort int, opts ClientOptions) error {
	createConn := func(ctx context.Context, connID string) (*websocket.Conn, error) {
		return createWebsocketConnection(ctx, client, connID, opts.ClusterID, serverPort)
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

func ensureSSHServerIsRunning(ctx context.Context, client *databricks.WorkspaceClient, version, secretScopeName string, opts ClientOptions) (string, int, error) {
	serverPort, userName, err := getServerMetadata(ctx, client, opts.ClusterID, version)
	if errors.Is(err, errServerMetadata) {
		cmdio.LogString(ctx, "SSH server is not running, starting it now...")

		runID, err := submitSSHTunnelJob(ctx, client, version, secretScopeName, opts)
		if err != nil {
			return "", 0, fmt.Errorf("failed to submit ssh server job: %w", err)
		}
		cmdio.LogString(ctx, fmt.Sprintf("Job submitted successfully with run ID: %d", runID))

		cmdio.LogString(ctx, "Waiting for the ssh server to start...")
		maxRetries := 30
		for retries := range maxRetries {
			if ctx.Err() != nil {
				return "", 0, ctx.Err()
			}
			serverPort, userName, err = getServerMetadata(ctx, client, opts.ClusterID, version)
			if err == nil {
				cmdio.LogString(ctx, "Health check successful, starting ssh WebSocket connection...")
				break
			} else if retries < maxRetries-1 {
				time.Sleep(2 * time.Second)
			} else {
				return "", 0, fmt.Errorf("failed to start the ssh server: %w", err)
			}
		}
	} else if err != nil {
		return "", 0, err
	}

	return userName, serverPort, nil
}
