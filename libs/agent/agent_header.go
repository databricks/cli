package agent

import (
	"context"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/httptransport"
	"github.com/databricks/databricks-sdk-go/config"
)

// agentHeaderPrefix starts the User-Agent token the SDK adds for a detected agent, e.g. agent/claude-code.
const agentHeaderPrefix = "agent/"

// ApplyNotAnAgentHeader makes cfg's requests leave the agent token out of the User-Agent header when
// NotAnAgent(ctx) holds. The SDK adds that token from useragent.AgentProvider, which does not read
// DATABRICKS_NOT_AN_AGENT, and it sets the header after the request visitors have run. The transport is the
// only hook that sees the final header, so cfg's transport is wrapped. Without the variable cfg is left alone.
func ApplyNotAnAgentHeader(ctx context.Context, cfg *config.Config) {
	if !NotAnAgent(ctx) {
		return
	}
	cfg.HTTPTransport = &noAgentHeaderTransport{base: cfg.HTTPTransport, cfg: cfg}
}

// noAgentHeaderTransport drops the agent token from User-Agent and passes the request on to base.
type noAgentHeaderTransport struct {
	// base is the transport cfg had. When base is nil, requests go to httptransport.HealthChecked, chosen per
	// request from cfg.InsecureSkipVerify so a skip_verify resolved from the profile applies; it stands in for
	// the SDK's unexported default.
	base http.RoundTripper
	cfg  *config.Config
}

// SkipRetryOnIO forwards the SDK's optional fixture hook from base. The SDK looks for it on cfg.HTTPTransport, so
// without this a fixture transport would lose its skip-retry behaviour behind the wrapper.
func (t *noAgentHeaderTransport) SkipRetryOnIO() bool {
	skippable, ok := t.base.(interface{ SkipRetryOnIO() bool })
	return ok && skippable.SkipRetryOnIO()
}

func (t *noAgentHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := t.base
	if next == nil {
		next = httptransport.HealthChecked(t.cfg.InsecureSkipVerify)
	}
	ua, dropped := withoutAgentToken(req.Header.Get("User-Agent"))
	if !dropped {
		return next.RoundTrip(req)
	}
	// A RoundTripper must not modify the request it is given.
	r := req.Clone(req.Context())
	r.Header.Set("User-Agent", ua)
	return next.RoundTrip(r)
}

// withoutAgentToken removes the agent/<name> tokens from a User-Agent value and reports whether it removed any.
func withoutAgentToken(ua string) (string, bool) {
	fields := strings.Fields(ua)
	kept := make([]string, 0, len(fields))
	for _, f := range fields {
		if !strings.HasPrefix(f, agentHeaderPrefix) {
			kept = append(kept, f)
		}
	}
	return strings.Join(kept, " "), len(kept) != len(fields)
}
