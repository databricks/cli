package sandbox

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"uuid"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/gorilla/websocket"
)

//go:embed setup.py
var setupScript string

//go:embed launch.py
var serverLauncherScript string

type remoteSetup struct {
	Home         string `json:"home"`
	Config       string `json:"config"`
	SSHD         string `json:"sshd"`
	User         string `json:"user"`
	HostKey      string `json:"host_key"`
	Architecture string `json:"architecture"`
	SessionDir   string `json:"session_dir"`
}

// WaitForRunning preserves stopped-sandbox startup for SSH callers.
func WaitForRunning(ctx context.Context, w *databricks.WorkspaceClient, name string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	started := false
	progressed := false
	for {
		sandbox, err := w.Sandbox.GetSandbox(ctx, sdk.GetSandboxRequest{Name: name})
		if err != nil {
			return err
		}
		if sandbox.Status == nil {
			return fmt.Errorf("sandbox %s has no runtime status", name)
		}
		switch sandbox.Status.State {
		case sdk.SandboxStateSandboxStateRunning:
			return nil
		case sdk.SandboxStateSandboxStateStopped:
			if started && progressed {
				return fmt.Errorf("sandbox %s stopped before becoming ready", name)
			}
			// Start acknowledgement can precede the observable state transition.
			if !started {
				if _, err := w.Sandbox.StartSandbox(ctx, sdk.StartSandboxRequest{Name: name}); err != nil {
					return err
				}
				started = true
			}
		case sdk.SandboxStateSandboxStateStopping, sdk.SandboxStateSandboxStatePending:
			progressed = started
		default:
			return fmt.Errorf("sandbox %s has unsupported state %q", name, sandbox.Status.State)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func prepareRemote(ctx context.Context, w *databricks.WorkspaceClient, name string, identity *Identity) (*remoteSetup, error) {
	request, err := json.Marshal(map[string]string{"public_key": string(identity.PublicKey)})
	if err != nil {
		return nil, err
	}
	result, err := w.Sandbox.ExecuteCommandSync(ctx, sdk.ExecuteCommandSyncRequest{Name: name, Cmd: "python3", Args: []string{"-c", setupScript, string(request)}})
	if err != nil {
		return nil, fmt.Errorf("prepare sandbox SSH configuration: %w", err)
	}
	if result.Status != sdk.ExecuteCommandStatusExecuteCommandStatusCompleted || result.ExitCode != 0 || result.Truncated {
		return nil, fmt.Errorf("sandbox SSH setup failed (%s, exit %d): %s", result.Status, result.ExitCode, result.Stderr)
	}
	var setup remoteSetup
	if err := json.Unmarshal([]byte(result.Stdout), &setup); err != nil {
		return nil, fmt.Errorf("read sandbox SSH configuration: %w", err)
	}
	if setup.User != "sandbox-agent" {
		return nil, fmt.Errorf("sandbox SSH requires the sandbox-agent account, got %q", setup.User)
	}
	if setup.Config == "" || setup.Home == "" || setup.SSHD == "" || setup.User == "" || setup.SessionDir == "" {
		return nil, errors.New("incomplete sandbox SSH configuration")
	}
	if err := identity.PinHostKey([]byte(setup.HostKey)); err != nil {
		return nil, err
	}
	return &setup, nil
}

// ConnectProxy prepares an invocation's server and carries the SSH stream. It
// never launches a local SSH client or writes diagnostics into the SSH stream.
func ConnectProxy(ctx context.Context, w *databricks.WorkspaceClient, name string, identity *Identity, releasesDir string, input io.ReadCloser, output, diagnostics io.Writer) error {
	return connectProxy(ctx, w, name, identity, releasesDir, input, output, diagnostics, 10*time.Minute)
}

func connectProxy(ctx context.Context, w *databricks.WorkspaceClient, name string, identity *Identity, releasesDir string, input io.ReadCloser, output, diagnostics io.Writer, startupTimeout time.Duration) (err error) {
	ctx, cancelStartup := context.WithCancelCause(ctx)
	defer cancelStartup(nil)
	defer func() {
		if err != nil && ctx.Err() != nil {
			err = context.Cause(ctx)
		}
	}()
	// Stop this timer after readiness; it must not limit an established session.
	startupTimer := time.AfterFunc(startupTimeout, func() { cancelStartup(context.DeadlineExceeded) })
	defer startupTimer.Stop()
	if err := WaitForRunning(ctx, w, name); err != nil {
		return err
	}
	setup, err := prepareRemote(ctx, w, name, identity)
	if err != nil {
		return err
	}
	defer cleanupRemoteDirectory(ctx, w, name, setup.SessionDir)
	binary, err := uploadExecutable(ctx, w, name, setup, releasesDir)
	if err != nil {
		return err
	}
	commandClient, err := sandboxexec.New(w, name)
	if err != nil {
		return err
	}
	defer commandClient.Close()
	commandID := uuid.New().String()
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		if err := commandClient.Terminate(cleanupCtx, commandID); err != nil {
			log.Warnf(ctx, "Could not stop sandbox SSH server: %v", err)
		}
	}()
	commandCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	timer := time.AfterFunc(30*time.Second, func() {
		cancelStartup(errors.New("timed out waiting for sandbox SSH server readiness"))
	})
	defer timer.Stop()
	stream, err := commandClient.Follow(commandCtx, commandID, "python3", []string{"-c", serverLauncherScript, setup.SessionDir, binary, "sandbox", "server", "--sshd-config", setup.Config, "--sshd", setup.SSHD, "--port", "0", "--log-level", "info", "--log-file", path.Join(setup.SessionDir, "server.log")}, 25*time.Minute)
	if err != nil {
		return fmt.Errorf("start sandbox SSH server: %w", err)
	}
	ready := make(chan Readiness, 1)
	stopped := make(chan error, 1)
	go readServerOutput(stream, commandID, ready, stopped, diagnostics)
	var readiness Readiness
	select {
	case result := <-ready:
		readiness = result
	case err := <-stopped:
		if err == nil {
			return errors.New("sandbox SSH server exited before connection")
		}
		return fmt.Errorf("sandbox SSH server exited before readiness: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
	if !timer.Stop() || !startupTimer.Stop() {
		return errors.New("sandbox SSH startup timed out")
	}
	// The output stream stays attached so command failure cancels forwarding.
	proxyCtx, cancelProxy := context.WithCancelCause(ctx)
	defer cancelProxy(nil)
	go func() {
		select {
		case err := <-stopped:
			if err != nil {
				cancelProxy(fmt.Errorf("sandbox SSH server stopped: %w", err))
			}
		case <-proxyCtx.Done():
		}
	}()
	proxyErr := proxyStream(proxyCtx, input, output,
		func(ctx context.Context, request proxy.DialRequest) (*websocket.Conn, error) {
			endpoint, err := forwardingURL(w.Config.Host, name, readiness.Port, request.ConnID)
			if err != nil {
				return nil, err
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err != nil {
				return nil, err
			}
			req.Header.Set(RelayTokenHeader, readiness.Token)
			for key, value := range auth.WorkspaceIDHeaders(w.Config) {
				req.Header.Set(key, value)
			}
			if err := w.Config.Authenticate(req); err != nil {
				return nil, err
			}
			dialer := *websocket.DefaultDialer
			dialer.HandshakeTimeout = 30 * time.Second
			dialer.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: w.Config.InsecureSkipVerify}
			conn, response, err := dialer.DialContext(ctx, endpoint, req.Header)
			if response != nil {
				response.Body.Close()
			}
			return conn, err
		})
	// Seal completion before the server-output watcher can report a later exit.
	cancelProxy(nil)
	if cause := context.Cause(proxyCtx); cause != nil && !errors.Is(cause, context.Canceled) {
		return cause
	}
	return proxyErr
}

func forwardingURL(host, name string, port int, id string) (string, error) {
	endpoint, err := url.Parse(host)
	if err != nil || endpoint.Scheme != "https" {
		return "", errors.New("port forwarding requires an HTTPS workspace URL")
	}
	if !strings.HasPrefix(name, "sandboxes/") || strings.ContainsAny(strings.TrimPrefix(name, "sandboxes/"), "/?#") {
		return "", fmt.Errorf("invalid sandbox resource name %q", name)
	}
	endpoint.Scheme = "wss"
	endpoint.Path = fmt.Sprintf("/sandbox-proxy/%s/ports/%d/ssh", name, port)
	endpoint.RawQuery = url.Values{"id": {id}}.Encode()
	return endpoint.String(), nil
}

func readServerOutput(stream interface {
	Recv() (*sandboxexec.ExecuteCommandResponse, error)
}, commandID string, ready chan<- Readiness, stopped chan<- error, diagnostics io.Writer,
) {
	var stdout strings.Builder
	announced := false
	for {
		response, err := stream.Recv()
		if err != nil {
			stopped <- err
			return
		}
		event := response.GetProcessEvent()
		if start := event.GetStart(); start != nil && start.GetCommandId() != commandID {
			stopped <- errors.New("sandbox returned an unexpected command ID")
			return
		}
		if end := event.GetEnd(); end != nil {
			if end.ExitCode == nil {
				stopped <- errors.New("server exited without an exit status")
				return
			}
			if announced && end.GetExitCode() == 0 {
				// Let the forwarding stream drain its final SSH bytes and close normally.
				stopped <- nil
				return
			}
			stopped <- fmt.Errorf("server exited with status %d", end.GetExitCode())
			return
		}
		if data := event.GetData(); data != nil {
			if !announced {
				if _, err := diagnostics.Write(data.GetStderr()); err != nil {
					stopped <- err
					return
				}
				stdout.Write(data.GetStdout())
				if stdout.Len() > 4096 {
					stopped <- errors.New("sandbox server readiness output is too large")
					return
				}
				if strings.Contains(stdout.String(), "\n") {
					var result Readiness
					if err := json.Unmarshal([]byte(strings.SplitN(stdout.String(), "\n", 2)[0]), &result); err != nil {
						stopped <- err
						return
					}
					if !validForwardPort(result.Port) {
						stopped <- fmt.Errorf("invalid sandbox relay port %d", result.Port)
						return
					}
					if result.Token == "" {
						stopped <- errors.New("sandbox relay readiness has no invocation token")
						return
					}
					ready <- result
					announced = true
				}
			}
		}
	}
}

func cleanupRemoteDirectory(ctx context.Context, w *databricks.WorkspaceClient, name, directory string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	_, err := w.Sandbox.ExecuteCommandSync(ctx, sdk.ExecuteCommandSyncRequest{Name: name, Cmd: "rm", Args: []string{"-rf", "--", directory}})
	if err != nil {
		log.Debugf(ctx, "Could not remove sandbox SSH session files: %v", err)
	}
}

// A read from inherited stdin can remain blocked after Close on Unix. Keep that
// read outside the shared proxy's wait group so a closed WebSocket can end the
// standalone proxy command. Process exit reclaims any remaining stdin read.
func proxyStream(ctx context.Context, input io.ReadCloser, output io.Writer, dial func(context.Context, proxy.DialRequest) (*websocket.Conn, error)) error {
	reader, writer := io.Pipe()
	defer input.Close()
	defer reader.Close()
	defer writer.Close()
	go func() {
		_, err := io.Copy(writer, input)
		writer.CloseWithError(err)
	}()
	// Refresh the WebSocket before the forwarding route's connection lifetime limit.
	handover := time.NewTicker(25 * time.Minute)
	defer handover.Stop()
	return proxy.RunClientProxy(ctx, reader, output, func() <-chan time.Time { return handover.C }, 30*time.Second, false, dial)
}
