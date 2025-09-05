package ssh

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
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

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/gorilla/websocket"
	"golang.org/x/sync/errgroup"
)

type PortMetadata struct {
	Port int `json:"port"`
}

//go:embed ssh-server-bootstrap.py
var sshServerBootstrapScript string

var errServerMetadata = errors.New("server metadata error")

const serverJobTimeoutSeconds = 24 * 60 * 60

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
	// Directory for local SSH tunnel development releases.
	// If not present, the CLI will use github releases with the current version.
	ReleasesDir string
	// Directory for local SSH keys. Defaults to ~/.databricks/ssh-tunnel-keys
	SSHKeysDir string
	// Name of the client public key file to be used in the ssh-tunnel secrets scope.
	ClientPublicKeyName string
	// Additional arguments to pass to the SSH client in the non proxy mode.
	AdditionalArgs []string
}

func RunClient(ctx context.Context, client *databricks.WorkspaceClient, opts ClientOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		<-sigCh
		cmdio.LogString(ctx, "Received termination signal, cleaning up...")
		cancel()
	}()

	keyPath, err := getLocalSSHKeyPath(opts.ClusterID, opts.SSHKeysDir)
	if err != nil {
		return fmt.Errorf("failed to get local keys folder: %w", err)
	}
	privateKeyPath, publicKey, err := checkAndGenerateSSHKeyPair(ctx, keyPath)
	if err != nil {
		return fmt.Errorf("failed to check or generate SSH key pair: %w", err)
	}

	secretsScopeName, err := putSecretInScope(ctx, client, opts.ClusterID, opts.ClientPublicKeyName, publicKey)
	if err != nil {
		return fmt.Errorf("failed to store public key in secret scope: %w", err)
	}
	cmdio.LogString(ctx, fmt.Sprintf("Secrets scope: %s, key name: %s", secretsScopeName, opts.ClientPublicKeyName))

	var userName string
	var serverPort int

	version := build.GetInfo().Version

	if opts.ServerMetadata == "" {
		cmdio.LogString(ctx, "Checking for ssh-tunnel binaries to upload...")
		if err := uploadTunnelBinaries(ctx, client, version, opts.ReleasesDir); err != nil {
			return fmt.Errorf("failed to upload ssh-tunnel binaries: %w", err)
		}
		userName, serverPort, err = ensureSSHServerIsRunning(ctx, client, opts.ClusterID, secretsScopeName, opts.ClientPublicKeyName, version, opts.ShutdownDelay, opts.MaxClients)
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
		return startSSHProxy(ctx, client, opts.ClusterID, serverPort, opts.HandoverTimeout)
	} else {
		cmdio.LogString(ctx, fmt.Sprintf("Additional SSH arguments: %v", opts.AdditionalArgs))
		return spawnSSHClient(ctx, opts.ClusterID, userName, privateKeyPath, serverPort, opts.HandoverTimeout, opts.AdditionalArgs)
	}
}

func getWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, clusterID string) (int, error) {
	contentDir, err := getWorkspaceContentDir(ctx, client, version, clusterID)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	metadataPath := filepath.ToSlash(filepath.Join(contentDir, "metadata.json"))

	content, err := client.Workspace.Download(ctx, metadataPath)
	if err != nil {
		return 0, fmt.Errorf("failed to download metadata file: %w", err)
	}
	defer content.Close()

	metadataBytes, err := io.ReadAll(content)
	if err != nil {
		return 0, fmt.Errorf("failed to read metadata content: %w", err)
	}

	var metadata PortMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return metadata.Port, nil
}

