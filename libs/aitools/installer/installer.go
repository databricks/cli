package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"sync"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

const (
	skillsRepoOwner      = "databricks"
	skillsRepoName       = "databricks-agent-skills"
	stableSkillsRepoPath = "skills"
	experimentalRepoPath = "experimental"

	// fetchConcurrency caps the number of in-flight skill file fetches.
	// Each file is one HTTPS GET to raw.githubusercontent.com; sequential
	// fetches were latency-bound on TLS handshakes. 8 is enough to amortise
	// the round-trip across a typical skill's files without overwhelming the
	// upstream CDN.
	fetchConcurrency = 8
)

// httpClient is shared across all skill file fetches so the underlying
// transport reuses TCP+TLS connections. The default MaxIdleConnsPerHost
// (2) is bumped to leave headroom above fetchConcurrency so a brief overlap
// between a closing and a new connection doesn't force a fresh handshake.
var httpClient = sync.OnceValue(func() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = fetchConcurrency * 2
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: t,
	}
})

func manifestHasExperimental(m *Manifest) bool {
	for _, meta := range m.Skills {
		if meta.IsExperimental() {
			return true
		}
	}
	return false
}

func stateRepoDir(state *InstallState, name string) string {
	if state != nil && state.RepoDirs != nil {
		if repoDir := state.RepoDirs[name]; repoDir != "" {
			return repoDir
		}
	}
	return stableSkillsRepoPath
}

// fetchFileFn is the function used to download individual skill files.
// It is a package-level var so tests can replace it with a mock.
var fetchFileFn func(ctx context.Context, ref, repoDir, skillName, filePath string) ([]byte, error) = fetchSkillFile

// latestSkillsRef is the ref used when nothing pins a version. It tracks the
// skills repo's default branch, the same content plugin agents receive (their
// marketplace clones the default branch), so skills and plugin agents stay in
// sync by default.
const latestSkillsRef = "main"

// skillsLatestSentinel is the value cli-compat.json uses for the skills version
// to mean "track latest" rather than pinning a specific release.
const skillsLatestSentinel = clicompat.AgentSkillsLatest

// GetSkillsRef returns the skills repo ref to use and whether it was explicitly
// pinned (as opposed to tracking latest).
//
// By default we track the latest skills. cli-compat.json is the remote control
// for this: it normally reports "latest" (so skills match what plugin agents
// install), but if a future change makes a skills release incompatible with
// older CLIs, it can be edited remotely to pin those CLIs to the last compatible
// version with no CLI release. DATABRICKS_SKILLS_REF overrides everything for
// eval/reproducibility. (AppKit version resolution is unaffected; cli-compat
// still manages it via its own field.)
func GetSkillsRef(ctx context.Context) (ref string, explicit bool, err error) {
	if ref := env.Get(ctx, "DATABRICKS_SKILLS_REF"); ref != "" {
		return ref, true, nil
	}
	v, err := clicompat.ResolveAgentSkillsVersion(ctx)
	if err != nil {
		// If compatibility can't be resolved (offline, parse error), track latest
		// rather than failing the install.
		log.Debugf(ctx, "could not resolve skills compatibility, tracking latest: %v", err)
		return latestSkillsRef, false, nil
	}
	if v == "" || v == skillsLatestSentinel {
		return latestSkillsRef, false, nil
	}
	return "v" + v, true, nil
}

// DisplaySkillsVersion renders a skills ref for humans: the latest sentinel
// becomes "latest"; a pinned tag drops its leading "v".
func DisplaySkillsVersion(ref string) string {
	if ref == latestSkillsRef {
		return "latest"
	}
	return strings.TrimPrefix(ref, "v")
}

