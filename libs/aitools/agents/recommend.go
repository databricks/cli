package agents

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// PrintSkillsTip prints the non-interactive `apps init` tip to install skills
// and records that it was shown. Callers decide whether the tooling is missing
// (see HasDatabricksToolingInstalled).
func PrintSkillsTip(ctx context.Context) {
	cmdio.LogString(ctx, "Tip: coding agents detected without Databricks skills. Run 'databricks aitools install' to install them.")
	logHintShown(ctx, protos.AitoolsHintKindAppsInitTip)
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
