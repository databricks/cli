package aitools

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// installOpts is the subset of install options telemetry records, kept narrow
// so every field that leaves the machine is visible here.
type installOpts struct {
	Scope        string
	Experimental bool
	Trigger      protos.AitoolsInstallTrigger
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
			Trigger:       opts.Trigger,
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
			Agent:         root.AgentType(o.agent.Name),
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
	targets := make([]*agents.Agent, 0, len(plan))
	for _, it := range plan {
		targets = append(targets, it.agent)
	}
	return agentTypes(targets)
}

// agentTypes returns the deduped, sorted agent enums of targets.
func agentTypes(targets []*agents.Agent) []protos.AitoolsAgentType {
	if len(targets) == 0 {
		return nil
	}
	out := make([]protos.AitoolsAgentType, 0, len(targets))
	seen := make(map[protos.AitoolsAgentType]struct{}, len(targets))
	for _, a := range targets {
		if a == nil {
			continue
		}
		t := root.AgentType(a.Name)
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	slices.Sort(out)
	return out
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

// instrumentCommand makes an aitools subcommand log its STARTED and FINISHED
// funnel events, and resolves auth best-effort so they can upload at exit.
// The end of a run does not wait for an unfinished resolution (the events
// upload only if it has already finished) unless RunE calls
// waitForTelemetryAuth, once the run gets past its cheap early exits.
func instrumentCommand(cmd *cobra.Command, command protos.AitoolsCommand) *cobra.Command {
	runE := withCommandFunnel(command, cmd.RunE)
	var auth *root.TelemetryAuth
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), telemetryAuthKey{}, auth))
		return runE(cmd, args)
	}
	auth = root.WithTelemetryAuth(cmd)
	return cmd
}

type telemetryAuthKey struct{}

// waitForTelemetryAuth makes the end of an instrumented run wait for the auth
// resolution within TelemetryAuthBudget, as install does past its plan
// confirmation. Only the RunE of a command wrapped by instrumentCommand calls
// it.
func waitForTelemetryAuth(ctx context.Context) {
	ctx.Value(telemetryAuthKey{}).(*root.TelemetryAuth).WaitAtExit = true
}
