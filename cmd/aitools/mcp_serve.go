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
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/browser"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/config/credentials"
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
		"Registry name of the calling agent, for the User-Agent and telemetry")
	return cmd
}

// runMcpServe runs the binding checks, then bridges stdio to the service until stdin closes or a
// signal arrives.
func runMcpServe(cmd *cobra.Command, f mcpServeFlags) error {
	start := time.Now()
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
	agent := proxyAgentType(f.agent)
	var authorization string
	cfg, entry, err := mcp.Bind(cmd.Context(), mcp.BindOptions{
		Profile:              profileName,
		Service:              f.service,
		State:                state,
		Stderr:               cmd.ErrOrStderr(),
		CaptureAuthorization: func(header string) { authorization = header },
	})
	if err != nil {
		return serveBindFailure(cmd, profileName, err)
	}

	// A server pinned to a config file names that file in its hints, as add and login do.
	pinnedFile := mcp.PinnedFile(entry, cfg.ConfigFile)
	ctx, stop := serveSignalContext(cmd.Context())
	defer stop()
	// The bridge and the sign-in client share the guard, so a refresh failure in either one stops the proxy.
	// Each token fetch re-checks the binding first; see rebindingAuthenticate.
	check := bindingCheck(cmd.Context(), state, entry, f.service, profileName, cfg)
	authenticate := rebindingAuthenticate(check, cfg.Authenticate)
	refresh := &refreshGuard{authenticate: authenticate, profile: profileName, pinnedFile: pinnedFile, cancel: stop}

	// The sign-in client is built only when serveChallenge reads the service or signs in. A server whose record says it
	// needs no sign-in skips the startup service lookup.
	var w *databricks.WorkspaceClient
	if entry == nil || entry.SignInRequired {
		w, err = guardedClient(cfg, refresh)
		if err != nil {
			return serveFailure(cmd, err, 1)
		}
	}
	challenge, err := serveChallenge(ctx, serveTarget{w: w, entry: entry, profile: profileName, service: f.service, pinnedFile: pinnedFile, agent: agent}, cmd.ErrOrStderr())
	if err != nil {
		// A rejected token stops serve by cancelling ctx, so the refresh failure is checked before the signal, and its
		// message is the one that names the login.
		if refreshErr := refresh.failure(); refreshErr != nil {
			return serveFailure(cmd, refreshErr, serveBindStatus)
		}
		if ctx.Err() != nil {
			// A signal during the start ends serve as a shutdown does, with no line.
			return nil
		}
		// serveChallenge fails only on an auth failure (see lookupAtStart), which exits serveBindStatus.
		return serveFailure(cmd, err, serveBindStatus)
	}
	// Only a serving start is logged, and it is uploaded now rather than at exit: the exit upload cannot refresh an expired
	// token, so a session idle past the token's life would lose its start. The upload authenticates with the profile as it
	// was bound here, with a config of its own that never refreshes a token, since the session's config refreshes its token.
	if logMcpEvent(ctx, proxyStartedEvent(agent, f.service)) {
		setUploadConfig(ctx, profileName, cfg.ConfigFile, cfg, authorization)
		uploadAtStart(ctx, cmd, start)
		if ctx.Err() != nil {
			telemetry.SkipUpload(ctx)
			return nil
		}
	}

	err = serveProxy(ctx, cmd, refresh, proxy.Options{
		In:       cmd.InOrStdin(),
		Out:      cmd.OutOrStdout(),
		Err:      cmd.ErrOrStderr(),
		Endpoint: mcp.ServiceURL(cfg.Host, f.service),
		Config:   cfg,
		Profile:  profileName,
		Agent:    serveAgent(f.agent, cmd.ErrOrStderr()),
		Transport: &signInRedirect{
			next:       newUpstreamLog(serveTransport(cfg)),
			profile:    profileName,
			configFile: mcp.PinnedFile(entry, cfg.ConfigFile),
		},
		Challenge: challenge,
	})
	if !telemetry.Disabled(ctx) && len(telemetry.BufferedEvents(ctx)) > 0 {
		dropUploadIfRebound(cmd.Context(), check, profileName)
	}
	return err
}

// dropUploadIfRebound drops exit telemetry if the profile no longer matches
// the workspace that serve bound. It uses the same check as each token fetch.
func dropUploadIfRebound(ctx context.Context, check func() error, profileName string) {
	if err := checkBindingWithRetry(check); err != nil {
		log.Debugf(ctx, "telemetry: profile %s no longer matches the binding serve started with; not uploading telemetry", profileName)
		telemetry.SkipUpload(ctx)
	}
}

