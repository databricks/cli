package aitools

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
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
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/shellquote"
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

// sessionHeader is the MCP session header. The proxy package keeps its copy unexported.
const sessionHeader = "Mcp-Session-Id"

// upstreamLog is the transport of the bridge's upstream requests when serve runs at debug level. It logs one debug line
// per request, with the JSON-RPC method and id from the body, the HTTP status, the time to the response headers, and
// whether a session id was sent, received, or replaced. It logs no body and no header value, so no token or session id
// reaches the log.
type upstreamLog struct {
	next http.RoundTripper

	mu      sync.Mutex
	session string // the last session id the server sent, "" before the first one
}

func newUpstreamLog(next http.RoundTripper) *upstreamLog {
	return &upstreamLog{next: next}
}

// RoundTrip sends req with next and logs the exchange. Below debug level it only forwards the request.
func (u *upstreamLog) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	logger := log.GetLogger(ctx)
	if !logger.Enabled(ctx, log.LevelDebug) {
		return u.next.RoundTrip(req)
	}
	rpcMethod, rpcID := jsonRPCEnvelope(req)
	start := time.Now()
	resp, err := u.next.RoundTrip(req)
	sent := req.Header.Get(sessionHeader) != ""
	received := ""
	status := 0
	if resp != nil {
		received = resp.Header.Get(sessionHeader)
		status = resp.StatusCode
	}
	replaced := u.observeSession(received)
	attrs := []slog.Attr{
		slog.String("http_method", req.Method),
		slog.String("rpc_method", rpcMethod),
		slog.String("rpc_id", rpcID),
		slog.Int("status", status),
		slog.Duration("duration", time.Since(start)),
		slog.Bool("session_sent", sent),
		slog.Bool("session_received", received != ""),
		slog.Bool("session_replaced", replaced),
	}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	logger.LogAttrs(ctx, slog.LevelDebug, "upstream request", attrs...)
	return resp, err
}

// observeSession records the session id that a response carried and reports whether it replaced the one seen before.
// An empty id is no session and changes nothing.
func (u *upstreamLog) observeSession(received string) bool {
	if received == "" {
		return false
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	replaced := u.session != "" && received != u.session
	u.session = received
	return replaced
}

// jsonRPCEnvelope returns the method and the raw id of the JSON-RPC message in req's body. It returns empty strings
// when there is no body, as for the session DELETE, and when the body cannot be read or decoded, since a debug line
// should not fail the request. The body is read through GetBody, so the request itself is unchanged.
func jsonRPCEnvelope(req *http.Request) (method, id string) {
	if req.GetBody == nil {
		return "", ""
	}
	body, err := req.GetBody()
	if err != nil {
		return "", ""
	}
	defer body.Close()
	var msg struct {
		Method string          `json:"method"`
		ID     json.RawMessage `json:"id"`
	}
	if err := json.NewDecoder(body).Decode(&msg); err != nil {
		return "", ""
	}
	return msg.Method, string(msg.ID)
}

// signInRedirect is the outermost transport of the bridge's upstream requests. The MCP endpoint answers a token that it does
// not accept with a 3xx redirect to sign-in, and the bridge does not follow it. Such a response fails its request with the
// login command for the profile, as a rejected credential does. Serve keeps running, and later requests are still sent. The
// Location is not read, since its query can carry state.
type signInRedirect struct {
	next       http.RoundTripper
	profile    string
	configFile string // the config file the profile is pinned to, or "" when the login's default file is used
}

// RoundTrip sends req with next. A 3xx response becomes the login error; any other response, or a transport error, is
// returned as it is.
func (s *signInRedirect) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := s.next.RoundTrip(req)
	if err != nil || resp.StatusCode/100 != 3 {
		return resp, err
	}
	resp.Body.Close()
	return nil, s.loginError(resp.StatusCode)
}

