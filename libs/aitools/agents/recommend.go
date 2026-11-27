package agents

import (
	"context"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/skills"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// skillsTipText is the tip printed by PrintSkillsTip and MaybeVersionCheckTip.
const skillsTipText = "Tip: coding agents detected without Databricks skills. Run 'databricks aitools install' to install them."

// PrintSkillsTip prints the non-interactive `apps init` tip to install skills
// and records that it was shown. Callers decide whether the tooling is missing
// (see HasDatabricksToolingInstalled). It prints nothing when
// DATABRICKS_AGENT_FEATURES_DISABLED is set.
func PrintSkillsTip(ctx context.Context) {
	if skills.AgentFeaturesDisabled(ctx) {
		return
	}
	cmdio.LogString(ctx, skillsTipText)
	logHintShown(ctx, protos.AitoolsHintKindAppsInitTip)
}

// MaybeVersionCheckTip prints skillsTipText after `databricks version --check`
// when a coding agent is detected without Databricks tooling. Callers that
// MaybeHint hints (Claude Code and the registry agents, see isHintedAgent) are
// skipped: `version` is in hintExemptCommands, and the tip must not undo that
// exemption for them. installedPlugins is a parameter because this package
// cannot import the installer; callers pass installer.InstalledPluginAgents,
// which counts a plugin recorded in the CLI install state with or without a
// known version. No telemetry is logged: `databricks version` never resolves
// auth, so the event would not be uploaded (see logHintShown). It prints
// nothing when DATABRICKS_AGENT_FEATURES_DISABLED is set.
func MaybeVersionCheckTip(ctx context.Context, installedPlugins func(context.Context) []string) {
	if skills.AgentFeaturesDisabled(ctx) {
		return
	}
	if isHintedAgent(useragent.AgentProvider()) {
		return
	}
	if dbr.HasDetection(ctx) && dbr.RunsOnRuntime(ctx) {
		return
	}
	// Any recorded plugin returns first, so the install state is read once.
	if len(installedPlugins(ctx)) > 0 || HasDatabricksToolingInstalled(ctx, noInstalledPlugins) {
		return
	}
	// Recorded plugins are checked above, so DatabricksTooling only needs the
	// agent's own manifest, its skills dir and the project dirs up to the
	// repository root.
	for _, a := range Registry {
		if a.DatabricksTooling(ctx, noRecordedPlugin) != ToolingNone {
			return
		}
	}
	cmdio.LogString(ctx, skillsTipText)
}

// noInstalledPlugins is the installedPlugins argument to
// HasDatabricksToolingInstalled for a CLI install state that records no plugin.
func noInstalledPlugins(context.Context) []string { return nil }

// RecommendSkillsInstall checks if coding agents are detected without Databricks tooling (skills
// or the databricks plugin) installed. Callers pass installer.InstalledPluginAgents as
// installedPlugins (see HasDatabricksToolingInstalled).
// In interactive mode, prompts the user to install now using installFn; a "no" also turns
// auto-install off, and a "yes" stores the consent to keep the tools up to date unless the
// user answered before. In non-interactive mode, prints a hint. It does nothing at all when
// DATABRICKS_AGENT_FEATURES_DISABLED is set: no hint, no prompt, no install and no consent.
func RecommendSkillsInstall(ctx context.Context, installedPlugins func(context.Context) []string, installFn func(context.Context) error) error {
	if skills.AgentFeaturesDisabled(ctx) {
		return nil
	}
	if HasDatabricksToolingInstalled(ctx, installedPlugins) {
		return nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		PrintSkillsTip(ctx)
		return nil
	}

	logHintShown(ctx, protos.AitoolsHintKindAppsInitPrompt)
	yes, err := cmdio.AskYesOrNo(ctx, "Coding agents detected without Databricks skills. Install skills now?")
	if err != nil {
		return err
	}
	if !yes {
		// The automatic install after this command (and after later upgrades)
		// must not install what the user just turned down.
		if _, err := autoinstall.Decline(ctx); err != nil {
			log.Warnf(ctx, "Could not turn off the automatic installation of Databricks AI tools: %v", err)
			return nil
		}
		// The prompt asked about now; say that the answer also holds later.
		cmdio.LogString(ctx, "Databricks AI tools won't be installed automatically either. To turn that back on, run: databricks aitools auto-install enable")
		return nil
	}

	// The user asked for the tools, so the automatic install may keep them up to
	// date, also when this install fails. An earlier opt-out stays.
	if _, err := autoinstall.GiveConsentIfUnknown(ctx); err != nil {
		log.Debugf(ctx, "Could not save the consent to install Databricks AI tools automatically: %v", err)
	}

	if err := installFn(ctx); err != nil {
		log.Warnf(ctx, "Skills installation failed: %v", err)
	}

	return nil
}
