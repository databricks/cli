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

// tryConfigureAuth resolves auth config onto the context so telemetry can
// upload at command exit. It does not fail if auth is not configured.
func tryConfigureAuth(cmd *cobra.Command, args []string) error {
	ctx := root.SkipPrompt(cmd.Context())
	ctx = root.SkipLoadBundle(ctx)
	cmd.SetContext(ctx)
	_, _ = root.MustAnyClient(cmd, args)
	return nil
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
		// A successful agent leaves errorCategory at its zero value ("", not the
		// TYPE_UNSPECIFIED sentinel), so key on emptiness to drop it here.
		if o.agent == nil || o.errorCategory == "" {
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
