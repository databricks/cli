package aitools

import (
	"slices"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/spf13/cobra"
)

// AutoInstallsFor reports whether the auto-install after cmd is due to give
// the agent agentName its first tools (see autoinstall.InstallsFor). It decides
// as autoInstall does up to the gate lookup, which it skips: a run that would
// look the gate up counts as installing, so on a version whose gate is off the
// install hint waits one command.
func AutoInstallsFor(cmd *cobra.Command, agentName string) bool {
	ctx := cmd.Context()
	in := autoInstallInputs(ctx, skipsAutoInstall(cmd))
	if slices.Contains(stateFreeSkips, autoinstall.Decide(in)) {
		return false
	}
	var err error
	in.State, err = autoinstall.Load(ctx)
	if err != nil || autoinstall.Decide(in) != autoinstall.ReasonGateOff {
		return false
	}
	tools, err := planTools(quietContext(ctx), in.State)
	if err != nil {
		return false
	}
	for _, it := range tools.plan {
		if it.agent.Name == agentName {
			return slices.Contains(tools.newAgents(), it.agent.DisplayName)
		}
	}
	return false
}