// loginError is the error for a redirect to sign-in. The login command names DATABRICKS_CONFIG_FILE when the profile is
// pinned to a config file, since the login must write that file.
func (s *signInRedirect) loginError(status int) error {
	return fmt.Errorf("the MCP endpoint redirected to sign-in (HTTP %d), so it did not accept the token for Databricks profile \"%s\"\nRun: %s\nThen restart the MCP server in your agent", status, s.profile, mcp.ConfigFileCommand(s.configFile, "databricks auth login --profile "+shellquote.BashArg(s.profile)))
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
		return serveBindFailure(cmd, profileName, err)
	}

	authenticate := rebindingAuthenticate(bindingCheck(cmd.Context(), state, entry, f.service, profileName, cfg), cfg.Authenticate)
	return serveProxy(cmd, proxy.Options{
		In:           cmd.InOrStdin(),
		Out:          cmd.OutOrStdout(),
		Err:          cmd.ErrOrStderr(),
		Endpoint:     mcp.ServiceURL(cfg.Host, f.service),
		Config:       cfg,
		Authenticate: authenticate,
		Profile:      profileName,
		Agent:        serveAgent(f.agent, cmd.ErrOrStderr()),
		Transport: &signInRedirect{
			next:       newUpstreamLog(serveTransport(cfg)),
			profile:    profileName,
			configFile: mcp.PinnedFile(entry, cfg.ConfigFile),
		},
	}, mcp.PinnedFile(entry, cfg.ConfigFile))
}

// bindingCheck returns the check that runs before each token is fetched. With an install record, entry, it checks the
// record. Without one, nothing records the host the profile was bound to, so it holds the profile to the host and
// workspace that cfg, which Bind resolved at start, names.
func bindingCheck(ctx context.Context, state *mcp.State, entry *mcp.Entry, service, profileName string, cfg *config.Config) func() error {
	if entry == nil {
		return func() error {
			return mcp.CheckUnrecordedBinding(service, profileName, cfg.ConfigFile, cfg)
		}
	}
	return func() error {
		_, err := mcp.CheckBinding(ctx, state, service, profileName, cfg.ConfigFile)
		return err
	}
}

// rebindingAuthenticate runs check before each token is fetched. A profile re-pointed after start fails the request
// here, before its token is fetched. The CLI token cache is keyed by profile name alone, so that token would otherwise
// reach the host serve is bound to.
func rebindingAuthenticate(check func() error, authenticate func(*http.Request) error) func(*http.Request) error {
	return func(r *http.Request) error {
		if err := checkBindingWithRetry(check); err != nil {
			return err
		}
		return authenticate(r)
	}
}

// bindingRetryDelay is the pause before a changed binding is checked again. A write that replaces .databrickscfg
// truncates it first, so one read can find no profile.
const bindingRetryDelay = 50 * time.Millisecond

// checkBindingWithRetry runs check. If the binding looks changed, it waits bindingRetryDelay and runs check once more,
// and returns that result. Other failures are returned at once.
func checkBindingWithRetry(check func() error) error {
	err := check()
	if !bindingChanged(err) {
		return err
	}
	time.Sleep(bindingRetryDelay)
	return check()
}

// serveProxy bridges opts to the service until stdin closes or a signal arrives. Authenticate goes through a
// refreshGuard, so a rejected token or a changed binding stops the proxy; see proxyResult. pinnedFile is the config
// file the server is pinned to, or "" (see mcp.PinnedFile).
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
		fmt.Fprintf(cmd.ErrOrStderr(), "Error: stopped after %s for profile \"%s\"\n", stopReason(refreshErr), refresh.profile)
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

// serveFailure prints err to stderr as `Error: <err>`, the form Execute uses for every other failure, and returns it
// with the exit status. Execute does not print it again, so the message is written exactly once.
func serveFailure(cmd *cobra.Command, err error, status int) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "Error: %s\n", err)
	return exitAlreadyPrinted(err, status)
}

// serveBindFailure returns the error for a failed binding check at startup. A token failure is classified as it is at
// runtime (see credentialRejected). A rejection needs a new login, so it keeps the login hint and exits serveBindStatus.
// Any other token failure is temporary: it has no hint, and exits 1, the status of the other failures, since
// serveBindStatus says the credential was rejected.
//
// config.ErrCannotConfigureDefault is also a login failure. The SDK returns it when no strategy configures a credential,
// as for a profile with no auth_type and no cached token, and it drops each strategy's error. It is checked only here,
// since the provider is configured once at startup and the sentinel cannot come from a later request.
func serveBindFailure(cmd *cobra.Command, profileName string, err error) error {
	bindErr, ok := errors.AsType[*mcp.BindError](err)
	if !ok || bindErr.Category != mcp.BindAuthFailed || credentialRejected(bindErr.Unwrap()) || errors.Is(err, config.ErrCannotConfigureDefault) {
		return serveFailure(cmd, err, serveBindStatus)
	}
	temporary := fmt.Errorf("could not get a token for Databricks profile \"%s\", a temporary failure: %w; try again", profileName, bindErr.Unwrap())
	return serveFailure(cmd, temporary, 1)
}

