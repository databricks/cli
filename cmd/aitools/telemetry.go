package aitools

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

// telemetryAuthBudget bounds how long after its start a command waits for the
// background auth resolution its telemetry upload needs. A resolution still
// unfinished when the run ends and the budget has passed drops the events.
// Matches the upload's own default timeout.
var telemetryAuthBudget = 3 * time.Second

// resolveTelemetryAuthFn resolves auth onto cmd's context. A seam for tests.
var resolveTelemetryAuthFn = func(cmd *cobra.Command, args []string) error {
	_, err := root.MustAnyClient(cmd, args)
	return err
}

// withTelemetryAuth makes cmd resolve auth best-effort so its telemetry can
// upload at exit: PreRunE starts the resolution in the background and RunE's
// wrapper waits for it within telemetryAuthBudget. Nothing is resolved when
// telemetry is disabled. The command never fails or prompts because of it.
func withTelemetryAuth(cmd *cobra.Command) {
	var pending *pendingAuth
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		pending = startTelemetryAuth(cmd, args)
		return nil
	}
	runE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		err := runE(cmd, args)
		pending.apply(cmd)
		return err
	}
}

// pendingAuth is a background auth resolution started by startTelemetryAuth.
type pendingAuth struct {
	done     chan struct{}
	deadline time.Time
	cfg      *config.Config
}

// startTelemetryAuth resolves auth in the background on a shadow command. The
// shadow shares cmd's flags (for --profile) but not its context, so a
// resolution that outlives the budget never touches cmd's context. It logs at
// Debug only, so its discarded errors do not interleave with the command's
// output and prompts. It never refreshes an OAuth token (root.SkipTokenRefresh),
// as the command may exit before a refresh finishes; an expired cached token
// fails the resolution and drops the events.
func startTelemetryAuth(cmd *cobra.Command, args []string) *pendingAuth {
	ctx := cmd.Context()
	if telemetry.Disabled(ctx) {
		return nil
	}
	sctx := log.NewContext(ctx, slog.New(debugOnlyHandler{log.GetLogger(ctx).Handler()}))
	shadow := &cobra.Command{Use: cmd.Use}
	shadow.Flags().AddFlagSet(cmd.Flags())
	shadow.SetContext(root.SkipTokenRefresh(root.SkipLoadBundle(root.SkipPrompt(sctx))))

	p := &pendingAuth{done: make(chan struct{}), deadline: time.Now().Add(telemetryAuthBudget)}
	resolve := resolveTelemetryAuthFn
	go func() {
		defer close(p.done)
		defer func() {
			if r := recover(); r != nil {
				log.Debugf(sctx, "auth for telemetry panicked: %v", r)
			}
		}()
		if err := resolve(shadow, args); err != nil {
			log.Debugf(sctx, "auth for telemetry failed: %v", err)
			return
		}
		if resolved := shadow.Context(); cmdctx.HasConfigUsed(resolved) {
			p.cfg = cmdctx.ConfigUsed(resolved)
		}
	}()
	return p
}

// apply waits for the resolution until the deadline and, if it finished, puts
// the resolved config on cmd's context for the upload. A run that logged no
// event has nothing to upload, so it does not wait. Only the config is
// copied: the upload needs nothing else, and a workspace client on the context
// would change how root decorates the command's own errors.
func (p *pendingAuth) apply(cmd *cobra.Command) {
	if p == nil {
		return
	}
	if len(telemetry.BufferedEvents(cmd.Context())) == 0 {
		return
	}
	if !p.wait() {
		log.Debugf(cmd.Context(), "auth for telemetry not resolved within %s; not uploading telemetry", telemetryAuthBudget)
		return
	}
	if p.cfg != nil {
		cmd.SetContext(cmdctx.SetConfigUsed(cmd.Context(), p.cfg))
	}
}

// wait reports whether the resolution finished by the deadline. A resolution
// that finished before the wait starts always counts: once a command outlasts
// the budget, a select on both channels would pick between them at random.
func (p *pendingAuth) wait() bool {
	select {
	case <-p.done:
		return true
	default:
	}
	timer := time.NewTimer(time.Until(p.deadline))
	defer timer.Stop()
	select {
	case <-p.done:
		return true
	case <-timer.C:
		return false
	}
}

// debugOnlyHandler logs records above Debug (e.g. hostmetadata's "failed to
// fetch host metadata" Warn) at Debug, so they show only with --debug.
type debugOnlyHandler struct{ slog.Handler }

func debugOnly(level slog.Level) slog.Level {
	return min(level, log.LevelDebug)
}

