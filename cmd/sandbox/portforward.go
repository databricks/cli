package sandbox

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// portForwardRoute is the workspace route that relays a request to an app
// listening on a sandbox port: <route>/<sandbox-id>/ports/<port>/<path>.
const portForwardRoute = "/sandbox-proxy/sandboxes/"

// portForwardBindHost keeps forwards reachable only from this machine, since
// every request through them carries the caller's credentials.
const portForwardBindHost = "127.0.0.1"

// portForwardReadHeaderTimeout bounds how long a local client may take to send
// request headers. Bodies and upgraded connections are not bounded.
const portForwardReadHeaderTimeout = time.Minute

// databricksHeaderPrefix marks the platform's auth and routing headers. Only
// the CLI's own values may reach the workspace.
const databricksHeaderPrefix = "X-Databricks-"

// portForward is one parsed -L LOCAL:REMOTE spec.
type portForward struct {
	localPort  int
	remotePort int
}

// portForwardTarget is where and how forwarded requests are sent.
type portForwardTarget struct {
	workspace    *url.URL
	sandboxID    string
	authenticate func(*http.Request) error
	headers      map[string]string
	transport    http.RoundTripper
}

func newPortForwardCommand() *cobra.Command {
	var specs []string

	cmd := &cobra.Command{
		Use:   "port-forward [sandbox-id] -L <local-port>:<remote-port>...",
		Short: "Forward local ports to apps in a Sandbox environment",
		Long: `Forward local ports to apps running in a Sandbox environment.

Each -L LOCAL:REMOTE serves http://127.0.0.1:LOCAL and relays HTTP and
WebSocket traffic to an app listening on 0.0.0.0:REMOTE in the sandbox,
authenticated as you. Sandboxes accept remote ports from 10000 to 65535 and
refuse others. Pass 0 as the local port to pick a free one. Runs until
interrupted.

Examples:
  databricks sandbox port-forward -L 8080:10080                         # default sandbox
  databricks sandbox port-forward happy-panda-1234 -L 8080:10080 -L 8081:10081`,
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
			sandboxID, err := resolvePortForwardSandbox(ctx, api, profile, args)
			if err != nil {
				return err
			}

			workspace, err := url.Parse(w.Config.Host)
			if err != nil {
				return fmt.Errorf("failed to parse workspace host %q: %w", w.Config.Host, err)
			}
			transport, err := newPortForwardTransport(w.Config)
			if err != nil {
				return err
			}
			target := portForwardTarget{
				workspace:    workspace,
				sandboxID:    sandboxID,
				authenticate: w.Config.Authenticate,
				headers:      auth.WorkspaceIDHeaders(w.Config),
				transport:    transport,
			}

			ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
			defer stop()
			return servePortForwards(ctx, target, forwards)
		},
	}

	cmd.Flags().StringArrayVarP(&specs, "forward", "L", nil, "Forward LOCAL:REMOTE ports, e.g. 8080:10080 (repeatable)")
	_ = cmd.MarkFlagRequired("forward")

	return cmd
}

// parseForwards parses -L LOCAL:REMOTE specs. LOCAL may be 0 to pick a free
// port. The sandbox enforces which remote ports it accepts.
func parseForwards(specs []string) ([]portForward, error) {
	forwards := make([]portForward, 0, len(specs))
	for _, spec := range specs {
		local, remote, found := strings.Cut(spec, ":")
		localPort, localErr := strconv.Atoi(local)
		remotePort, remoteErr := strconv.Atoi(remote)
		if !found || localErr != nil || remoteErr != nil || localPort < 0 || localPort > 65535 || remotePort < 1 || remotePort > 65535 {
			return nil, fmt.Errorf("invalid -L %q: expected LOCAL:REMOTE ports, e.g. 8080:10080", spec)
		}
		forwards = append(forwards, portForward{localPort: localPort, remotePort: remotePort})
	}
	return forwards, nil
}

