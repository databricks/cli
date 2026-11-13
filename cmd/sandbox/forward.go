package sandbox

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// forwardRoute is the workspace route that relays a request to an app
// listening on a sandbox port: <route>/<sandbox-id>/ports/<port>/<path>.
const forwardRoute = "/sandbox-proxy/sandboxes/"

// Forwards listen only on loopback, since every request through them carries
// the caller's credentials. Some clients (e.g. Node 17+) resolve localhost to
// ::1 first, so the IPv6 loopback is served too when it is available.
const (
	forwardBindHost     = "127.0.0.1"
	forwardBindHostIPv6 = "::1"
)

// The sandbox only forwards to remote ports in this range.
const (
	minRemotePort = 10000
	maxRemotePort = 65535
)

// forwardReadHeaderTimeout bounds how long a local client may take to send
// request headers. Bodies and upgraded connections are not bounded.
const forwardReadHeaderTimeout = time.Minute

// platformHeaderPrefixes mark the platform's auth, identity, and routing
// headers, in lowercase. Only the CLI's own values may reach the workspace.
var platformHeaderPrefixes = []string{"x-databricks-", "s2s-"}

// forwardSpec is one parsed -L LOCAL:REMOTE spec.
type forwardSpec struct {
	localPort  int
	remotePort int
}

// forwardTarget is where and how forwarded requests are sent.
type forwardTarget struct {
	workspace    *url.URL
	sandboxID    string
	authenticate func(*http.Request) error
	headers      map[string]string
	transport    http.RoundTripper
}

func newForwardCommand() *cobra.Command {
	var specs []string

	cmd := &cobra.Command{
		Use:   "forward [sandbox-id] -L <local-port>:<remote-port>...",
		Short: "Forward local ports to apps in a Sandbox environment",
		Long: `Forward local ports to apps running in a Sandbox environment.

Each -L LOCAL:REMOTE serves http://127.0.0.1:LOCAL and relays HTTP and
WebSocket traffic to an app listening on 0.0.0.0:REMOTE in the sandbox,
authenticated as you. Remote ports must be between 10000 and 65535. Pass 0
as the local port to pick a free one. A stopped sandbox is started first.
Runs until interrupted.

Examples:
  databricks sandbox forward -L 8080:10080                         # default sandbox
  databricks sandbox forward happy-panda-1234 -L 8080:10080 -L 8081:10081`,
		Args:              cobra.MaximumNArgs(1),
		PreRunE:           root.MustWorkspaceClient,
		ValidArgsFunction: completeSandboxIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			forwards, err := parseForwards(specs)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			w := cmdctx.WorkspaceClient(ctx)

			profile := w.Config.Profile
			if profile == "" {
				profile = w.Config.Host
			}

			api, err := newSandboxAPI(w)
			if err != nil {
				return err
			}
			var arg string
			if len(args) > 0 {
				arg = args[0]
			}
			sandboxID, sb, err := resolveSandbox(ctx, api, profile, arg)
			if err != nil {
				return err
			}
			if sb != nil {
				if _, err := ensureRunning(ctx, api, sandboxID, sb.Status); err != nil {
					return err
				}
			}

			workspace, err := url.Parse(w.Config.Host)
			if err != nil {
				return fmt.Errorf("failed to parse workspace host %q: %w", w.Config.Host, err)
			}
			transport, err := newForwardTransport(w.Config)
			if err != nil {
				return err
			}
			target := forwardTarget{
				workspace:    workspace,
				sandboxID:    sandboxID,
				authenticate: w.Config.Authenticate,
				headers:      api.headers(),
				transport:    transport,
			}

			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()
			return serveForwards(ctx, target, forwards)
		},
	}

	cmd.Flags().StringArrayVarP(&specs, "forward", "L", nil, "Forward LOCAL:REMOTE ports, e.g. 8080:10080 (repeatable)")
	_ = cmd.MarkFlagRequired("forward")

	return cmd
}

