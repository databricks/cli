package aitools

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/proxy"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// serveConnectTimeout bounds one upstream connect. http.DefaultTransport waits 30 s.
const serveConnectTimeout = 10 * time.Second

// serveDialer dials the upstream service with serveConnectTimeout.
func serveDialer() *net.Dialer {
	return &net.Dialer{Timeout: serveConnectTimeout}
}

// serveTransport is the transport of the bridge's upstream requests. It starts from
// http.DefaultTransport, so proxy use and the other timeouts stay as they are, and replaces the
// dialer to set the connect timeout. The profile's InsecureSkipVerify is honored.
func serveTransport(cfg *config.Config) *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = serveDialer().DialContext
	if cfg.InsecureSkipVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return tr
}

type mcpServeFlags struct {
	service string
	agent   string
}

// NewMcpServeCmd returns the hidden `aitools mcp serve`: the stdio MCP proxy an agent starts for one
// installed server. Its stdout carries only JSON-RPC messages; every diagnostic goes to stderr.
func NewMcpServeCmd() *cobra.Command {
	var f mcpServeFlags
	cmd := &cobra.Command{
		Use:    "serve",
		Short:  "Serve one Databricks managed MCP server over stdio",
		Hidden: true,
		Args:   cobra.NoArgs,
		// Stdout carries the MCP protocol, so root keeps logs and the stale-version warning off it.
		Annotations: map[string]string{root.StdoutCarriesProtocol: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMcpServe(cmd, f)
		},
	}
	cmd.Flags().StringVar(&f.service, "service", "", "Full name of the MCP service, for example system.ai.github")
	cmd.Flags().StringVar(&f.agent, "agent", "", "Registry name of the calling agent, used only for the User-Agent")
	return cmd
}

// runMcpServe runs the binding checks, then bridges stdio to the service until stdin closes or a
// signal arrives.
func runMcpServe(cmd *cobra.Command, f mcpServeFlags) error {
	if cmd.Flag("output").Changed {
		return errors.New("--output does not apply to serve: its stdout carries the MCP protocol")
	}
	profileName := profileFlag(cmd)
	if profileName == "" {
		return fmt.Errorf("%s is required", mcpconfig.ProfileFlag)
	}
	if f.service == "" {
		return fmt.Errorf("%s is required", mcpconfig.ServiceFlag)
	}

	state, err := mcp.LoadState(cmd.Context())
	if err != nil {
		return serveFailure(cmd, err, 1)
	}
	cfg, _, err := mcp.Bind(cmd.Context(), mcp.BindOptions{
		Profile: profileName,
		Service: f.service,
		State:   state,
		Stderr:  cmd.ErrOrStderr(),
	})
	if err != nil {
		return serveFailure(cmd, err, serveBindStatus)
	}

	// The bound profile authenticates the exit-time telemetry upload.
	cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), cfg))

	ctx, stop := serveSignalContext(cmd.Context())
	defer stop()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	refresh := &refreshGuard{authenticate: cfg.Authenticate, profile: profileName, cancel: cancel}
	err = proxy.Run(ctx, proxy.Options{
		In:           cmd.InOrStdin(),
		Out:          cmd.OutOrStdout(),
		Err:          cmd.ErrOrStderr(),
		Endpoint:     mcp.ServiceURL(cfg.Host, f.service),
		Config:       cfg,
		Authenticate: refresh.authenticateRequest,
		Profile:      profileName,
		Agent:        serveAgent(cmd.Context(), f.agent),
		Transport:    serveTransport(cfg),
	})
	if err != nil {
		return serveFailure(cmd, err, 1)
	}
	if refreshErr := refresh.failure(); refreshErr != nil {
		return serveFailure(cmd, refreshErr, serveBindStatus)
	}
	return nil
}

// serveSignalContext returns a context that the first SIGINT or SIGTERM cancels. The handler is released
// before the context reports done, so a second signal during shutdown ends the process rather than being
// swallowed. stop releases the handler and cancels the context.
func serveSignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		select {
		case <-sigs:
			signal.Stop(sigs)
			cancel()
		case <-ctx.Done():
			signal.Stop(sigs)
		}
	}()
	return ctx, cancel
}

// serveAgent returns the registry name of the calling agent for the User-Agent. A name this binary does
// not know, such as an agent added by a later release, is left out and logged; the server still serves.
func serveAgent(ctx context.Context, name string) string {
	if name == "" || agents.ByName(name) != nil {
		return name
	}
	log.Debugf(ctx, "--agent %q is not a known agent; leaving it out of the User-Agent", name)
	return ""
}

// serveBindStatus is the exit status of a binding or auth failure. Agents and `list --check` tell it from a
// transport error, so it is not 1.
const serveBindStatus = 2

// serveFailure prints err to stderr and returns it with the exit status. Execute does not print it again,
// so the message is written exactly once.
func serveFailure(cmd *cobra.Command, err error, status int) error {
	fmt.Fprintln(cmd.ErrOrStderr(), err)
	return errors.Join(root.ErrAlreadyPrinted, &root.ExitStatusError{Status: status, Err: err})
}

// refreshGuard wraps the profile's authenticate for the bridge. A token that cannot be refreshed
// mid-session fails that request with an error naming the profile, then stops the proxy, so the
// command exits 2 and the next spawn re-runs the binding checks. Two failures do not stop it: a request
// whose own context ended, which the client cancelled or shutdown ended, and the session DELETE at
// shutdown, which is best effort.
type refreshGuard struct {
	authenticate func(*http.Request) error
	profile      string
	cancel       context.CancelFunc

	mu  sync.Mutex
	err error
}

// authenticateRequest is the bridge's Authenticate. It is safe for concurrent use.
func (g *refreshGuard) authenticateRequest(r *http.Request) error {
	err := g.authenticate(r)
	if err == nil {
		return nil
	}
	if r.Context().Err() != nil || r.Method == http.MethodDelete {
		return err
	}
	authErr := mcp.AuthError(g.profile, err)
	g.mu.Lock()
	if g.err == nil {
		g.err = authErr
	}
	g.mu.Unlock()
	g.cancel()
	return authErr
}

// failure returns the first refresh error, or nil when every token was refreshed.
func (g *refreshGuard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}
