package proxy

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// User-Agent extras of a proxied session. useragent.FromContext carries cli/<version> but not
// agent/<name>, so Run adds that from useragent.AgentProvider. The extras identify the proxy and
// the calling agent.
const (
	proxyAgentKey  = "aitools-mcp"
	clientAgentKey = "aitools-mcp-client"
)

// Options is one `serve` session. The binding checks (mcp.Bind) have already passed.
type Options struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
	// Endpoint is the service URL, as mcp.ServiceURL builds it.
	Endpoint string
	// Config is the bound profile's config. Its host is named in the instructions, and its
	// workspace ID is sent as the routing header.
	Config *config.Config
	// Authenticate sets the Authorization header on each upstream request.
	Authenticate func(*http.Request) error
	// Profile is the bound profile's name, named in the instructions.
	Profile string
	// Agent is the registry name of the calling agent, or "" when --agent was not given.
	Agent string
	// Transport performs the upstream requests. Nil means http.DefaultTransport, as in Bridge.
	Transport http.RoundTripper
	// Challenge enables sign-in handling for a sign-in-required service. Nil passes everything through.
	Challenge *Challenge
}

// Run bridges opts.In to the service endpoint until In closes or ctx is done. It returns an error
// only when reading In fails.
func Run(ctx context.Context, opts Options) error {
	ctx = useragent.InContext(ctx, proxyAgentKey, "proxy")
	if opts.Agent != "" {
		ctx = useragent.InContext(ctx, clientAgentKey, opts.Agent)
	}
	if agent := useragent.AgentProvider(); agent != "" {
		ctx = useragent.InContext(ctx, useragent.AgentKey, agent)
	}

	headers := map[string]string{"User-Agent": useragent.FromContext(ctx)}
	maps.Copy(headers, auth.WorkspaceIDHeaders(opts.Config))

	// The bridge's diagnostics and the progress of a sign-in are written to the same stderr, from different goroutines.
	stderr := &serialWriter{w: opts.Err}
	b := &Bridge{
		In:           opts.In,
		Out:          opts.Out,
		Err:          stderr,
		Endpoint:     opts.Endpoint,
		Authenticate: opts.Authenticate,
		Headers:      headers,
		Transport:    opts.Transport,
		Instructions: fmt.Sprintf("This server is bound to Databricks workspace %s (CLI profile \"%s\"). Its tools act on that workspace only.", opts.Config.Host, opts.Profile),
	}
	if opts.Challenge != nil {
		// The initialize result must advertise tools/list_changed so an agent re-lists after sign-in.
		b.ForceListChanged = true
		b.Hooks = &signInHooks{c: opts.Challenge, stderr: stderr}
	}
	return b.Run(ctx)
}

// serialWriter writes to w one write at a time, so that the writes of different goroutines do not interleave or race.
type serialWriter struct {
	mu sync.Mutex
	w  io.Writer
}

// Write implements io.Writer.
func (s *serialWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}