func getServerMetadata(ctx context.Context, client *databricks.WorkspaceClient, clusterID, version string) (int, string, error) {
	serverPort, err := getWorkspaceMetadata(ctx, client, version, clusterID)
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

func submitSSHTunnelJob(ctx context.Context, client *databricks.WorkspaceClient, clusterID, secretsScope, publicKeySecretName, version string, shutdownDelay time.Duration, maxClients int) (int64, error) {
	contentDir, err := getWorkspaceContentDir(ctx, client, version, clusterID)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	err = client.Workspace.MkdirsByPath(ctx, contentDir)
	if err != nil {
		return 0, fmt.Errorf("failed to create directory in the remote workspace: %w", err)
	}

	sshTunnelJobName := "ssh-server-bootstrap-" + clusterID
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
		TimeoutSeconds: serverJobTimeoutSeconds,
		Tasks: []jobs.SubmitTask{
			{
				TaskKey: "start_ssh_server",
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: jobNotebookPath,
					BaseParameters: map[string]string{
						"version":             version,
						"secretsScope":        secretsScope,
						"publicKeySecretName": publicKeySecretName,
						"shutdownDelay":       shutdownDelay.String(),
						"maxClients":          strconv.Itoa(maxClients),
					},
				},
				TimeoutSeconds:    serverJobTimeoutSeconds,
				ExistingClusterId: clusterID,
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

func spawnSSHClient(ctx context.Context, clusterID, userName, privateKeyPath string, serverPort int, handoverTimeout time.Duration, additionalArgs []string) error {
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	proxyCommand := fmt.Sprintf("%s ssh connect --proxy --cluster=%s --handover-timeout=%s --metadata=%s,%d",
		executablePath, clusterID, handoverTimeout.String(), userName, serverPort)

	sshArgs := []string{
		"-l", userName,
		"-i", privateKeyPath,
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=360",
		"-o", "ProxyCommand=" + proxyCommand,
		clusterID,
	}
	sshArgs = append(sshArgs, additionalArgs...)

	cmdio.LogString(ctx, "Launching SSH client: ssh "+strings.Join(sshArgs, " "))

	sshCmd := exec.CommandContext(ctx, "ssh", sshArgs...)

	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr

	return sshCmd.Run()
}

func startSSHProxy(ctx context.Context, client *databricks.WorkspaceClient, clusterID string, serverPort int, handoverTimeout time.Duration) error {
	g, gCtx := errgroup.WithContext(ctx)

	cmdio.LogString(ctx, "Establishing SSH proxy connection...")
	proxy := newProxyConnection(func(ctx context.Context, connID string) (*websocket.Conn, error) {
		return createWebsocketConnection(ctx, client, connID, clusterID, serverPort)
	})
	if err := proxy.Connect(gCtx); err != nil {
		return fmt.Errorf("failed to connect to proxy: %w", err)
	}
	defer proxy.Close()
	cmdio.LogString(ctx, "SSH proxy connection established")

	cmdio.LogString(ctx, fmt.Sprintf("Connection handover timeout: %v", handoverTimeout))
	handoverTicker := time.NewTicker(handoverTimeout)
	defer handoverTicker.Stop()

	g.Go(func() error {
		for {
			select {
			case <-gCtx.Done():
				return gCtx.Err()
			case <-handoverTicker.C:
				err := proxy.InitiateHandover(gCtx)
				if err != nil {
					return err
				}
			}
		}
	})

	g.Go(func() error {
		return proxy.Start(gCtx, os.Stdin, os.Stdout)
	})

	return g.Wait()
}

func ensureSSHServerIsRunning(ctx context.Context, client *databricks.WorkspaceClient, clusterID, secretsScope, publicKeySecretName, version string, shutdownDelay time.Duration, maxClients int) (string, int, error) {
	cmdio.LogString(ctx, "Ensuring the cluster is running: "+clusterID)
	err := client.Clusters.EnsureClusterIsRunning(ctx, clusterID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to ensure that the cluster is running: %w", err)
	}

	serverPort, userName, err := getServerMetadata(ctx, client, clusterID, version)
	if errors.Is(err, errServerMetadata) {
		cmdio.LogString(ctx, "SSH server is not running, starting it now...")

		runID, err := submitSSHTunnelJob(ctx, client, clusterID, secretsScope, publicKeySecretName, version, shutdownDelay, maxClients)
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
			serverPort, userName, err = getServerMetadata(ctx, client, clusterID, version)
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
