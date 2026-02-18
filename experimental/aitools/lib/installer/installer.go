package installer

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
)

const (
	skillsRepoOwner         = "databricks"
	skillsRepoName          = "databricks-agent-skills"
	skillsRepoPath          = "skills"
	defaultSkillsRepoBranch = "main"
)

func getSkillsBranch() string {
	if branch := os.Getenv("DATABRICKS_SKILLS_BRANCH"); branch != "" {
		return branch
	}
	return defaultSkillsRepoBranch
}

// Manifest describes the skills manifest fetched from the skills repo.
type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`
}

// SkillMeta describes a single skill entry in the manifest.
type SkillMeta struct {
	Version   string   `json:"version"`
	UpdatedAt string   `json:"updated_at"`
	Files     []string `json:"files"`
}

// FetchManifest fetches the skills manifest from the skills repo.
func FetchManifest(ctx context.Context) (*Manifest, error) {
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

// ListSkills fetches and prints available skills.
func ListSkills(ctx context.Context) error {
	manifest, err := FetchManifest(ctx)
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

// InstallAllSkills fetches the manifest and installs all skills for detected agents.
func InstallAllSkills(ctx context.Context) error {
	manifest, err := FetchManifest(ctx)
	if err != nil {
		return err
	}

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

// InstallSkill fetches the manifest and installs a single skill by name.
func InstallSkill(ctx context.Context, skillName string) error {
	manifest, err := FetchManifest(ctx)
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
		canonicalDir = filepath.Join(homeDir, agents.CanonicalSkillsDir, skillName)
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
