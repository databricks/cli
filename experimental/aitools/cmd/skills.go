package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

const (
	skillsRepoOwner         = "databricks"
	skillsRepoName          = "databricks-agent-skills"
	skillsRepoPath          = "skills"
	defaultSkillsRepoBranch = "main"
	canonicalSkillsDir      = ".databricks/agent-skills" // canonical location for symlink source
)

func getSkillsBranch() string {
	if branch := os.Getenv("DATABRICKS_SKILLS_BRANCH"); branch != "" {
		return branch
	}
	return defaultSkillsRepoBranch
}

type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`
}

type SkillMeta struct {
	Version   string   `json:"version"`
	UpdatedAt string   `json:"updated_at"`
	Files     []string `json:"files"`
}

func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Manage Databricks skills for coding agents",
		Long:  `Manage Databricks skills that extend coding agents with Databricks-specific capabilities.`,
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
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for detected coding agents",
		Long: `Install Databricks skills to all detected coding agents.

Skills are installed globally to each agent's skills directory.
When multiple agents are detected, skills are stored in a canonical location
and symlinked to each agent to avoid duplication.

Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return installSkill(cmd.Context(), args[0])
			}
			return installAllSkills(cmd.Context())
		},
	}
}

func fetchManifest(ctx context.Context) (*Manifest, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/manifest.json",
		skillsRepoOwner, skillsRepoName, getSkillsBranch())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func fetchSkillFile(ctx context.Context, skillName, filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/%s/%s",
		skillsRepoOwner, skillsRepoName, getSkillsBranch(), skillsRepoPath, skillName, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", filePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filePath, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func listSkills(ctx context.Context) error {
	manifest, err := fetchManifest(ctx)
	if err != nil {
		return err
	}

	cmdio.LogString(ctx, "Available skills:")
	cmdio.LogString(ctx, "")

	for name, meta := range manifest.Skills {
		cmdio.LogString(ctx, fmt.Sprintf("  %s (v%s)", name, meta.Version))
	}

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Install all with: databricks experimental aitools skills install")
	cmdio.LogString(ctx, "Install one with: databricks experimental aitools skills install <skill-name>")
	return nil
}

func installAllSkills(ctx context.Context) error {
	manifest, err := fetchManifest(ctx)
	if err != nil {
		return err
	}

	// detect agents once for all skills
	detectedAgents := agents.DetectInstalled()
	if len(detectedAgents) == 0 {
		printNoAgentsDetected(ctx)
		return nil
	}

	printDetectedAgents(ctx, detectedAgents)

	for name, meta := range manifest.Skills {
		if err := installSkillForAgents(ctx, name, meta.Files, detectedAgents); err != nil {
			return err
		}
	}
	return nil
}

func printNoAgentsDetected(ctx context.Context) {
	cmdio.LogString(ctx, color.YellowString("No supported coding agents detected."))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity")
	cmdio.LogString(ctx, "Please install at least one coding agent first.")
}

func printDetectedAgents(ctx context.Context, detectedAgents []*agents.Agent) {
	cmdio.LogString(ctx, "Detected coding agents:")
	for _, agent := range detectedAgents {
		cmdio.LogString(ctx, "  - "+agent.DisplayName)
	}
	cmdio.LogString(ctx, "")
}

func installSkill(ctx context.Context, skillName string) error {
	manifest, err := fetchManifest(ctx)
	if err != nil {
		return err
	}

	if _, ok := manifest.Skills[skillName]; !ok {
		return fmt.Errorf("skill %q not found", skillName)
	}

	detectedAgents := agents.DetectInstalled()
	if len(detectedAgents) == 0 {
		printNoAgentsDetected(ctx)
		return nil
	}

	printDetectedAgents(ctx, detectedAgents)

	return installSkillForAgents(ctx, skillName, manifest.Skills[skillName].Files, detectedAgents)
}

func installSkillForAgents(ctx context.Context, skillName string, files []string, detectedAgents []*agents.Agent) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// determine installation strategy
	useSymlinks := len(detectedAgents) > 1
	var canonicalDir string

	if useSymlinks {
		// install to canonical location and symlink to each agent
		canonicalDir = filepath.Join(homeDir, canonicalSkillsDir, skillName)
		if err := installSkillToDir(ctx, skillName, canonicalDir, files); err != nil {
			return err
		}
	}

	// install/symlink to each agent
	for _, agent := range detectedAgents {
		agentSkillDir, err := agent.SkillsDir()
		if err != nil {
			cmdio.LogString(ctx, color.YellowString("⊘ Skipped %s: %v", agent.DisplayName, err))
			continue
		}

		destDir := filepath.Join(agentSkillDir, skillName)

		if useSymlinks {
			if err := createSymlink(canonicalDir, destDir); err != nil {
				// fallback to copy on symlink failure (e.g., Windows without admin)
				cmdio.LogString(ctx, color.YellowString("  Symlink failed for %s, copying instead...", agent.DisplayName))
				if err := installSkillToDir(ctx, skillName, destDir, files); err != nil {
					cmdio.LogString(ctx, color.YellowString("⊘ Failed to install for %s: %v", agent.DisplayName, err))
					continue
				}
			}
			cmdio.LogString(ctx, color.GreenString("✓ Installed %q for %s (symlinked)", skillName, agent.DisplayName))
		} else {
			// single agent - install directly
			if err := installSkillToDir(ctx, skillName, destDir, files); err != nil {
				cmdio.LogString(ctx, color.YellowString("⊘ Failed to install for %s: %v", agent.DisplayName, err))
				continue
			}
			cmdio.LogString(ctx, color.GreenString("✓ Installed %q for %s", skillName, agent.DisplayName))
		}
	}

	return nil
}

func installSkillToDir(ctx context.Context, skillName, destDir string, files []string) error {
	// remove existing skill directory for clean install
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove existing skill: %w", err)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// download all files
	for _, file := range files {
		content, err := fetchSkillFile(ctx, skillName, file)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, file)

		// create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := os.WriteFile(destPath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	return nil
}

func createSymlink(source, dest string) error {
	// ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}

	// remove existing symlink or directory
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to remove existing path: %w", err)
	}

	// create symlink
	if err := os.Symlink(source, dest); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}