func (h debugOnlyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, debugOnly(level))
}

func (h debugOnlyHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Level = debugOnly(r.Level)
	return h.Handler.Handle(ctx, r)
}

func (h debugOnlyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return debugOnlyHandler{h.Handler.WithAttrs(attrs)}
}

func (h debugOnlyHandler) WithGroup(name string) slog.Handler {
	return debugOnlyHandler{h.Handler.WithGroup(name)}
}

// installOpts is the subset of install options telemetry records, kept narrow
// so every field that leaves the machine is visible here.
type installOpts struct {
	Scope        string
	Experimental bool
}

// logInstallEvent buffers an install event; cmd/root uploads it at exit.
// errCategory is the top-level command outcome (Unspecified on success or when
// the failure is reported per agent), and outcomes carries the per-agent results
// so a skipped-with-warning failure is still recorded even when the command
// exits 0.
func logInstallEvent(ctx context.Context, plan []agentPlanItem, opts installOpts, errCategory protos.AitoolsErrorCategory, outcomes []agentOutcome) {
	telemetry.Log(ctx, protos.DatabricksCliLog{
		AitoolsInstallEvent: &protos.AitoolsInstallEvent{
			Agents:        agentsField(plan),
			Scope:         scopeType(opts.Scope),
			Experimental:  opts.Experimental,
			ErrorCategory: errCategory,
			AgentResults:  agentResultsField(outcomes),
		},
	})
}

// agentResultsField returns the per-agent failure/skip categories, one entry per
// non-successful agent, sorted by agent enum for stable output. Successful
// agents produce no entry.
func agentResultsField(outcomes []agentOutcome) []protos.AitoolsAgentResult {
	var out []protos.AitoolsAgentResult
	for _, o := range outcomes {
		// Successful agents produce no entry; only failed/skipped outcomes carry a
		// category. Keying on status (rather than errorCategory == "") avoids a trap
		// where a future Unspecified category on a success would emit a bogus entry.
		if o.status == outcomeInstalled {
			continue
		}
		out = append(out, protos.AitoolsAgentResult{
			Agent:         agentType(o.agent.Name),
			ErrorCategory: o.errorCategory,
		})
	}
	slices.SortFunc(out, func(a, b protos.AitoolsAgentResult) int {
		return cmp.Compare(a.Agent, b.Agent)
	})
	return out
}

// agentsField returns the deduped agent enums from the plan, sorted so the
// same set of agents always produces the same array on the analytics side.
func agentsField(plan []agentPlanItem) []protos.AitoolsAgentType {
	if len(plan) == 0 {
		return nil
	}
	out := make([]protos.AitoolsAgentType, 0, len(plan))
	seen := make(map[protos.AitoolsAgentType]struct{}, len(plan))
	for _, it := range plan {
		if it.agent == nil {
			continue
		}
		t := agentType(it.agent.Name)
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	slices.Sort(out)
	return out
}

// agentType maps a CLI agent registry name to its telemetry enum. Every agent
// in agents.Registry must have a case here; TestAgentTypeCoversRegistry fails
// if one is missing. The default maps genuinely unknown names to Unspecified
// so an older proto never drops an install.
func agentType(name string) protos.AitoolsAgentType {
	switch name {
	case agents.NameClaudeCode:
		return protos.AitoolsAgentTypeClaudeCode
	case agents.NameCursor:
		return protos.AitoolsAgentTypeCursor
	case agents.NameCodex:
		return protos.AitoolsAgentTypeCodex
	case agents.NameOpenCode:
		return protos.AitoolsAgentTypeOpenCode
	case agents.NameCopilot:
		return protos.AitoolsAgentTypeCopilot
	case agents.NameAntigravity:
		return protos.AitoolsAgentTypeAntigravity
	case agents.NamePi:
		return protos.AitoolsAgentTypePi
	case agents.NameGemini:
		return protos.AitoolsAgentTypeGemini
	case agents.NameGoose:
		return protos.AitoolsAgentTypeGoose
	case agents.NameKiro:
		return protos.AitoolsAgentTypeKiro
	default:
		return protos.AitoolsAgentTypeUnspecified
	}
}

// scopeType maps a resolved install scope to its telemetry enum.
func scopeType(scope string) protos.AitoolsInstallScope {
	switch scope {
	case installer.ScopeGlobal:
		return protos.AitoolsInstallScopeGlobal
	case installer.ScopeProject:
		return protos.AitoolsInstallScopeProject
	default:
		return protos.AitoolsInstallScopeUnspecified
	}
}
