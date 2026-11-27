package aitools

import (
	"slices"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// AutoInstallsFor reports whether the auto-install after cmd may give the
// agent agentName tools (see autoinstall.InstallsFor). It decides as
// autoInstall does up to the gate lookup, which it skips: a run that would
// look the gate up counts as installing, so on a version whose gate is off the
// install hint moves to the end of the command. A machine whose stored ID puts
// it in the holdout never installs. A first install gives tools to the agents
// its plan names; an update gives skills to every agent with a config
// directory when a skill changed, which is only known after the manifest is
// fetched, so it counts as installing for each of them. With
// DATABRICKS_AGENT_FEATURES_DISABLED set, nothing is installed, so it reports
// false, as AutoInstall does.
func AutoInstallsFor(cmd *cobra.Command, agentName string) bool {
	ctx := cmd.Context()
	if skills.AgentFeaturesDisabled(ctx) {
		return false
	}
	in := autoInstallInputs(ctx, skipsAutoInstall(cmd))
	if slices.Contains(stateFreeSkips, autoinstall.Decide(in)) {
		return false
	}
	var err error
	in.State, err = autoinstall.Load(ctx)
	if err != nil || autoinstall.Decide(in) != autoinstall.ReasonGateOff {
		return false
	}
	if in.State.ID != "" && autoinstall.InHoldout(autoinstall.Bucket(in.State.ID)) {
		return false
	}
	tools, err := planTools(quietContext(ctx), in.State)
	if err != nil || tools.skip != "" {
		return false
	}
	if tools.existing != nil {
		return slices.ContainsFunc(agents.DetectInstalled(ctx), func(a *agents.Agent) bool {
			return a.Name == agentName
		})
	}
	for _, it := range tools.plan {
		if it.agent.Name == agentName {
			return slices.Contains(tools.newAgents(), it.agent.DisplayName)
		}
	}
	return false
}
