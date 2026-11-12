package aitools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/proxy"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/spf13/cobra"
)

// serveExitCode is the exit status for binding and auth failures, so agents and `list --check`
// can tell them from transport errors.
const serveExitCode = 2

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
	if out := cmd.Flag("output"); out != nil && out.Changed {
		return errors.New("--output does not apply to serve: its stdout carries the MCP protocol")
	}
	if logFile := cmd.Flag("log-file"); logFile != nil && logFile.Value.String() == "stdout" {
		return errors.New("--log-file stdout is not supported by serve: its stdout carries the MCP protocol")
	}
	profileName := profileFlag(cmd)
	if profileName == "" {
		return errors.New("--profile is required")
	}
	if f.service == "" {
		return errors.New("--service is required")
	}
	if f.agent != "" && agents.ByName(f.agent) == nil {
		return fmt.Errorf("unknown agent %q", f.agent)
	}

	state, err := mcp.LoadState(cmd.Context())
	if err != nil {
		return serveFailure(cmd, 1, err)
	}
	cfg, _, err := mcp.Bind(cmd.Context(), mcp.BindOptions{
		Profile: profileName,
		Service: f.service,
		State:   state,
		Stderr:  cmd.ErrOrStderr(),
	})
	if err != nil {
		return serveFailure(cmd, serveExitCode, err)
	}

	// The bound profile authenticates the exit-time telemetry upload.
	cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), cfg))
	// Telemetry for the proxy start (PROXY_STARTED) is logged here by the telemetry change.

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
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
		Agent:        f.agent,
	})
	if err != nil {
		return serveFailure(cmd, 1, err)
	}
	if refreshErr := refresh.failure(); refreshErr != nil {
		return serveFailure(cmd, serveExitCode, refreshErr)
	}
	return nil
}

// serveFailure prints err to stderr and returns the error that sets the exit status. Execute does
// not print it again, so the message is written exactly once.
func serveFailure(cmd *cobra.Command, status int, err error) error {
	fmt.Fprintln(cmd.ErrOrStderr(), err)
	return errors.Join(root.ErrAlreadyPrinted, &root.ExitStatusError{Status: status, Err: err})
}

// refreshGuard wraps the profile's authenticate for the bridge. A token that cannot be refreshed
// mid-session fails that request with an error naming the profile, then stops the proxy, so the
// command exits 2 and the next spawn re-runs the binding checks.
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
	g.mu.Lock()
	if g.err == nil {
		g.err = err
	}
	g.mu.Unlock()
	g.cancel()
	return fmt.Errorf("token refresh failed for profile %s: %w\nRun: databricks auth login --profile %s", g.profile, err, g.profile)
}

// failure returns the first refresh error, or nil when every token was refreshed.
func (g *refreshGuard) failure() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}
