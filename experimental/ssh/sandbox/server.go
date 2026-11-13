package sandbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
)

// ServerOptions describes a relay whose SSH configuration was prepared by the caller.
type ServerOptions struct {
	SSHD            string
	Config          string
	IdleTimeout     time.Duration
	OutputHeartbeat time.Duration
}

// Readiness is written to command output only after the listener has been bound.
type Readiness struct {
	Port int `json:"port"`
}

// Serve runs a sandbox relay without workspace credentials or compute initialization.
// Each invocation owns its listener and all the SSH processes it starts.
// With heartbeats enabled, output must support Close interrupting pending writes;
// Serve closes it during shutdown and waits for the output worker to finish.
func Serve(ctx context.Context, listener net.Listener, opts ServerOptions, output io.Writer) error {
	defer listener.Close()
	if opts.IdleTimeout <= 0 {
		return errors.New("idle timeout must be positive")
	}
	if opts.Config == "" || opts.SSHD == "" {
		return errors.New("sshd executable and configuration are required")
	}
	var heartbeatOutput io.WriteCloser
	if opts.OutputHeartbeat > 0 {
		var ok bool
		heartbeatOutput, ok = output.(io.WriteCloser)
		if !ok {
			return errors.New("command heartbeat output must support closing")
		}
	}
	// Validate the configuration before announcing readiness.
	check := exec.CommandContext(ctx, opts.SSHD, "-t", "-f", opts.Config)
	if diagnostic, err := check.CombinedOutput(); err != nil {
		return fmt.Errorf("validate sshd configuration: %w: %s", err, diagnostic)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	connections := proxy.NewConnectionsManager(1, opts.IdleTimeout)
	defer connections.Close()
	handler := proxy.NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, opts.SSHD, "-i", "-f", opts.Config)
	})

	// One relay belongs to one invocation; the sandbox client does not negotiate
	// compute's handover or resume protocols.
	var requests sync.WaitGroup
	var requestMu sync.Mutex
	active, closing := false, false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ssh", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Has("delivered") || query.Has("reattach") || query.Has("resume_version") {
			http.Error(w, "Session resume is not supported by this relay", http.StatusBadRequest)
			return
		}
		requestMu.Lock()
		if active || closing {
			requestMu.Unlock()
			http.Error(w, "SSH connection already active or relay is stopping", http.StatusServiceUnavailable)
			return
		}
		active = true
		requests.Add(1)
		requestMu.Unlock()
		defer func() {
			requestMu.Lock()
			active = false
			connections.ExtendIdleTimeout(opts.IdleTimeout)
			requestMu.Unlock()
			requests.Done()
		}()
		handler.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	stopped := make(chan struct{})
	var serveErr error
	go func() { serveErr = server.Serve(listener); close(stopped) }()
	defer func() {
		requestMu.Lock()
		closing = true
		requestMu.Unlock()
		cancel()
		server.Close()
		<-stopped
		requests.Wait()
	}()
	if err := json.NewEncoder(output).Encode(Readiness{Port: listener.Addr().(*net.TCPAddr).Port}); err != nil {
		return fmt.Errorf("report relay readiness: %w", err)
	}
	heartbeatErrors := make(chan error, 1)
	if opts.OutputHeartbeat > 0 {
		heartbeatDone := make(chan struct{})
		defer func() {
			cancel()
			heartbeatOutput.Close()
			<-heartbeatDone
		}()
		go func() {
			defer close(heartbeatDone)
			ticker := time.NewTicker(opts.OutputHeartbeat)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					// SSH traffic travels over a separate WebSocket connection.
					if _, err := io.WriteString(heartbeatOutput, "\n"); err != nil {
						heartbeatErrors <- err
						return
					}
				}
			}
		}()
	}
	for {
		select {
		case err := <-heartbeatErrors:
			return fmt.Errorf("write command heartbeat: %w", err)
		case <-ctx.Done():
			return ctx.Err()
		case <-connections.TimedOut:
			// A timer may fire just as a connection is admitted. Admission and the idle
			// decision share a lock so an already queued timeout cannot end that session.
			requestMu.Lock()
			if active {
				requestMu.Unlock()
				continue
			}
			closing = true
			requestMu.Unlock()
			return nil
		case <-stopped:
			if errors.Is(serveErr, http.ErrServerClosed) {
				return nil
			}
			return serveErr
		}
	}
}