// resolvePortForwardSandbox returns the sandbox named in args, or the default
// sandbox when args is empty. Like `sandbox ssh`, it clears a default that no
// longer exists and only warns on other lookup errors, since the forward may
// still work.
func resolvePortForwardSandbox(ctx context.Context, api *sandboxAPI, profile string, args []string) (string, error) {
	var sandboxID string
	if len(args) == 0 {
		if sandboxID = getDefault(ctx, profile); sandboxID == "" {
			return "", errors.New("no default sandbox configured — run `databricks sandbox create` to provision one, or `databricks sandbox default <id>` to point at an existing sandbox")
		}
	} else {
		resolved, err := resolveLocalID(ctx, profile, args[0])
		if err != nil {
			return "", err
		}
		sandboxID = resolved
	}

	// A "." or ".." ID would move every forwarded request off the port-forward route.
	if hasDotSegment("/" + url.PathEscape(sandboxID)) {
		return "", fmt.Errorf("invalid sandbox ID %q", sandboxID)
	}

	if _, err := api.get(ctx, sandboxID); err != nil {
		switch {
		case !errors.Is(err, apierr.ErrNotFound):
			warn(ctx, fmt.Sprintf("Could not validate sandbox %s: %v", sandboxID, err))
		case len(args) == 0:
			_ = clearDefault(ctx, profile)
			return "", fmt.Errorf("saved default %q no longer exists (cleared) — run `databricks sandbox create` to provision a new one, or `databricks sandbox default <id>` to point at an existing sandbox", sandboxID)
		default:
			return "", fmt.Errorf("no sandbox named %q — `databricks sandbox list` shows available IDs", sandboxID)
		}
	}
	return sandboxID, nil
}

// newPortForwardTransport returns the workspace config's transport, or the
// default one with the config's TLS verification setting.
func newPortForwardTransport(cfg *config.Config) (http.RoundTripper, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to configure port forward transport: %w", err)
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

// servePortForwards listens on every forward's local port before serving any,
// so a port that is already in use fails the command up front.
func servePortForwards(ctx context.Context, target portForwardTarget, forwards []portForward) error {
	var lc net.ListenConfig
	listeners := make([]net.Listener, 0, len(forwards))
	for _, f := range forwards {
		ln, err := lc.Listen(ctx, "tcp", net.JoinHostPort(portForwardBindHost, strconv.Itoa(f.localPort)))
		if err != nil {
			for _, open := range listeners {
				open.Close()
			}
			return fmt.Errorf("failed to listen on local port %d: %w", f.localPort, err)
		}
		listeners = append(listeners, ln)
	}

	g, ctx := errgroup.WithContext(ctx)
	for i, f := range forwards {
		srv := &http.Server{
			Handler:           newPortForwardHandler(target, f.remotePort),
			ReadHeaderTimeout: portForwardReadHeaderTimeout,
		}
		cmdio.LogString(ctx, fmt.Sprintf("Forwarding http://%s -> %s:%d", listeners[i].Addr(), target.sandboxID, f.remotePort))
		g.Go(func() error {
			if err := srv.Serve(listeners[i]); !errors.Is(err, http.ErrServerClosed) {
				return fmt.Errorf("port forward on %s failed: %w", listeners[i].Addr(), err)
			}
			return nil
		})
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

// newPortForwardHandler relays each request to remotePort in the sandbox
// through the workspace's port-forward route.
func newPortForwardHandler(target portForwardTarget, remotePort int) http.Handler {
	base := strings.TrimSuffix(target.workspace.Path, "/") + portForwardRoute
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
				if strings.HasPrefix(name, databricksHeaderPrefix) {
					pr.Out.Header.Del(name)
				}
			}
			for name, value := range target.headers {
				pr.Out.Header.Set(name, value)
			}
		},
		Transport: &authenticatingTransport{authenticate: target.authenticate, base: target.transport},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Debugf(r.Context(), "port forward to %s:%d failed: %v", target.sandboxID, remotePort, err)
			http.Error(w, "port forward failed: "+err.Error(), http.StatusBadGateway)
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case !isLoopbackHost(r.Host):
			http.Error(w, "port forward only accepts requests addressed to localhost", http.StatusForbidden)
		case hasDotSegment(r.URL.EscapedPath()):
			http.Error(w, "request path must not contain . or .. segments", http.StatusBadRequest)
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

// hasDotSegment reports whether an escaped URL path contains a "." or ".."
// segment, including percent-encoded and backslash-separated forms. The
// workspace would resolve such a path out of the port-forward route and onto
// another API, with the caller's credentials attached.
func hasDotSegment(escapedPath string) bool {
	decoded, err := url.PathUnescape(escapedPath)
	if err != nil {
		return true
	}
	for _, segment := range strings.FieldsFunc(decoded, isPathSeparator) {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}

func isPathSeparator(r rune) bool {
	return r == '/' || r == '\\'
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