// parseForwards parses -L LOCAL:REMOTE specs. LOCAL may be 0 to pick a free
// port. A few platform ports inside the remote range are refused by the sandbox.
func parseForwards(specs []string) ([]forwardSpec, error) {
	forwards := make([]forwardSpec, 0, len(specs))
	for _, spec := range specs {
		local, remote, found := strings.Cut(spec, ":")
		localPort, localErr := strconv.Atoi(local)
		remotePort, remoteErr := strconv.Atoi(remote)
		if !found || localErr != nil || remoteErr != nil || localPort < 0 || localPort > 65535 {
			return nil, fmt.Errorf("invalid -L %q: expected LOCAL:REMOTE ports, e.g. 8080:10080", spec)
		}
		if remotePort < minRemotePort || remotePort > maxRemotePort {
			return nil, fmt.Errorf("invalid -L %q: remote port must be between %d and %d", spec, minRemotePort, maxRemotePort)
		}
		forwards = append(forwards, forwardSpec{localPort: localPort, remotePort: remotePort})
	}
	return forwards, nil
}

// newForwardTransport returns the workspace config's transport, or the
// default one with the config's TLS verification setting.
func newForwardTransport(cfg *config.Config) (http.RoundTripper, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure forward transport: %w", err)
	}
	if clientCfg.Transport != nil {
		return clientCfg.Transport, nil
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if clientCfg.InsecureSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return transport, nil
}

// serveForwards listens on every forward's local port before serving any,
// so a port that is already in use fails the command up front.
func serveForwards(ctx context.Context, target forwardTarget, forwards []forwardSpec) error {
	var lc net.ListenConfig
	listeners := make([][]net.Listener, 0, len(forwards))
	for _, f := range forwards {
		ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(forwardBindHost, strconv.Itoa(f.localPort)))
		if err != nil {
			for _, group := range listeners {
				for _, open := range group {
					open.Close()
				}
			}
			return fmt.Errorf("failed to listen on local port %d: %w", f.localPort, err)
		}
		group := []net.Listener{ln}
		port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
		if ln6, err := lc.Listen(ctx, "tcp", net.JoinHostPort(forwardBindHostIPv6, port)); err == nil {
			group = append(group, ln6)
		} else {
			log.Debugf(ctx, "not forwarding [%s]:%s: %v", forwardBindHostIPv6, port, err)
		}
		listeners = append(listeners, group)
	}

	// net/http reports routine client disconnects through ErrorLog; keep them at
	// debug level in the CLI log instead of on stderr.
	logger := log.GetLogger(ctx)
	g, ctx := errgroup.WithContext(ctx)
	for i, f := range forwards {
		srv := &http.Server{
			Handler:           newForwardHandler(target, f.remotePort, logger),
			ReadHeaderTimeout: forwardReadHeaderTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelDebug),
		}
		cmdio.LogString(ctx, fmt.Sprintf("Forwarding http://%s -> %s:%d", listeners[i][0].Addr(), target.sandboxID, f.remotePort))
		for _, ln := range listeners[i] {
			g.Go(func() error {
				if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
					return fmt.Errorf("forward on %s failed: %w", ln.Addr(), err)
				}
				return nil
			})
		}
		// An interactive interrupt stops at once: WebSockets and streams never
		// go idle, so a graceful Shutdown would only delay the exit.
		g.Go(func() error {
			<-ctx.Done()
			return srv.Close()
		})
	}
	cmdio.LogString(ctx, "Press Ctrl-C to stop.")
	return g.Wait()
}

