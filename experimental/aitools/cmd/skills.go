package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
}

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
		Use:   "install [skill-name]",
		Short: "Install Databricks skills for Claude Code",
		Long:  `Install Databricks skills to ~/.claude/skills/ for use with Claude Code. If no skill name is provided, installs all available skills.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return installSkill(cmd.Context(), args[0])
			}
			return installAllSkills(cmd.Context())
		},
	}
}

func fetchManifest(ctx context.Context) (*Manifest, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/manifest.json",
		skillsRepoOwner, skillsRepoName, getSkillsBranch(), skillsRepoPath)
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

	// get list of files in skill
	files, err := fetchSkillFileList(ctx, skillName)
	if err != nil {
		return fmt.Errorf("failed to list skill files: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	destDir := filepath.Join(homeDir, ".claude", "skills", skillName)

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

	cmdio.LogString(ctx, color.GreenString("✓ Installed %q to %s", skillName, destDir))
	return nil
}
