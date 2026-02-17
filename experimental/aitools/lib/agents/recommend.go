package agents

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
)

// RecommendSkillsInstall checks if coding agents are detected but have no skills installed.
// In interactive mode, prompts the user to install now. In non-interactive mode, prints a hint.
func RecommendSkillsInstall(ctx context.Context) error {
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

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.CommandContext(ctx, executable, "experimental", "aitools", "skills", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		log.Warnf(ctx, "Skills installation failed: %v", err)
	}

	return nil
}
