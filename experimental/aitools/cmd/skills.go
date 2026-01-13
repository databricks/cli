package mcp

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/experimental/aitools/lib/agent_skills"
	appkitdocs "github.com/databricks/cli/experimental/aitools/templates/appkit"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage Databricks skills for Claude Code",
		Long:  `Manage Databricks skills that can be installed to ~/.claude/skills/ for use with Claude Code.`,
	}

	cmd.AddCommand(newSkillsListCmd())
	cmd.AddCommand(newSkillsInstallCmd())

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available skills",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listSkills(cmd.Context())
		},
	}
}

func newSkillsInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install all Databricks skills for Claude Code",
		Long:  `Install all Databricks skills to ~/.claude/skills/ for use with Claude Code.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return installAllSkills(cmd.Context())
		},
	}
}

func getSkillNames() ([]string, error) {
	entries, err := fs.ReadDir(agent_skills.SkillsFS, ".")
	if err != nil {
		return nil, fmt.Errorf("failed to read skills: %w", err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

func listSkills(ctx context.Context) error {
	names, err := getSkillNames()
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Available skills:")
	cmdio.LogString(ctx, "")
	for _, name := range names {
		cmdio.LogString(ctx, "  "+name)
	}
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Install all with: databricks experimental aitools skills install")
	return nil
}

func installAllSkills(ctx context.Context) error {
	names, err := getSkillNames()
	if err != nil {
		return err
	}

	for _, name := range names {
		if err := installSkill(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func installSkill(ctx context.Context, skillName string) error {
	skillFS, err := fs.Sub(agent_skills.SkillsFS, skillName)
	if err != nil {
		return fmt.Errorf("skill %q not found", skillName)
	}

	if _, err := fs.Stat(skillFS, "SKILL.md"); err != nil {
		return fmt.Errorf("skill %q not found", skillName)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	destDir := filepath.Join(homeDir, ".claude", "skills", skillName)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// copy skill-specific files (SKILL.md, authentication.md, etc.)
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

		return os.WriteFile(destPath, content, 0o644)
	})
	if err != nil {
		return fmt.Errorf("failed to copy skill files: %w", err)
	}

	// copy shared docs from appkit template
	if err := copySharedDocs(destDir); err != nil {
		return fmt.Errorf("failed to copy shared docs: %w", err)
	}

	cmdio.LogString(ctx, color.GreenString("✓ Installed %q to %s", skillName, destDir))
	return nil
}

func copySharedDocs(destDir string) error {
	refsDir := filepath.Join(destDir, "references")
	if err := os.MkdirAll(refsDir, 0o755); err != nil {
		return err
	}

	// docs from appkit template to copy as skill references
	sharedDocs := []string{
		"appkit-sdk.md",
		"frontend.md",
		"sql-queries.md",
		"testing.md",
		"trpc.md",
	}

	for _, doc := range sharedDocs {
		content, err := appkitdocs.DocsFS.ReadFile("template/{{.project_name}}/docs/" + doc)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", doc, err)
		}
		if err := os.WriteFile(filepath.Join(refsDir, doc), content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", doc, err)
		}
	}

	return nil
}