// uploadAtStart uploads the events logged so far, which at serve's start is its PROXY_STARTED. The upload waits for the
// service, bounded by its timeout, so the proxy starts after it. A failed upload keeps its events for the exit upload.
// The run is still going, so its exit code is 0.
func uploadAtStart(ctx context.Context, cmd *cobra.Command, start time.Time) {
	err := telemetry.Upload(ctx, root.TelemetryExecutionContext(cmd, start, 0))
	if err != nil {
		log.Debugf(ctx, "telemetry: upload at start failed: %s", err)
	}
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

// guardedClient returns the client of the bound profile whose requests the refresh guard authenticates. The SDK calls
// the guard for each request the client makes, so a token that cannot be refreshed stops the proxy from any of them. The
// profile's config is read again for the client, since its own credentials are the guard's.
func guardedClient(cfg *config.Config, refresh *refreshGuard) (*databricks.WorkspaceClient, error) {
	guarded, err := mcp.BoundConfig(cfg.Profile, cfg.ConfigFile)
	if err != nil {
		return nil, err
	}
	// The "none" sentinel is the CLI's, not a workspace ID, so the SDK must not send it as the routing header.
	if guarded.WorkspaceID == profile.WorkspaceIDNone {
		guarded.WorkspaceID = ""
	}
	guarded.Credentials = guardCredentials{refresh: refresh}
	return databricks.NewWorkspaceClient((*databricks.Config)(guarded))
}

// guardCredentials is the credentials strategy of the sign-in client: each request it makes is authenticated by the
// refresh guard.
type guardCredentials struct {
	refresh *refreshGuard
}

// Name implements config.CredentialsStrategy.
func (guardCredentials) Name() string {
	return "aitools-mcp"
}

// Configure implements config.CredentialsStrategy.
func (g guardCredentials) Configure(context.Context, *config.Config) (credentials.CredentialsProvider, error) {
	return credentials.CredentialsProviderFn(g.refresh.authenticateCall), nil
}

// serveTarget is the server that a serve session bridges: its service, the bound profile, the install record, the config
// file it is pinned to, and the agent that started it. w is the profile's client, which must be guardedClient's.
type serveTarget struct {
	// w is nil when entry says the service needs no sign-in. serveChallenge returns before it would use it then.
	w          *databricks.WorkspaceClient
	entry      *mcp.Entry
	profile    string
	service    string
	pinnedFile string
	agent      protos.AitoolsAgentType
}

// serveChallenge returns the sign-in handling of a service that needs provider sign-in, or nil for
// a pure passthrough. The state entry decides when there is one; otherwise the service is read once. A read that fails
// for a reason other than the credential does not stop the serve: the sign-in handling is on then, since it acts only on
// a challenge that names the service, and the failure is written to errOut.
func serveChallenge(ctx context.Context, t serveTarget, errOut io.Writer) (*proxy.Challenge, error) {
	signInRequired := t.entry != nil && t.entry.SignInRequired
	if t.entry == nil {
		svc, err := lookupAtStart(ctx, t.w, t.profile, t.pinnedFile, t.service)
		switch {
		case err == nil:
			signInRequired = svc.SignInRequired()
		case authFailed(err):
			return nil, err
		case ctx.Err() != nil:
			// The serve is stopping, so the read's failure is its cancellation, which is not reported.
			return nil, ctx.Err()
		default:
			fmt.Fprintf(errOut, "%v; serving it with sign-in handling on\n", err)
			signInRequired = true
		}
	}
	if !signInRequired {
		return nil, nil
	}
	// The record holds the workspace the profile was bound to, which the binding check keeps it on. A server with no record
	// has no workspace ID that is known without a request, so its challenges match the profile's host only.
	var workspaceID string
	if t.entry != nil {
		workspaceID = t.entry.WorkspaceID
	}
	return &proxy.Challenge{
		Service:       t.service,
		Profile:       t.profile,
		Host:          t.w.Config.Host,
		WorkspaceID:   workspaceID,
		Label:         mcp.ProviderLabel(t.service),
		LoginCommand:  mcpLoginHint(t.service, t.profile, t.pinnedFile),
		LogoutCommand: mcpLogoutHint(t.service, t.profile, t.pinnedFile),
		PinnedFile:    t.pinnedFile,
		SignIn:        serveSignIn(t),
		Recorder:      mcpProxyTelemetry{agent: t.agent, service: t.service},
	}, nil
}

// lookupAtStart reads a service for a server with no install record. The read is bounded by profileCheckTimeout, as the
// checks of list --check are, so that a host that does not answer does not hold up the start for the SDK's retry budget.
func lookupAtStart(ctx context.Context, w *databricks.WorkspaceClient, profileName, pinnedFile, service string) (mcp.Service, error) {
	ctx, cancel := context.WithTimeout(ctx, profileCheckTimeout)
	defer cancel()
	svc, err := mcp.GetService(ctx, w, service)
	if err != nil {
		// A credential the workspace rejects gets its sign-in page instead of the service's answer, so that is an auth failure.
		if errors.Is(err, httpclient.ErrHTMLContent) {
			cause := fmt.Errorf("cannot read MCP service %s: the workspace answered with a sign-in page instead of JSON", service)
			return mcp.Service{}, mcp.AuthErrorIn(profileName, pinnedFile, cause)
		}
		err = describeRequestError(profileName, pinnedFile, err)
		return mcp.Service{}, fmt.Errorf("cannot read MCP service %s: %w", service, err)
	}
	return svc, nil
}

// serveOpenPage opens the sign-in page for serve. Its stdin is the protocol stream, so the browser command gets none. Tests
// replace it.
var serveOpenPage = browser.OpenOutputToStderrWithoutStdin

// serveSignIn is the authenticate tool's sign-in, the one `login` runs. Its progress goes to the request's stderr, which the
// proxy serializes, since stdout carries the protocol. The page opens only when req.Open is set, so the proxy decides how
// often the browser opens. w must be guardedClient's.
func serveSignIn(t serveTarget) proxy.SignInFunc {
	w, profileName, service, pinnedFile := t.w, t.profile, t.service, t.pinnedFile
	// The record holds the workspace the profile was bound to, which the binding check keeps it on, so its ID needs no
	// request. Without one, the first sign-in that needs it resolves it; the proxy runs one sign-in at a time.
	var workspaceID string
	if t.entry != nil && t.entry.WorkspaceID != profile.WorkspaceIDNone {
		workspaceID = t.entry.WorkspaceID
	}
	return func(ctx context.Context, req proxy.SignInRequest) (bool, bool, error) {
		if workspaceID == "" {
			id, err := auth.ResolveWorkspaceID(ctx, w)
			if err != nil {
				return false, false, describeRequestError(profileName, pinnedFile, fmt.Errorf("cannot read the workspace ID of %s: %w", w.Config.Host, err))
			}
			workspaceID = id
		}
		url, err := mcp.SignInURL(w.Config.Host, service, workspaceID)
		if err != nil {
			return false, false, err
		}
		// The page opens only when req.Open is set. Otherwise the sign-in only waits, and the proxy's progress names the page.
		var openPage func(context.Context, string) error
		if req.Open {
			// The browser command must outlive the request: the request's context would kill it when the call is cancelled, and
			// with it the page the user is on.
			openPage = func(pageCtx context.Context, url string) error {
				return serveOpenPage(context.WithoutCancel(pageCtx), url)
			}
		}
		svc := mcp.Service{FullName: service, Known: mcp.KnownOf(service)}
		_, alreadySignedIn, urlShown, err := signIn(ctx, req.Stderr, w, pinnedFile, url, svc, signInOptions{Profile: profileName, Open: openPage, OnPoll: func() { req.Progress(url) }})
		if err != nil {
			return false, urlShown, err
		}
		// The note is printed here, as login prints it, so the authenticate tool's sign-in shows it too.
		if svc.Known != nil && svc.Known.Note != "" {
			fmt.Fprintln(req.Stderr, svc.Known.Note)
		}
		return alreadySignedIn, urlShown, nil
	}
}

// serveProxy bridges opts to the service until stdin closes or ctx is done. The requests are authenticated by refresh, so
// that a refresh failure or a changed binding stops the proxy; see proxyResult.
func serveProxy(ctx context.Context, cmd *cobra.Command, refresh *refreshGuard, opts proxy.Options) error {
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

// authFailed reports whether err is a credential the profile cannot provide.
func authFailed(err error) bool {
	be, ok := errors.AsType[*mcp.BindError](err)
	return ok && be.Category == mcp.BindAuthFailed
}

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

// refreshGuard wraps the profile's authenticate for the bridge and for the sign-in client. A rejected token (see
// credentialRejected) fails that request with an error that names the profile and its login command, then stops the
// proxy, which exits 2. A changed binding (see rebindingAuthenticate) stops it the same way, with the error of the binding
// check. Later requests fail at once, without asking for a token again. A transient failure fails only its request, and
// the next request asks for a token again. A request whose own context ended does not stop it, nor does the bridge's
// session DELETE at shutdown, which is skipped after a failure.
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
	if err == nil || r.Context().Err() != nil || r.Method == http.MethodDelete {
		return err
	}
	return g.fail(err)
}

// authenticateCall is the sign-in client's authentication. Unlike authenticateRequest it does not skip a DELETE, since the
// sign-in's own DELETE is a call that needs its credential.
func (g *refreshGuard) authenticateCall(r *http.Request) error {
	if g.failure() != nil {
		return g.stoppedError()
	}
	err := g.authenticate(r)
	if err == nil || r.Context().Err() != nil {
		return err
	}
	return g.fail(err)
}

// fail records err, a token that cannot be refreshed or a changed binding, and returns the error for the request that hit
// it. The first failure stops the proxy. The hint says to restart the server: a new spawn re-runs the binding checks, and
// it is the one that picks up the new login.
func (g *refreshGuard) fail(err error) error {
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
