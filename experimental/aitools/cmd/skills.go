package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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

// getGitHubToken returns GitHub token from environment or gh CLI.
// TODO: once databricks-agent-skills repo is public, replace GitHub API calls
// with raw.githubusercontent.com URLs and remove authentication logic.
func getGitHubToken() string {
	// check environment variables first
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	if token := os.Getenv("GH_TOKEN"); token != "" {
		return token
	}
	// try gh CLI
	out, err := exec.Command("gh", "auth", "token").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}

// addGitHubAuth adds authentication header if token is available.
func addGitHubAuth(req *http.Request) {
	if token := getGitHubToken(); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}

// AgentConfig defines how to detect and install skills for an agent.
type AgentConfig struct {
	Name           string
	DisplayName    string
	GlobalSkillDir func() (string, error) // returns global skills directory path
	Detect         func() bool            // returns true if agent is installed
}

// getHomeDir returns home directory, handling Windows USERPROFILE.
func getHomeDir() (string, error) {
	if runtime.GOOS == "windows" {
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			return userProfile, nil
		}
	}
	return os.UserHomeDir()
}

// supportedAgents defines all agents we can install skills to.
var supportedAgents = []AgentConfig{
	{
		Name:        "claude-code",
		DisplayName: "Claude Code",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".claude", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".claude"))
			return err == nil
		},
	},
	{
		Name:        "cursor",
		DisplayName: "Cursor",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".cursor", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".cursor"))
			return err == nil
		},
	},
	{
		Name:        "windsurf",
		DisplayName: "Windsurf",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".codeium", "windsurf", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".codeium", "windsurf"))
			return err == nil
		},
	},
	{
		Name:        "cline",
		DisplayName: "Cline",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".cline", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".cline"))
			return err == nil
		},
	},
	{
		Name:        "roo-code",
		DisplayName: "Roo Code",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".roo-code", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".roo-code"))
			return err == nil
		},
	},
	{
		Name:        "codex",
		DisplayName: "Codex CLI",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".codex", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".codex"))
			return err == nil
		},
	},
	{
		Name:        "amp",
		DisplayName: "Amp",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".amp", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".amp"))
			return err == nil
		},
	},
	{
		Name:        "opencode",
		DisplayName: "OpenCode",
		GlobalSkillDir: func() (string, error) {
			home, err := getHomeDir()
			if err != nil {
				return "", err
			}
			return filepath.Join(home, ".config", "opencode", "skills"), nil
		},
		Detect: func() bool {
			home, err := getHomeDir()
			if err != nil {
				return false
			}
			_, err = os.Stat(filepath.Join(home, ".config", "opencode"))
			return err == nil
		},
	},
}

// detectInstalledAgents returns list of agents that are installed on the system.
func detectInstalledAgents() []AgentConfig {
	var installed []AgentConfig
	for _, agent := range supportedAgents {
		if agent.Detect() {
			installed = append(installed, agent)
		}
	}
	return installed
}

type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`
}

type SkillMeta struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
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

Supported agents: Claude Code, Cursor, Windsurf, Cline, Roo Code, Codex CLI, Amp, OpenCode`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return installSkill(cmd.Context(), args[0])
			}
			return installAllSkills(cmd.Context())
		},
	}
}

func fetchManifest(ctx context.Context) (*Manifest, error) {
	// use GitHub API for private repo support
	// manifest.json is at repo root, skills are in skillsRepoPath subdirectory
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/manifest.json?ref=%s",
		skillsRepoOwner, skillsRepoName, getSkillsBranch())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	addGitHubAuth(req)

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
	// use GitHub API for private repo support
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s/%s/%s?ref=%s",
		skillsRepoOwner, skillsRepoName, skillsRepoPath, skillName, filePath, getSkillsBranch())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github.raw+json")
	addGitHubAuth(req)

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

func fetchSkillFileList(ctx context.Context, skillName string) ([]string, error) {
	// use GitHub API to list files in skill directory
	skillPath := skillsRepoPath + "/" + skillName
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		skillsRepoOwner, skillsRepoName, skillPath, getSkillsBranch())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	addGitHubAuth(req)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list skill files: HTTP %d", resp.StatusCode)
	}

	var items []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	var files []string
	for _, item := range items {
		switch item.Type {
		case "file":
			// strip skills/skill-name prefix from path
			relPath := strings.TrimPrefix(item.Path, skillPath+"/")
			files = append(files, relPath)
		case "dir":
			// recursively list subdirectory
			subFiles, err := fetchSubdirFiles(ctx, item.Path)
			if err != nil {
				return nil, err
			}
			for _, sf := range subFiles {
				relPath := strings.TrimPrefix(sf, skillPath+"/")
				files = append(files, relPath)
			}
		}
	}

	return files, nil
}

func fetchSubdirFiles(ctx context.Context, dirPath string) ([]string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s?ref=%s",
		skillsRepoOwner, skillsRepoName, dirPath, getSkillsBranch())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	addGitHubAuth(req)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to list directory %s: HTTP %d", dirPath, resp.StatusCode)
	}

	var items []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, err
	}

	var files []string
	for _, item := range items {
		switch item.Type {
		case "file":
			files = append(files, item.Path)
		case "dir":
			subFiles, err := fetchSubdirFiles(ctx, item.Path)
			if err != nil {
				return nil, err
			}
			files = append(files, subFiles...)
		}
	}

	return files, nil
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

	for name := range manifest.Skills {
		if err := installSkill(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

func installSkill(ctx context.Context, skillName string) error {
	manifest, err := fetchManifest(ctx)
	if err != nil {
		return err
	}

	if _, ok := manifest.Skills[skillName]; !ok {
		return fmt.Errorf("skill %q not found", skillName)
	}

	// detect installed agents
	agents := detectInstalledAgents()
	if len(agents) == 0 {
		cmdio.LogString(ctx, color.YellowString("No supported coding agents detected."))
		cmdio.LogString(ctx, "")
		cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Windsurf, Cline, Roo Code, Codex CLI, Amp, OpenCode")
		cmdio.LogString(ctx, "Please install at least one coding agent first.")
		return nil
	}

	// print detected agents
	cmdio.LogString(ctx, "Detected coding agents:")
	for _, agent := range agents {
		cmdio.LogString(ctx, "  - "+agent.DisplayName)
	}
	cmdio.LogString(ctx, "")

	// get list of files in skill
	files, err := fetchSkillFileList(ctx, skillName)
	if err != nil {
		return fmt.Errorf("failed to list skill files: %w", err)
	}

	homeDir, err := getHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// determine installation strategy
	useSymlinks := len(agents) > 1
	var canonicalDir string

	if useSymlinks {
		// install to canonical location and symlink to each agent
		canonicalDir = filepath.Join(homeDir, canonicalSkillsDir, skillName)
		if err := installSkillToDir(ctx, skillName, canonicalDir, files); err != nil {
			return err
		}
	}

	// install/symlink to each agent
	for _, agent := range agents {
		agentSkillDir, err := agent.GlobalSkillDir()
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