// newForwardHandler relays each request to remotePort in the sandbox
// through the workspace's forward route.
func newForwardHandler(target forwardTarget, remotePort int, logger *slog.Logger) http.Handler {
	base := strings.TrimSuffix(target.workspace.Path, "/") + forwardRoute
	port := "/ports/" + strconv.Itoa(remotePort)
	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = target.workspace.Scheme
			pr.Out.URL.Host = target.workspace.Host
			pr.Out.URL.Path = base + target.sandboxID + port + pr.In.URL.Path
			pr.Out.URL.RawPath = base + url.PathEscape(target.sandboxID) + port + pr.In.URL.EscapedPath()
			pr.Out.Host = ""
			pr.Out.Header.Del("Authorization")
			for name := range pr.Out.Header {
				if isPlatformHeader(name) {
					pr.Out.Header.Del(name)
				}
			}
			for name, value := range target.headers {
				pr.Out.Header.Set(name, value)
			}
		},
		Transport: &authenticatingTransport{authenticate: target.authenticate, base: target.transport},
		ErrorLog:  slog.NewLogLogger(logger.Handler(), slog.LevelDebug),
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Debugf(r.Context(), "forward to %s:%d failed: %v", target.sandboxID, remotePort, err)
			http.Error(w, "forward failed: "+err.Error(), http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case !isLoopbackHost(r.Host):
			http.Error(w, "forward only accepts requests addressed to localhost", http.StatusForbidden)
		case isCrossSiteRequest(r):
			http.Error(w, "forward does not accept cross-site browser requests", http.StatusForbidden)
		case isAmbiguousPath(r.URL.EscapedPath()):
			http.Error(w, "forward does not accept . or .. path segments, malformed escapes, or segments containing ;, \\, control characters, or an encoded /, \\, %, ?, or #", http.StatusBadRequest)
		default:
			proxy.ServeHTTP(w, r)
		}
	})
}

// isLoopbackHost reports whether a Host header names this machine. A
// DNS-rebinding page in a browser sends its own hostname, so refusing other
// hosts keeps it from acting with the caller's credentials.
func isLoopbackHost(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// isCrossSiteRequest reports whether a browser sent the request from a page
// outside this machine. WebSocket handshakes bypass CORS, so this is their only
// guard. Pages from any loopback origin, such as a local frontend, are allowed:
// browsers treat localhost and 127.0.0.1 as different sites, so their Origin
// decides rather than Sec-Fetch-Site. Requests without an Origin fall back to
// Sec-Fetch-Site, except that top-level GET navigations from other sites are
// allowed, as in the SDK, so links and sign-in redirects to the app work.
func isCrossSiteRequest(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		return err != nil || !isLoopbackHost(u.Host)
	}
	if r.Header.Get("Sec-Fetch-Site") != "cross-site" {
		return false
	}
	navigation := (r.Method == http.MethodGet || r.Method == http.MethodHead) &&
		r.Header.Get("Sec-Fetch-Mode") == "navigate" &&
		r.Header.Get("Sec-Fetch-Dest") == "document"
	return !navigation
}

// isAmbiguousPath reports whether a proxy on the way to the workspace could
// resolve a segment of escapedPath as "." or "..", moving the request off the
// forward route with the caller's credentials. Proxies also resolve forms
// such as a dot segment with path parameters, so only segments that decode
// once to plain text other than "." and ".." are accepted.
func isAmbiguousPath(escapedPath string) bool {
	for segment := range strings.SplitSeq(escapedPath, "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, `;/\%?#`) || strings.ContainsFunc(decoded, unicode.IsControl) {
			return true
		}
	}
	return false
}

// isPlatformHeader reports whether name starts with one of
// platformHeaderPrefixes, in any case.
func isPlatformHeader(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range platformHeaderPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// authenticatingTransport adds the CLI's credentials to each outgoing request,
// so OAuth tokens refresh during a long-running forward.
type authenticatingTransport struct {
	authenticate func(*http.Request) error
	base         http.RoundTripper
}

func (t *authenticatingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the request it is given.
	req = req.Clone(req.Context())
	if err := t.authenticate(req); err != nil {
		return nil, fmt.Errorf("failed to authenticate with the workspace: %w", err)
	}
	return t.base.RoundTrip(req)
}
