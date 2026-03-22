package installer

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/fatih/color"
	"golang.org/x/mod/semver"
)

const (
	skillsRepoOwner      = "databricks"
	skillsRepoName       = "databricks-agent-skills"
	skillsRepoPath       = "skills"
	defaultSkillsRepoRef = "v0.1.3"
)

// fetchFileFn is the function used to download individual skill files.
// It is a package-level var so tests can replace it with a mock.
var fetchFileFn = fetchSkillFile

func getSkillsRef(ctx context.Context) string {
	if ref := env.Get(ctx, "DATABRICKS_SKILLS_REF"); ref != "" {
		return ref
	}
	return defaultSkillsRepoRef
}

// Manifest describes the skills manifest fetched from the skills repo.
type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`
}

// SkillMeta describes a single skill entry in the manifest.
type SkillMeta struct {
	Version      string   `json:"version"`
	UpdatedAt    string   `json:"updated_at"`
	Files        []string `json:"files"`
	Experimental bool     `json:"experimental,omitempty"`
	Description  string   `json:"description,omitempty"`
	MinCLIVer    string   `json:"min_cli_version,omitempty"`
}

// InstallOptions controls the behavior of InstallSkillsForAgents.
type InstallOptions struct {
	IncludeExperimental bool
	SpecificSkills      []string // empty = all skills
}

// FetchManifest fetches the skills manifest from the skills repo.
// This is a convenience wrapper that uses the default GitHubManifestSource.
func FetchManifest(ctx context.Context) (*Manifest, error) {
	src := &GitHubManifestSource{}
	ref := getSkillsRef(ctx)
	return src.FetchManifest(ctx, ref)
}

func fetchSkillFile(ctx context.Context, ref, skillName, filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s/%s/%s",
		skillsRepoOwner, skillsRepoName, ref, skillsRepoPath, skillName, filePath)

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

// InstallSkillsForAgents fetches the manifest and installs skills for the given agents.
// This is the core installation function. Callers are responsible for agent detection,
// prompting, and printing the "Installing..." header.
func InstallSkillsForAgents(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent, opts InstallOptions) error {
	latestTag, _, err := src.FetchLatestRelease(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	manifest, err := src.FetchManifest(ctx, latestTag)
	if err != nil {
		return err
	}

	globalDir, err := GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}

	// Load existing state for idempotency checks.
	state, err := LoadState(globalDir)
	if err != nil {
		return fmt.Errorf("failed to load install state: %w", err)
	}

	// Detect legacy installs (skills on disk but no state file).
	if state == nil {
		checkLegacyInstall(ctx, globalDir)
	}

	// Filter skills based on options, experimental flag, and CLI version.
	targetSkills, err := resolveSkills(ctx, manifest.Skills, opts)
	if err != nil {
		return err
	}

	// Install each skill.
	for name, meta := range targetSkills {
		// Idempotency: skip if same version is already installed and on disk.
		if state != nil && state.Skills[name] == meta.Version {
			skillDir := filepath.Join(globalDir, name)
			if _, statErr := os.Stat(skillDir); statErr == nil {
				log.Debugf(ctx, "%s v%s already installed, skipping", name, meta.Version)
				continue
			}
		}

		if err := installSkillForAgents(ctx, latestTag, name, meta.Files, targetAgents, globalDir); err != nil {
			return err
		}
	}

	// Save state. Merge into existing state so skills from previous installs
	// (e.g., experimental skills from a prior run) are preserved.
	existingState, _ := LoadState(globalDir)
	var newState *InstallState
	if existingState != nil {
		newState = existingState
		newState.Release = latestTag
		newState.LastUpdated = time.Now()
		newState.IncludeExperimental = opts.IncludeExperimental
		for name, meta := range targetSkills {
			newState.Skills[name] = meta.Version
		}
	} else {
		newState = &InstallState{
			SchemaVersion:       1,
			IncludeExperimental: opts.IncludeExperimental,
			Release:             latestTag,
			LastUpdated:         time.Now(),
			Skills:              make(map[string]string, len(targetSkills)),
		}
		for name, meta := range targetSkills {
			newState.Skills[name] = meta.Version
		}
	}
	if err := SaveState(globalDir, newState); err != nil {
		return err
	}

	tag := strings.TrimPrefix(latestTag, "v")
	noun := "skills"
	if len(targetSkills) == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Installed %d %s (v%s).", len(targetSkills), noun, tag))
	return nil
}

// resolveSkills filters the manifest skills based on the install options,
// experimental flag, and CLI version constraints.
func resolveSkills(ctx context.Context, skills map[string]SkillMeta, opts InstallOptions) (map[string]SkillMeta, error) {
	isSpecific := len(opts.SpecificSkills) > 0
	cliVersion := build.GetInfo().Version
	isDev := strings.HasPrefix(cliVersion, build.DefaultSemver)

	// Start with all skills or only the requested ones.
	var candidates map[string]SkillMeta
	if isSpecific {
		candidates = make(map[string]SkillMeta, len(opts.SpecificSkills))
		for _, name := range opts.SpecificSkills {
			meta, ok := skills[name]
			if !ok {
				return nil, fmt.Errorf("skill %q not found", name)
			}
			candidates[name] = meta
		}
	} else {
		candidates = skills
	}

	result := make(map[string]SkillMeta, len(candidates))
	for name, meta := range candidates {
		if meta.Experimental && !opts.IncludeExperimental {
			if isSpecific {
				return nil, fmt.Errorf("skill %q is experimental; use --include-experimental to install", name)
			}
			log.Debugf(ctx, "Skipping experimental skill %s", name)
			continue
		}

		if meta.MinCLIVer != "" && !isDev && semver.Compare("v"+cliVersion, "v"+meta.MinCLIVer) < 0 {
			if isSpecific {
				return nil, fmt.Errorf("skill %q requires CLI version %s (running %s)", name, meta.MinCLIVer, cliVersion)
			}
			log.Warnf(ctx, "Skipping %s: requires CLI version %s (running %s)", name, meta.MinCLIVer, cliVersion)
			continue
		}

		result[name] = meta
	}
	return result, nil
}

// InstallAllSkills fetches the manifest and installs all skills for detected agents.
// The signature is func(context.Context) error to satisfy the callback in cmd/apps/init.go.
func InstallAllSkills(ctx context.Context) error {
	installed := agents.DetectInstalled(ctx)
	if len(installed) == 0 {
		printNoAgentsDetected(ctx)
		return nil
	}

	PrintInstallingFor(ctx, installed)
	src := &GitHubManifestSource{}
	return InstallSkillsForAgents(ctx, src, installed, InstallOptions{})
}

// InstallSkill installs a single skill by name for all detected agents.
func InstallSkill(ctx context.Context, skillName string) error {
	installed := agents.DetectInstalled(ctx)
	if len(installed) == 0 {
		printNoAgentsDetected(ctx)
		return nil
	}

	PrintInstallingFor(ctx, installed)
	src := &GitHubManifestSource{}
	return InstallSkillsForAgents(ctx, src, installed, InstallOptions{SpecificSkills: []string{skillName}})
}

// PrintInstallingFor prints the "Installing..." header with agent names.
func PrintInstallingFor(ctx context.Context, targetAgents []*agents.Agent) {
	names := make([]string, len(targetAgents))
	for i, a := range targetAgents {
		names[i] = a.DisplayName
	}
	cmdio.LogString(ctx, fmt.Sprintf("Installing Databricks AI skills for %s...", strings.Join(names, ", ")))
}

func printNoAgentsDetected(ctx context.Context) {
	cmdio.LogString(ctx, color.YellowString("No supported coding agents detected."))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity")
	cmdio.LogString(ctx, "Please install at least one coding agent first.")
}

// checkLegacyInstall prints a message if skills exist on disk but no state file was found.
func checkLegacyInstall(ctx context.Context, globalDir string) {
	if hasSkillsOnDisk(globalDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks experimental aitools install' to refresh.")
		return
	}
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return
	}
	legacyDir := filepath.Join(homeDir, ".databricks", "agent-skills")
	if hasSkillsOnDisk(legacyDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks experimental aitools install' to refresh.")
	}
}

// hasSkillsOnDisk checks if a directory contains subdirectories starting with "databricks".
func hasSkillsOnDisk(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "databricks") {
			return true
		}
	}
	return false
}

func installSkillForAgents(ctx context.Context, ref, skillName string, files []string, detectedAgents []*agents.Agent, globalDir string) error {
	canonicalDir := filepath.Join(globalDir, skillName)
	if err := installSkillToDir(ctx, ref, skillName, canonicalDir, files); err != nil {
		return err
	}

	useSymlinks := len(detectedAgents) > 1

	for _, agent := range detectedAgents {
		agentSkillDir, err := agent.SkillsDir(ctx)
		if err != nil {
			log.Warnf(ctx, "Skipped %s: %v", agent.DisplayName, err)
			continue
		}

		destDir := filepath.Join(agentSkillDir, skillName)

		if err := backupThirdPartySkill(ctx, destDir, canonicalDir, skillName, agent.DisplayName); err != nil {
			log.Warnf(ctx, "Failed to back up existing skill for %s: %v", agent.DisplayName, err)
			continue
		}

		if useSymlinks {
			if err := createSymlink(canonicalDir, destDir); err != nil {
				log.Debugf(ctx, "Symlink failed for %s, copying instead: %v", agent.DisplayName, err)
				if err := installSkillToDir(ctx, ref, skillName, destDir, files); err != nil {
					log.Warnf(ctx, "Failed to install for %s: %v", agent.DisplayName, err)
					continue
				}
			}
			log.Debugf(ctx, "Installed %q for %s (symlinked)", skillName, agent.DisplayName)
		} else {
			if err := installSkillToDir(ctx, ref, skillName, destDir, files); err != nil {
				log.Warnf(ctx, "Failed to install for %s: %v", agent.DisplayName, err)
				continue
			}
			log.Debugf(ctx, "Installed %q for %s", skillName, agent.DisplayName)
		}
	}

	return nil
}

// backupThirdPartySkill moves destDir to a temp directory if it exists and is not
// a symlink pointing to canonicalDir. This preserves skills installed by other tools.
func backupThirdPartySkill(ctx context.Context, destDir, canonicalDir, skillName, agentName string) error {
	fi, err := os.Lstat(destDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// If it's a symlink to our canonical dir, no backup needed.
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(destDir)
		if err == nil && target == canonicalDir {
			return nil
		}
	}

	backupDir, err := os.MkdirTemp("", fmt.Sprintf("databricks-skill-backup-%s-*", skillName))
	if err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupDest := filepath.Join(backupDir, skillName)
	if err := os.Rename(destDir, backupDest); err != nil {
		return fmt.Errorf("failed to move existing skill: %w", err)
	}

	log.Debugf(ctx, "Existing %q for %s moved to %s", skillName, agentName, backupDest)
	return nil
}

func installSkillToDir(ctx context.Context, ref, skillName, destDir string, files []string) error {
	// remove existing skill directory for clean install
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove existing skill: %w", err)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	for _, file := range files {
		content, err := fetchFileFn(ctx, ref, skillName, file)
		if err != nil {
			return err
		}

		destPath := filepath.Join(destDir, file)

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		log.Debugf(ctx, "Downloading %s/%s", skillName, file)
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
