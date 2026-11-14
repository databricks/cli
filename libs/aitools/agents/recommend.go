package agents

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// skillsTipText is the tip printed by PrintSkillsTip and MaybeVersionCheckTip.
const skillsTipText = "Tip: coding agents detected without Databricks skills. Run 'databricks aitools install' to install them."

// PrintSkillsTip prints the non-interactive `apps init` tip to install skills
// and records that it was shown. Callers decide whether the tooling is missing
// (see HasDatabricksToolingInstalled).
func PrintSkillsTip(ctx context.Context) {
	cmdio.LogString(ctx, skillsTipText)
	logHintShown(ctx, protos.AitoolsHintKindAppsInitTip)
}

// MaybeVersionCheckTip prints skillsTipText after `databricks version --check`
// when a coding agent is detected without Databricks tooling. Claude Code
// callers are skipped because MaybeHint owns their hints, and it recommends a
// skill only on the commands that skill helps with. installedPlugins is a parameter because this package
// cannot import the installer; callers pass installer.InstalledPluginAgents,
// which counts a plugin recorded in the CLI install state with or without a
// known version. No telemetry is logged: `databricks version` never resolves
// auth, so the event would not be uploaded (see logHintShown).
func MaybeVersionCheckTip(ctx context.Context, installedPlugins func(context.Context) []string) {
	if useragent.AgentProvider() == claudeAgentProvider {
		return
	}
	if dbr.HasDetection(ctx) && dbr.RunsOnRuntime(ctx) {
		return
	}
	if HasDatabricksToolingInstalled(ctx) || len(installedPlugins(ctx)) > 0 {
		return
	}
	// Recorded plugins are checked above, so DatabricksTooling only needs the
	// agent's own manifest, its skills dir and the project dirs up to the
	// repository root.
	noRecordedPlugin := func() bool { return false }
	for _, a := range Registry {
		if a.DatabricksTooling(ctx, noRecordedPlugin) != ToolingNone {
			return
		}
	}
	cmdio.LogString(ctx, skillsTipText)
}

// RecommendSkillsInstall checks if coding agents are detected without Databricks tooling (skills
// or the databricks plugin) installed.
// In interactive mode, prompts the user to install now using installFn. In non-interactive mode, prints a hint.
func RecommendSkillsInstall(ctx context.Context, installFn func(context.Context) error) error {
	if HasDatabricksToolingInstalled(ctx) {
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
		return nil
	}

	if err := installFn(ctx); err != nil {
		log.Warnf(ctx, "Skills installation failed: %v", err)
	}

	return nil
}
