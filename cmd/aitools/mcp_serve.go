package aitools

import (
	"context"
	"crypto/tls"
	"encoding/json"
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
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// serveConnectTimeout bounds one upstream connect. http.DefaultTransport waits 30 s.
const serveConnectTimeout = 10 * time.Second

// serveTransport is the transport of the bridge's upstream requests. It starts from
// http.DefaultTransport, so proxy use and the other timeouts stay as they are, and replaces the
// dialer to set the connect timeout. The profile's InsecureSkipVerify is honored; the clone keeps
// HTTP/2 enabled.
func serveTransport(cfg *config.Config) *http.Transport {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = (&net.Dialer{Timeout: serveConnectTimeout}).DialContext
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
//
// Known limitation: the binding is checked only when serve starts. If the profile is pointed at another host during
// the session, for example by `databricks auth login --profile P --host B`, serve keeps sending the profile's token,
// which is now B's, to the endpoint of the host it was bound to.
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
	// The names are the ones `aitools mcp add` writes into each agent's config, so an entry binds what add wrote.
	cmd.Flags().StringVar(&f.service, strings.TrimPrefix(mcpconfig.ServiceFlag, "--"), "",
		"Full name of the MCP service, for example system.ai.github")
	cmd.Flags().StringVar(&f.agent, strings.TrimPrefix(mcpconfig.AgentFlag, "--"), "",
		"Registry name of the calling agent, used only for the User-Agent")
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
	if !isServiceName(f.service) {
		return fmt.Errorf("%s needs a full service name such as system.ai.github", mcpconfig.ServiceFlag)
	}

	state, err := mcp.LoadState(cmd.Context())
	if err != nil {
		return serveFailure(cmd, err, serveBindStatus)
	}
	cfg, entry, err := mcp.Bind(cmd.Context(), mcp.BindOptions{
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
	}, mcp.PinnedFile(entry, cfg.ConfigFile))
}

// serveProxy bridges opts to the service until stdin closes or a signal arrives. Authenticate goes through a
// refreshGuard, so a rejected token stops the proxy; see proxyResult. pinnedFile is the config file the server is
// pinned to, or "" (see mcp.PinnedFile).
func serveProxy(cmd *cobra.Command, opts proxy.Options, pinnedFile string) error {
	ctx, stop := serveSignalContext(cmd.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: opts.Authenticate, profile: opts.Profile, pinnedFile: pinnedFile, cancel: stop}
	opts.Authenticate = refresh.authenticateRequest
	return proxyResult(cmd, refresh, proxy.Run(ctx, opts))
}

// proxyResult returns the outcome of serve once the bridge has returned with runErr. A rejected token wins, since a
// stdin error after it is a consequence. The final line always names the rejection, as the bridge may not have logged it.
func proxyResult(cmd *cobra.Command, refresh *refreshGuard, runErr error) error {
	if refreshErr := refresh.failure(); refreshErr != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "stopped after a token refresh failure for profile %s\n", refresh.profile)
		return exitAlreadyPrinted(refreshErr, serveBindStatus)
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

// serveBindStatus is the exit status of a binding or auth failure. It is not 1, so an agent can tell a failed
// binding or sign-in from the other failures, which exit 1.
const serveBindStatus = 2

// serveFailure prints err to stderr and returns it with the exit status. Execute does not print it again,
// so the message is written exactly once.
func serveFailure(cmd *cobra.Command, err error, status int) error {
	fmt.Fprintln(cmd.ErrOrStderr(), err)
	return exitAlreadyPrinted(err, status)
}

// exitAlreadyPrinted returns err as an error that exits with status and that Execute does not print again.
func exitAlreadyPrinted(err error, status int) error {
	return errors.Join(root.ErrAlreadyPrinted, &root.ExitStatusError{Status: status, Err: err})
}

// refreshGuard wraps the profile's authenticate for the bridge. A rejected token (see credentialRejected) fails that
// request with an error that names the profile and its login command, then stops the proxy, which exits 2. Later
// requests fail at once, without asking for a token again. A transient failure fails only its request, and the next
// request asks for a token again. A request whose own context ended does not stop it, nor does the session DELETE at
// shutdown, which is skipped after a rejection.
type refreshGuard struct {
	authenticate func(*http.Request) error
	profile      string
	// pinnedFile is the config file the server is pinned to, or "" (see mcp.PinnedFile).
	pinnedFile string
	cancel     context.CancelFunc

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
	if !credentialRejected(err) {
		// A failure that is not a rejection, such as a network blip, says nothing about the credential: a login would
		// not fix it, and the next request asks for a token again, so the message names neither.
		return fmt.Errorf("could not get a token for Databricks profile \"%s\": %w; the next request tries again", g.profile, err)
	}
	// The hint says to restart the server: a new spawn re-runs the binding checks, and it is the one that picks up the
	// new login.
	authErr := fmt.Errorf("%w, then restart the MCP server in your agent", mcp.AuthError(g.profile, g.pinnedFile, err))
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

// failure returns the first rejection, or nil while no token has been rejected.
func (g *refreshGuard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// credentialRejected reports whether a token failure is one that a new login fixes. That is the CLI's U2M reporting an
// invalid refresh token, or a token endpoint that answers 400 or 401 with the RFC 6749 section 5.2 error invalid_grant
// or invalid_client. Everything else is transient and returns false: network errors, timeouts, 5xx responses, and
// any error this check cannot classify. A transient failure is retried by the next request.
func credentialRejected(err error) bool {
	if _, ok := errors.AsType[*u2m.InvalidRefreshTokenError](err); ok {
		return true
	}
	// The CLI's U2M client returns an *httpclient.HttpError for an error response, with the response body as its message.
	if httpErr, ok := errors.AsType[*httpclient.HttpError](err); ok {
		var body struct {
			Error string `json:"error"`
		}
		if json.Unmarshal([]byte(httpErr.Message), &body) != nil {
			return false
		}
		return rejectedOAuthError(httpErr.StatusCode, body.Error)
	}
	// golang.org/x/oauth2 returns this for an error response from its own HTTP client. It always sets Response.
	if retrieveErr, ok := errors.AsType[*oauth2.RetrieveError](err); ok {
		return rejectedOAuthError(retrieveErr.Response.StatusCode, retrieveErr.ErrorCode)
	}
	return false
}

// rejectedOAuthError reports whether a token endpoint's error response, with its HTTP status and RFC 6749 error code,
// is a rejection.
func rejectedOAuthError(status int, code string) bool {
	return (status == http.StatusBadRequest || status == http.StatusUnauthorized) &&
		(code == "invalid_grant" || code == "invalid_client")
}