// exitAlreadyPrinted returns err as an error that exits with status and that Execute does not print again.
func exitAlreadyPrinted(err error, status int) error {
	return errors.Join(root.ErrAlreadyPrinted, &root.ExitStatusError{Status: status, Err: err})
}

// refreshGuard wraps the profile's authenticate for the bridge. A rejected token (see credentialRejected) fails that
// request with an error that names the profile and its login command, then stops the proxy, which exits 2. A changed
// binding (see rebindingAuthenticate) stops it the same way, with the error of the binding check. Later requests fail
// at once, without asking for a token again. A transient failure fails only its request, and the next request asks for
// a token again. A request whose own context ended does not stop it, nor does the session DELETE at shutdown, which is
// skipped after a failure.
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
	if bindingChanged(err) {
		// The check failed before any token was fetched, so the error says what changed and how to fix it.
		return g.stop(err)
	}
	if !credentialRejected(err) {
		// A failure that is not a rejection, such as a network blip, says nothing about the credential: a login would
		// not fix it, and the next request asks for a token again, so the message names neither.
		return fmt.Errorf("could not get a token for Databricks profile \"%s\": %w; the next request tries again", g.profile, err)
	}
	// The hint says to restart the server: a new spawn re-runs the binding checks, and it is the one that picks up the
	// new login.
	authErr := fmt.Errorf("%w\nThen restart the MCP server in your agent", mcp.AuthErrorIn(g.profile, g.pinnedFile, err))
	return g.stop(authErr)
}

// stop records err as the first failure of the guard and cancels the proxy, then returns err. A failure after the first
// returns the stopped error and does not cancel again.
func (g *refreshGuard) stop(err error) error {
	g.mu.Lock()
	first := g.err == nil
	if first {
		g.err = err
	}
	g.mu.Unlock()
	if !first {
		return g.stoppedError()
	}
	g.cancel()
	return err
}

// stoppedError is the error for a request that arrives after the failure. It does not repeat the login hint.
func (g *refreshGuard) stoppedError() error {
	return fmt.Errorf("the session is stopping after %s for profile \"%s\"", stopReason(g.failure()), g.profile)
}

// bindingChanged reports whether err is a binding check that failed because the profile no longer names the bound host
// and workspace, or no longer exists in the config file.
func bindingChanged(err error) bool {
	bindErr, ok := errors.AsType[*mcp.BindError](err)
	return ok && (bindErr.Category == mcp.BindProfileHostChanged || bindErr.Category == mcp.BindProfileNotFound)
}

// stopReason says what stopped the proxy, for the messages that report it.
func stopReason(err error) string {
	if bindingChanged(err) {
		return "the binding changed"
	}
	return "a token refresh failure"
}

// failure returns the first rejection, or nil while no token has been rejected.
func (g *refreshGuard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// credentialRejected reports whether a token failure is one that a new login fixes. That is the CLI's U2M reporting an
// invalid refresh token, a missing refresh token or a missing cached token, or a token endpoint that answers 400 or 401
// with the RFC 6749 section 5.2 error invalid_grant or invalid_client. Everything else is transient and returns false:
// network errors, timeouts, 5xx responses, and any error this check cannot classify. A transient failure is retried by
// the next request.
func credentialRejected(err error) bool {
	if _, ok := errors.AsType[*u2m.InvalidRefreshTokenError](err); ok {
		return true
	}
	// No refresh token, or no cached token at all (after an auth logout), is also fixed by a login.
	if errors.Is(err, u2m.ErrMissingRefreshToken) || errors.Is(err, storage.ErrNotFound) {
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
