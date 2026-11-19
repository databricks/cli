package sandbox

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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
	"github.com/databricks/cli/libs/log"
)

// RelayTokenHeader authenticates connections to this invocation's relay.
const RelayTokenHeader = "X-Sandbox-SSH-Token"

// ServerOptions describes a relay whose SSH configuration was prepared by the caller.
type ServerOptions struct {
	SSHD           string
	Config         string
	ConnectTimeout time.Duration
}

// Readiness is written as one JSON line after the sshd configuration is validated.
// Token is only returned over the caller's authenticated command-output stream.
type Readiness struct {
	Port          int    `json:"port"`
	Token         string `json:"token"`
	ResumeVersion int    `json:"resume_version"`
}

// Serve relays /ssh WebSockets to one sshd session, including its handovers.
// It owns listener and stops its SSH subprocess before returning.
func Serve(ctx context.Context, listener net.Listener, opts ServerOptions, output io.Writer) error {
	defer listener.Close()
	if opts.ConnectTimeout <= 0 {
		return errors.New("connection timeout must be positive")
	}
	if opts.Config == "" || opts.SSHD == "" {
		return errors.New("sshd executable and configuration are required")
	}
	check := exec.CommandContext(ctx, opts.SSHD, "-t", "-f", opts.Config)
	if diagnostic, err := check.CombinedOutput(); err != nil {
		return fmt.Errorf("validate sshd configuration: %w: %s", err, diagnostic)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	connections := proxy.NewConnectionsManager(1, 0, proxy.WithoutIdleShutdown())
	token := rand.Text()
	started, completed := make(chan struct{}), make(chan struct{})
	var requests sync.WaitGroup
	var mu sync.Mutex
	closing := false
	handler := proxy.NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		mu.Lock()
		close(started)
		mu.Unlock()
		return exec.CommandContext(ctx, opts.SSHD, "-i", "-f", opts.Config)
	}, proxy.WithSingleSession(), proxy.WithSessionCompletion(func() { close(completed) }))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ssh", func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(RelayTokenHeader)), []byte(token)) != 1 {
			log.Info(ctx, "Rejecting sandbox SSH connection without its invocation token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		mu.Lock()
		if closing {
			mu.Unlock()
			log.Info(ctx, "Rejecting sandbox SSH connection while stopping")
			http.Error(w, "SSH server is stopping", http.StatusServiceUnavailable)
			return
		}
		requests.Add(1)
		mu.Unlock()
		defer requests.Done()
		handler.ServeHTTP(w, r)
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	stopped := make(chan struct{})
	var serveErr error
	go func() { serveErr = server.Serve(listener); close(stopped) }()
	defer func() {
		mu.Lock()
		closing = true
		mu.Unlock()
		cancel()
		server.Close()
		<-stopped
		requests.Wait()
		log.Info(ctx, "Sandbox SSH server stopped")
	}()
	if err := json.NewEncoder(output).Encode(Readiness{Port: listener.Addr().(*net.TCPAddr).Port, Token: token, ResumeVersion: proxy.ResumeProtocolVersion}); err != nil {
		return fmt.Errorf("report relay readiness: %w", err)
	}
	log.Infof(ctx, "Sandbox SSH server ready on port %d", listener.Addr().(*net.TCPAddr).Port)
	timer := time.NewTimer(opts.ConnectTimeout)
	defer timer.Stop()
	firstConnection := started
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-firstConnection:
			timer.Stop()
			firstConnection = nil
		case <-timer.C:
			mu.Lock()
			connected := false
			select {
			case <-started:
				connected = true
			default:
				closing = true
				cancel()
			}
			mu.Unlock()
			if !connected {
				log.Info(ctx, "Sandbox SSH server timed out waiting for its first connection")
				return nil
			}
		case <-completed:
			return nil
		case <-stopped:
			if errors.Is(serveErr, http.ErrServerClosed) {
				return nil
			}
			return serveErr
		}
	}
}
