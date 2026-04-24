package installer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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
	defaultSkillsRepoRef = "v0.1.5"
)

// fetchFileFn is the function used to download individual skill files.
// It is a package-level var so tests can replace it with a mock.
var fetchFileFn = fetchSkillFile

// GetSkillsRef returns the skills repo ref to use. If DATABRICKS_SKILLS_REF
// is set, it returns that value; otherwise it returns the default ref.
func GetSkillsRef(ctx context.Context) string {
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
	Scope               string   // ScopeGlobal or ScopeProject (default: global)
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

// InstallSkillsForAgents fetches the manifest and installs skills for the given agents.
// This is the core installation function. Callers are responsible for agent detection,
// prompting, and printing the "Installing..." header.
func InstallSkillsForAgents(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent, opts InstallOptions) error {
	ref := GetSkillsRef(ctx)
	manifest, err := src.FetchManifest(ctx, ref)
	if err != nil {
		return err
	}

	scope := opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}

	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return err
	}

	// For project scope, filter to agents that support it and warn about the rest.
	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to determine working directory: %w", err)
		}
		incompatible := incompatibleAgentNames(targetAgents)
		targetAgents = filterProjectAgents(ctx, targetAgents)
		if len(targetAgents) == 0 {
			return fmt.Errorf("no agents support project-scoped skills. The following detected agents are global-only: %s", strings.Join(incompatible, ", "))
		}
	}

	// Load existing state for idempotency checks.
	state, err := LoadState(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load install state: %w", err)
	}

	// Detect legacy installs (skills on disk but no state file). Global only.
	// Block targeted installs on legacy setups to avoid writing incomplete state
	// that would hide the legacy warning on future runs.
	if state == nil && scope == ScopeGlobal {
		isLegacy := checkLegacyInstall(ctx, baseDir)
		if isLegacy && len(opts.SpecificSkills) > 0 {
			return errors.New("legacy install detected without state tracking; run 'databricks experimental aitools install' (without a skill name) first to rebuild state")
		}
	}

	// Filter skills based on options, experimental flag, and CLI version.
	targetSkills, err := resolveSkills(ctx, manifest.Skills, opts)
	if err != nil {
		return err
	}

	params := installParams{
		baseDir: baseDir,
		scope:   scope,
		cwd:     cwd,
		ref:     ref,
	}

	// Install each skill in sorted order for determinism.
	skillNames := slices.Sorted(maps.Keys(targetSkills))

	for _, name := range skillNames {
		meta := targetSkills[name]
		// Idempotency: skip if same version is already installed, the canonical
		// dir exists, AND every requested agent already has the skill on disk.
		if state != nil && state.Skills[name] == meta.Version {
			skillDir := filepath.Join(baseDir, name)
			if _, statErr := os.Stat(skillDir); statErr == nil && allAgentsHaveSkill(ctx, name, targetAgents, scope, cwd) {
				log.Debugf(ctx, "%s v%s already installed for all agents, skipping", name, meta.Version)
				continue
			}
		}

		if err := installSkillForAgents(ctx, name, meta.Files, targetAgents, params); err != nil {
			return err
		}
	}

	// Save state. Merge into existing state (loaded above) so skills from
	// previous installs (e.g., experimental skills from a prior run) are preserved.
	if state == nil {
		state = &InstallState{
			SchemaVersion: 1,
			Skills:        make(map[string]string, len(targetSkills)),
		}
	}
	state.Release = ref
	state.LastUpdated = time.Now()
	// IncludeExperimental reflects the last invocation's flag value. The Skills
	// map may still contain experimental entries from a prior run with the flag
	// enabled; this field does not retroactively remove them.
	state.IncludeExperimental = opts.IncludeExperimental
	state.Scope = scope
	for name, meta := range targetSkills {
		state.Skills[name] = meta.Version
	}
	if err := SaveState(baseDir, state); err != nil {
		return err
	}

	tag := strings.TrimPrefix(ref, "v")
	noun := "skills"
	if len(targetSkills) == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Installed %d %s (v%s).", len(targetSkills), noun, tag))
	return nil
}

// skillsDir returns the base skills directory for the given scope.
func skillsDir(ctx context.Context, scope string) (string, error) {
	if scope == ScopeProject {
		return ProjectSkillsDir(ctx)
	}
	return GlobalSkillsDir(ctx)
}