// GetSkillsBaseURL returns the manifest and skill fetch base URL.
// DATABRICKS_SKILLS_BASE_URL overrides GitHub raw URLs for acceptance tests.
func GetSkillsBaseURL(ctx context.Context) string {
	if base := env.Get(ctx, "DATABRICKS_SKILLS_BASE_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "https://raw.githubusercontent.com/" + skillsRepoOwner + "/" + skillsRepoName
}

// Manifest describes the skills manifest fetched from the skills repo.
type Manifest struct {
	Version   string               `json:"version"`
	UpdatedAt string               `json:"updated_at"`
	Skills    map[string]SkillMeta `json:"skills"`
}

// SkillMeta describes a single skill entry in the manifest.
type SkillMeta struct {
	Version     string   `json:"version"`
	UpdatedAt   string   `json:"updated_at"`
	Files       []string `json:"files"`
	Description string   `json:"description,omitempty"`
	MinCLIVer   string   `json:"min_cli_version,omitempty"`

	// RepoDir is "skills" or "experimental" (manifest field repo_dir).
	RepoDir string `json:"repo_dir,omitempty"`

	// SourceName is the upstream skill directory name within RepoDir.
	// Set during normalization, not from JSON.
	SourceName string `json:"-"`
}

func (s SkillMeta) IsExperimental() bool {
	return s.RepoDir == experimentalRepoPath
}

// InstallOptions controls the behavior of InstallSkillsForAgents.
type InstallOptions struct {
	IncludeExperimental bool
	SpecificSkills      []string // empty = all skills
	Scope               string   // ScopeGlobal or ScopeProject (default: global)
}

func fetchSkillFile(ctx context.Context, ref, repoDir, skillName, filePath string) ([]byte, error) {
	if repoDir == "" {
		repoDir = stableSkillsRepoPath
	}
	url := fmt.Sprintf("%s/%s/%s/%s/%s",
		GetSkillsBaseURL(ctx), ref, repoDir, skillName, filePath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", filePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: HTTP %d", filePath, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// FetchSkillsManifestWithFallback fetches the skills manifest at the given ref.
// If the ref points to a non-existent tag (not-found error), it falls back to
// the embedded manifest's skills version. Returns the manifest, the (possibly
// updated) ref, and any error.
func FetchSkillsManifestWithFallback(ctx context.Context, src ManifestSource, ref string, allowFallback bool) (*Manifest, string, error) {
	tag := strings.TrimPrefix(ref, "v")
	manifest, err := src.FetchManifest(ctx, ref)
	if err != nil && allowFallback && clicompat.IsNotFoundError(err) {
		fallbackVersion, fbErr := clicompat.ResolveEmbeddedAgentSkillsVersion()
		if fbErr == nil && fallbackVersion != "" && fallbackVersion != skillsLatestSentinel && fallbackVersion != tag {
			log.Warnf(ctx, "Skills version %s not found, falling back to embedded version %s", tag, fallbackVersion)
			ref = "v" + fallbackVersion
			manifest, err = src.FetchManifest(ctx, ref)
		} else if fbErr != nil {
			log.Warnf(ctx, "Could not resolve embedded skills version: %v", fbErr)
		}
	}
	return manifest, ref, err
}

// InstallSkillsForAgents fetches the manifest and installs skills for the given agents.
// This is the core installation function. Callers are responsible for agent detection,
// prompting, and printing the "Installing..." header.
func InstallSkillsForAgents(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent, opts InstallOptions) error {
	ref, explicit, err := GetSkillsRef(ctx)
	if err != nil {
		return err
	}
	cmdio.LogString(ctx, "Using skills version "+DisplaySkillsVersion(ref))
	manifest, ref, err := FetchSkillsManifestWithFallback(ctx, src, ref, !explicit)
	if err != nil {
		return err
	}

	if opts.IncludeExperimental && !manifestHasExperimental(manifest) {
		log.Warnf(ctx, "--experimental was set but the manifest at %s exposes no experimental skills. Set DATABRICKS_SKILLS_REF to a release that includes them (or =main for the latest).", ref)
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
			return errors.New("legacy install detected without state tracking; run 'databricks aitools install' (without a skill name) first to rebuild state")
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

	// Accumulate file provenance for skills we (re)fetch this run. Skipped
	// (already-installed) skills keep their existing records via the merge below.
	fileRecords := map[string]FileRecord{}
	var refetched []string

	for _, name := range skillNames {
		meta := targetSkills[name]

		// Idempotency: skip if same version is already installed, the canonical
		// dir exists, AND every requested agent already has the skill on disk.
		if state != nil && state.Skills[name] == meta.Version && stateRepoDir(state, name) == meta.RepoDir {
			skillDir := filepath.Join(baseDir, name)
			if _, statErr := os.Stat(skillDir); statErr == nil && allAgentsHaveSkill(ctx, name, targetAgents, scope, cwd) {
				log.Debugf(ctx, "%s v%s already installed for all agents, skipping", name, meta.Version)
				continue
			}
		}

		records, err := installSkillForAgents(ctx, name, meta, targetAgents, params)
		if err != nil {
			return err
		}
		maps.Copy(fileRecords, records)
		refetched = append(refetched, name)
	}

	// Save state. Merge into existing state (loaded above) so skills from
	// previous installs (e.g., experimental skills from a prior run) are preserved.
	if state == nil {
		state = &InstallState{
			SchemaVersion: schemaVersionV2,
			Skills:        make(map[string]string, len(targetSkills)),
			RepoDirs:      make(map[string]string, len(targetSkills)),
		}
	}
	if state.Skills == nil {
		state.Skills = make(map[string]string, len(targetSkills))
	}
	if state.RepoDirs == nil {
		state.RepoDirs = make(map[string]string, len(state.Skills)+len(targetSkills))
	}
	if state.Files == nil {
		state.Files = make(map[string]FileRecord, len(fileRecords))
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
		state.RepoDirs[name] = meta.RepoDir
	}
	// Drop stale provenance for refetched skills before recording the fresh set,
	// so files removed/renamed in a new version don't leave orphaned records.
	for _, name := range refetched {
		clearFileRecords(state.Files, name)
	}
	maps.Copy(state.Files, fileRecords)
	if err := SaveState(baseDir, state); err != nil {
		return err
	}

	noun := "skills"
	if len(targetSkills) == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Installed %d %s.", len(targetSkills), noun))
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
		if meta.IsExperimental() && !opts.IncludeExperimental {
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
	cmdio.LogString(ctx, fmt.Sprintf("Installing Databricks skills for %s...", strings.Join(names, ", ")))
}

func printNoAgentsDetected(ctx context.Context) {
	cmdio.LogString(ctx, cmdio.Yellow(ctx, "No supported coding agents detected."))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported agents: Claude Code, Cursor, Codex CLI, OpenCode, GitHub Copilot, Antigravity")
	cmdio.LogString(ctx, "Please install at least one coding agent first.")
}

// checkLegacyInstall prints a message if skills exist on disk but no state file was found.
// Returns true if a legacy install was detected.
func checkLegacyInstall(ctx context.Context, globalDir string) bool {
	if hasSkillsOnDisk(globalDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks aitools install' to refresh.")
		return true
	}
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	legacyDir := filepath.Join(homeDir, ".databricks", "agent-skills")
	if hasSkillsOnDisk(legacyDir) {
		cmdio.LogString(ctx, "Found skills installed before state tracking was added. Run 'databricks aitools install' to refresh.")
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

func installSkillForAgents(ctx context.Context, skillName string, meta SkillMeta, detectedAgents []*agents.Agent, params installParams) (map[string]FileRecord, error) {
	canonicalDir := filepath.Join(params.baseDir, skillName)
	records, err := installSkillToDir(ctx, params.ref, meta.RepoDir, meta.SourceName, canonicalDir, meta.Files)
	if err != nil {
		return nil, err
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

	return records, nil
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

// installSkillToDir downloads a skill's files into destDir and returns a
// FileRecord per file (keyed by "<skillName>/<file>", forward slashes) capturing
// the sha256 of the bytes written, so a later update can prune only the files we
// wrote that the user hasn't modified.
func installSkillToDir(ctx context.Context, ref, repoDir, skillName, destDir string, files []string) (map[string]FileRecord, error) {
	// remove existing skill directory for clean install
	if err := os.RemoveAll(destDir); err != nil {
		return nil, fmt.Errorf("failed to remove existing skill: %w", err)
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	var mu sync.Mutex
	records := make(map[string]FileRecord, len(files))

	// Fetch files concurrently. Each file is a separate HTTPS GET, so
	// wall-clock time is dominated by per-request TLS handshakes rather
	// than payload size.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(fetchConcurrency)
	for _, file := range files {
		g.Go(func() error {
			content, err := fetchFileFn(gctx, ref, repoDir, skillName, file)
			if err != nil {
				return err
			}
			destPath := filepath.Join(destDir, file)
			if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
			log.Debugf(gctx, "Downloading %s/%s", skillName, file)
			if err := os.WriteFile(destPath, content, 0o644); err != nil {
				return fmt.Errorf("failed to write %s: %w", file, err)
			}
			sum := sha256.Sum256(content)
			mu.Lock()
			records[skillName+"/"+filepath.ToSlash(file)] = FileRecord{
				SHA256: hex.EncodeToString(sum[:]),
				Origin: ref,
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return records, nil
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
