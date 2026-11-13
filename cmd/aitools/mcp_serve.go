package aitools

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/proxy"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
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
// dialer to set the connect timeout. The profile's InsecureSkipVerify is honored; the clone keeps
// HTTP/2 enabled.
func serveTransport(cfg *config.Config) *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = serveDialer().DialContext
	if cfg.InsecureSkipVerify {
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = true
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
	if !isFullName(strings.Split(f.service, ".")) {
		return fmt.Errorf("%s needs a full service name such as system.ai.github", mcpconfig.ServiceFlag)
	}

	state, err := mcp.LoadState(cmd.Context())
	if err != nil {
		return serveFailure(cmd, err, serveBindStatus)
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

	return serveProxy(cmd, proxy.Options{
		In:           cmd.InOrStdin(),
		Out:          cmd.OutOrStdout(),
		Err:          cmd.ErrOrStderr(),
		Endpoint:     mcp.ServiceURL(cfg.Host, f.service),
		Config:       cfg,
		Authenticate: cfg.Authenticate,
		Profile:      profileName,
		Agent:        serveAgent(f.agent, cmd.ErrOrStderr()),
		Transport:    serveTransport(cfg),
	})
}

// serveProxy bridges opts to the service until stdin closes or a signal arrives. opts.Authenticate is wrapped by a
// refreshGuard. A token that cannot be refreshed mid-session stops the proxy with serveBindStatus; see proxyResult.
func serveProxy(cmd *cobra.Command, opts proxy.Options) error {
	ctx, stop := serveSignalContext(cmd.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: opts.Authenticate, profile: opts.Profile, cancel: stop}
	opts.Authenticate = refresh.authenticateRequest
	return proxyResult(cmd, refresh, proxy.Run(ctx, opts))
}

// proxyResult is the outcome of serve once the bridge has returned with runErr. A refresh failure wins over runErr: the
// stdin error that can follow a stopped session is a consequence of it. The bridge logs the failing request's error,
// but not when the client cancelled that request first, so the final line always says why the proxy stopped.
func proxyResult(cmd *cobra.Command, refresh *refreshGuard, runErr error) error {
	if refreshErr := refresh.failure(); refreshErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "stopped after a token refresh failure for profile %s\n", refresh.profile)
		return errors.Join(root.ErrAlreadyPrinted, &root.ExitStatusError{Status: serveBindStatus, Err: refreshErr})
	}
	if runErr != nil {
		return serveFailure(cmd, runErr, 1)
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
// not know, such as an agent added by a later release, is left out and reported on stderr, so the server
// still serves. The registry filter also keeps useragent.With from panicking on a value it does not accept.
func serveAgent(name string, stderr io.Writer) string {
	if name == "" || agents.ByName(name) != nil {
		return name
	}
	fmt.Fprintf(stderr, "unknown agent %q; it is left out of the User-Agent\n", name)
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

// refreshGuard wraps the profile's authenticate for the bridge. A token that cannot be refreshed mid-session fails
// that request with an error that names the profile and its login command, then stops the proxy, so the command exits
// 2 and the next spawn re-runs the binding checks. Later requests fail at once, without asking for a token again. A
// request whose own context ended, which the client cancelled or shutdown ended, does not stop it. Nor does the
// session DELETE at shutdown, which is best effort: after a refresh failure it is skipped, as the session is over.
type refreshGuard struct {
	authenticate func(*http.Request) error
	profile      string
	cancel       context.CancelFunc

	mu  sync.Mutex
	err error
}

// authenticateRequest is the bridge's Authenticate. It is safe for concurrent use.
func (g *refreshGuard) authenticateRequest(r *http.Request) error {
	if g.failure() != nil {
		if r.Method == http.MethodDelete {
			return proxy.ErrSkipRequest
		}
		return g.stoppedError()
	}
	err := g.authenticate(r)
	if err == nil {
		return nil
	}
	if r.Context().Err() != nil || r.Method == http.MethodDelete {
		return err
	}
	// The hint says to restart the server, since the next spawn is what picks up the new login.
	authErr := fmt.Errorf("%w, then restart the MCP server in your agent", mcp.AuthError(g.profile, err))
	g.mu.Lock()
	first := g.err == nil
	if first {
		g.err = authErr
	}
	g.mu.Unlock()
	if !first {
		return g.stoppedError()
	}
	g.cancel()
	return authErr
}

// stoppedError is the error for a request that arrives after the refresh failure. It does not repeat the login hint.
func (g *refreshGuard) stoppedError() error {
	return fmt.Errorf("the session is stopping after a token refresh failure for profile %s", g.profile)
}

// failure returns the first refresh error, or nil when every token was refreshed.
func (g *refreshGuard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}
