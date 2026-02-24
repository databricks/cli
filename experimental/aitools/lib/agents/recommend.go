package agents

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// RecommendSkillsInstall checks if coding agents are detected but have no skills installed.
// In interactive mode, prompts the user to install now using installFn. In non-interactive mode, prints a hint.
func RecommendSkillsInstall(ctx context.Context, installFn func(context.Context) error) error {
	if HasDatabricksSkillsInstalled() {
		return nil
	}

	if !cmdio.IsPromptSupported(ctx) {
		cmdio.LogString(ctx, "Tip: coding agents detected without Databricks skills. Run 'databricks experimental aitools skills install' to install them.")
		return nil
	}

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
