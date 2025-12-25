package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/experimental/apps-mcp/lib/skill"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const cliPathPlaceholder = "__DATABRICKS_CLI_PATH__"

func newInstallSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-skill",
		Short: "Install the databricks-apps skill for Claude Code",
		Long:  `Install the databricks-apps skill to ~/.claude/skills/ for use with Claude Code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstallSkill(cmd.Context())
		},
	}
	return cmd
}

func runInstallSkill(ctx context.Context) error {
	cliPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get CLI path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	destDir := filepath.Join(homeDir, ".claude", "skills", "databricks-apps")

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	skillFS, err := fs.Sub(skill.SkillFS, "databricks-apps")
	if err != nil {
		return fmt.Errorf("failed to access embedded skill: %w", err)
	}

	err = fs.WalkDir(skillFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, path)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		content, err := fs.ReadFile(skillFS, path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		perm := os.FileMode(0o644)
		if path == "scripts/db" {
			content = []byte(strings.ReplaceAll(string(content), cliPathPlaceholder, cliPath))
			perm = 0o755
		}

		return os.WriteFile(destPath, content, perm)
	})
	if err != nil {
		return fmt.Errorf("failed to copy skill files: %w", err)
	}

	cmdio.LogString(ctx, color.GreenString("✓ Installed databricks-apps skill to ")+destDir)
	return nil
}