// filterProjectAgents returns only agents that support project scope and warns about skipped agents.
func filterProjectAgents(ctx context.Context, targetAgents []*agents.Agent) []*agents.Agent {
	var compatible []*agents.Agent
	for _, a := range targetAgents {
		if a.SupportsProjectScope {
			compatible = append(compatible, a)
		} else {
			cmdio.LogString(ctx, "Skipped "+a.DisplayName+": does not support project-scoped skills.")
		}
	}
	return compatible
}

// incompatibleAgentNames returns the display names of agents that do not support project scope.
func incompatibleAgentNames(targetAgents []*agents.Agent) []string {
	var names []string
	for _, a := range targetAgents {
		if !a.SupportsProjectScope {
			names = append(names, a.DisplayName)
		}
	}
	return names
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
				return nil, fmt.Errorf("skill %q is experimental; use --experimental to install", name)
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
// Returns true if a legacy install was detected.
func checkLegacyInstall(ctx context.Context, globalDir string) bool {
	if hasSkillsOnDisk(globalDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks experimental aitools install' to refresh.")
		return true
	}
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	legacyDir := filepath.Join(homeDir, ".databricks", "agent-skills")
	if hasSkillsOnDisk(legacyDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks experimental aitools install' to refresh.")
		return true
	}
	return false
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

// allAgentsHaveSkill returns true if every agent in the list has the named
// skill directory present (either as a real directory or symlink).
func allAgentsHaveSkill(ctx context.Context, skillName string, targetAgents []*agents.Agent, scope, cwd string) bool {
	for _, agent := range targetAgents {
		agentSkillDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			return false
		}
		if _, err := os.Stat(filepath.Join(agentSkillDir, skillName)); err != nil {
			return false
		}
	}
	return true
}

// installParams bundles the parameters for installSkillForAgents to keep the signature manageable.
type installParams struct {
	baseDir string
	scope   string
	cwd     string
	ref     string
}

func installSkillForAgents(ctx context.Context, skillName string, files []string, detectedAgents []*agents.Agent, params installParams) error {
	canonicalDir := filepath.Join(params.baseDir, skillName)
	if err := installSkillToDir(ctx, params.ref, skillName, canonicalDir, files); err != nil {
		return err
	}

	// For project scope, always symlink. For global, symlink when multiple agents.
	useSymlinks := params.scope == ScopeProject || len(detectedAgents) > 1

	for _, agent := range detectedAgents {
		agentSkillDir, err := agentSkillsDirForScope(ctx, agent, params.scope, params.cwd)
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
			symlinkTarget := canonicalDir
			// For project scope, use relative symlinks so they work for teammates.
			if params.scope == ScopeProject {
				rel, relErr := filepath.Rel(filepath.Dir(destDir), canonicalDir)
				if relErr == nil {
					symlinkTarget = rel
				}
			}
			if err := createSymlink(symlinkTarget, destDir); err != nil {
				log.Debugf(ctx, "Symlink failed for %s, copying instead: %v", agent.DisplayName, err)
				if err := copyDir(canonicalDir, destDir); err != nil {
					log.Warnf(ctx, "Failed to install for %s: %v", agent.DisplayName, err)
					continue
				}
			}
			log.Debugf(ctx, "Installed %q for %s (symlinked)", skillName, agent.DisplayName)
		} else {
			// Copy from canonical dir instead of re-downloading.
			if err := copyDir(canonicalDir, destDir); err != nil {
				log.Warnf(ctx, "Failed to install for %s: %v", agent.DisplayName, err)
				continue
			}
			log.Debugf(ctx, "Installed %q for %s", skillName, agent.DisplayName)
		}
	}

	return nil
}

// agentSkillsDirForScope returns the agent's skills directory for the given scope.
func agentSkillsDirForScope(ctx context.Context, agent *agents.Agent, scope, cwd string) (string, error) {
	if scope == ScopeProject {
		return agent.ProjectSkillsDir(cwd), nil
	}
	return agent.SkillsDir(ctx)
}

// backupThirdPartySkill moves destDir to a temp directory if it exists and is not
// a symlink pointing to canonicalDir. This preserves skills installed by other tools.
func backupThirdPartySkill(ctx context.Context, destDir, canonicalDir, skillName, agentName string) error {
	fi, err := os.Lstat(destDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	// If it's a symlink to our canonical dir, no backup needed.
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(destDir)
		if err == nil {
			absTarget := target
			if !filepath.IsAbs(target) {
				absTarget = filepath.Clean(filepath.Join(filepath.Dir(destDir), target))
			}
			if absTarget == canonicalDir {
				return nil
			}
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

// copyDir copies all files from src to dest, recreating the directory structure.
func copyDir(src, dest string) error {
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("failed to remove existing path: %w", err)
	}

	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
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
