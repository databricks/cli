package installer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/skills"
	"golang.org/x/mod/semver"
	"golang.org/x/sync/errgroup"
)

const (
	skillsRepoOwner      = skills.RepoOwner
	skillsRepoName       = skills.RepoName
	stableSkillsRepoPath = skills.StableRepoDir
	experimentalRepoPath = skills.ExperimentalRepoDir

	// fetchConcurrency caps the number of in-flight skill file fetches. The
	// value lives in libs/skills, next to the HTTP client sized for it.
	fetchConcurrency = skills.FetchConcurrency
)

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

// renamePathFn is the function used to move paths into place. Tests replace it
// to exercise cross-device fallback paths without depending on host mounts.
var renamePathFn = os.Rename

// DisplaySkillsVersion renders a skills ref for humans: a pinned tag drops its
// leading "v".
func DisplaySkillsVersion(ref string) string {
	return strings.TrimPrefix(ref, "v")
}

// GetSkillsRef returns the skills ref to install from and whether it was
// explicitly pinned. See skills.GetSkillsRef.
func GetSkillsRef(ctx context.Context) (ref string, explicit bool, err error) {
	return skills.GetSkillsRef(ctx)
}

// GetSkillsBaseURL returns the manifest and skill fetch base URL.
func GetSkillsBaseURL(ctx context.Context) string {
	return skills.GetSkillsBaseURL(ctx)
}

// Manifest and SkillMeta are defined in libs/skills, which the skill sources share.
type (
	Manifest  = skills.Manifest
	SkillMeta = skills.SkillMeta
)

// InstallOptions controls the behavior of InstallSkillsForAgents.
type InstallOptions struct {
	IncludeExperimental bool
	SpecificSkills      []string // empty = all skills
	Scope               string   // ScopeGlobal or ScopeProject (default: global)
	// SkillsOnly records the target agents that could have had the plugin in
	// InstallState.SkillsOnlyAgents: the user chose raw skills for them.
	SkillsOnly bool
	// KeepHandPlacedSkills refuses the install with a *RefusedError, before
	// it writes anything, when a target agent's skills directory has an entry
	// named after a skill to install that is neither a link to that skill in
	// the base directory nor a copy of a subset of its files. The install
	// would move such an entry aside: the user or another tool put it there.
	// One such entry refuses the whole install, for every target agent and
	// skill: the install state records the skills for all of them at once,
	// so installing the rest would record skills that an agent lacks, and
	// a later update, which reinstalls only skills whose version changed,
	// would never give them to it.
	KeepHandPlacedSkills bool
}

func fetchSkillFile(ctx context.Context, ref, repoDir, skillName, filePath string) ([]byte, error) {
	if repoDir == "" {
		repoDir = stableSkillsRepoPath
	}
	url := fmt.Sprintf("%s/%s/%s/%s/%s",
		GetSkillsBaseURL(ctx), ref, repoDir, skillName, escapeFilePath(filePath))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := skills.HTTPClient().Do(req)
	if err != nil {
		return nil, &FetchError{Stage: FetchStageFile, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, skills.NewHTTPFetchError(FetchStageFile, resp, fmt.Errorf("HTTP %d", resp.StatusCode))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &FetchError{Stage: FetchStageFile, StatusCode: resp.StatusCode, Err: fmt.Errorf("failed to read response: %w", err)}
	}
	return data, nil
}

// escapeFilePath escapes each slash-separated element of a skill file path, so
// a '#' or '?' in a name stays part of the path instead of starting a fragment
// or query. Ordinary names are unchanged.
func escapeFilePath(filePath string) string {
	elems := strings.Split(filePath, "/")
	for i, elem := range elems {
		elems[i] = url.PathEscape(elem)
	}
	return strings.Join(elems, "/")
}

// FetchSkillsManifestWithFallback fetches the skills manifest at the given ref.
// If the ref points to a non-existent tag (not-found error), it falls back to
// the embedded manifest's skills version. A manifest that names a skill or file
// outside its directory fails with ErrUnsafeManifestPath, whatever the source.
// Returns the manifest, the (possibly updated) ref, and any error.
func FetchSkillsManifestWithFallback(ctx context.Context, src ManifestSource, ref string, allowFallback bool) (*Manifest, string, error) {
	tag := strings.TrimPrefix(ref, "v")
	manifest, err := src.FetchManifest(ctx, ref)
	if err != nil && allowFallback && clicompat.IsNotFoundError(err) {
		fallbackVersion, fbErr := clicompat.ResolveEmbeddedAgentSkillsVersion()
		if fbErr == nil && fallbackVersion != "" && fallbackVersion != clicompat.AgentSkillsLatest && fallbackVersion != tag {
			log.Warnf(ctx, "Skills version %s not found, falling back to embedded version %s", tag, fallbackVersion)
			ref = "v" + fallbackVersion
			manifest, err = src.FetchManifest(ctx, ref)
		} else if fbErr != nil {
			log.Warnf(ctx, "Could not resolve embedded skills version: %v", fbErr)
		}
	}
	if err != nil {
		return nil, ref, err
	}
	if err := skills.CheckManifestPaths(manifest); err != nil {
		return nil, ref, err
	}
	return manifest, ref, nil
}

// installedSkillsRelease returns the newest recorded raw-skills release and
// whether fallback is safe to consider. Branch refs cannot be ordered, so they
// require a successful latest-release fetch instead of an embedded fallback.
func installedSkillsRelease(state *InstallState) (ref string, allowFallback bool) {
	if state == nil || len(state.Skills) == 0 {
		return "", true
	}
	ref = semver.Canonical("v" + DisplaySkillsVersion(state.Release))
	allowFallback = ref != ""
	// Plugin installs can overwrite state.Release, and selective skill installs
	// can leave files from a newer release. Keep their provenance in the check.
	knownSkills := make(map[string]bool, len(state.Skills))
	for path, record := range state.Files {
		if record.Origin == "" {
			continue
		}
		origin := semver.Canonical("v" + DisplaySkillsVersion(record.Origin))
		if origin == "" {
			allowFallback = false
		} else {
			name, _, _ := strings.Cut(path, "/")
			knownSkills[name] = true
		}
		if semver.Compare(origin, ref) > 0 {
			ref = origin
		}
	}
	// Without raw-file provenance, a mixed legacy/plugin install has no
	// reliable raw-skills release to compare with the fallback.
	if len(state.Plugins) > 0 {
		for name := range state.Skills {
			if !knownSkills[name] {
				allowFallback = false
			}
		}
	}
	return ref, allowFallback
}

func checkSkillsDowngrade(installedRef, ref string, explicit bool) error {
	if !explicit && semver.Compare(ref, installedRef) < 0 {
		return &RefusedError{Err: fmt.Errorf("refusing to downgrade skills from %s to %s; retry when the latest release is available or set DATABRICKS_SKILLS_REF=%s to select this version explicitly", DisplaySkillsVersion(installedRef), DisplaySkillsVersion(ref), ref)}
	}
	return nil
}

// InstallSkillsForAgents fetches the manifest and installs skills for the given agents.
// This is the core installation function. Callers are responsible for agent detection,
// prompting, and printing the "Installing..." header. Agents that did not get
// every skill are reported in an *AgentSkillsError, after the install state is
// saved.
func InstallSkillsForAgents(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent, opts InstallOptions) error {
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

	// Load existing state for idempotency and downgrade checks.
	state, err := LoadState(baseDir)
	if err != nil {
		return fmt.Errorf("failed to load install state: %w", err)
	}

	// A "legacy install" is one from before the CLI tracked install state in a
	// JSON file: skills exist on disk but there's no state file (state == nil).
	// A targeted install (--skills foo) here would write a state file listing only
	// foo, leaving the other on-disk skills untracked. Block it so the user runs a
	// full install first, which rebuilds complete state. Global scope only, since
	// legacy installs only ever wrote to the global dir.
	// hasLegacyInstall lives in update.go (shared with the update/uninstall paths).
	legacyInstall := state == nil && scope == ScopeGlobal && hasLegacyInstall(ctx, baseDir)
	if legacyInstall && len(opts.SpecificSkills) > 0 {
		return &RefusedError{Err: errors.New("legacy install detected without state tracking; run 'databricks aitools install --scope=global --skills-only' (without --skills) first to rebuild state")}
	}

	installedRef, allowFallback := installedSkillsRelease(state)
	allowFallback = allowFallback && !legacyInstall
	ref, explicit, err := skills.ResolveSkillsRef(ctx, allowFallback)
	if err != nil {
		return err
	}
	cmdio.LogProgress(ctx, "Using skills version "+DisplaySkillsVersion(ref))
	cmdio.LogProgress(ctx, "Fetching skills manifest...")
	manifest, ref, err := FetchSkillsManifestWithFallback(ctx, src, ref, allowFallback && !explicit)
	if err != nil {
		return err
	}
	if err := checkSkillsDowngrade(installedRef, ref, explicit); err != nil {
		return err
	}

	if opts.IncludeExperimental && !manifestHasExperimental(manifest) {
		log.Warnf(ctx, "--experimental was set but the manifest at %s exposes no experimental skills. Set DATABRICKS_SKILLS_REF to a release or ref that includes them.", ref)
	}

	// Filter skills based on options, experimental flag, and CLI version.
	targetSkills, err := resolveSkills(ctx, manifest.Skills, opts)
	if err != nil {
		return err
	}
	if len(targetSkills) == 0 {
		return ErrNoSkillsResolved
	}

	params := installParams{
		baseDir: baseDir,
		scope:   scope,
		cwd:     cwd,
		ref:     ref,
	}

	// Install each skill in sorted order for determinism. Idempotency: skip a
	// skill if the same version is already installed, the canonical dir
	// exists, AND every requested agent already has the skill on disk.
	skillNames := slices.DeleteFunc(slices.Sorted(maps.Keys(targetSkills)), func(name string) bool {
		meta := targetSkills[name]
		if state == nil || state.Skills[name] != meta.Version || stateRepoDir(state, name) != meta.RepoDir {
			return false
		}
		if _, statErr := os.Stat(filepath.Join(baseDir, name)); statErr != nil || !allAgentsHaveSkill(ctx, name, targetAgents, scope, cwd) {
			return false
		}
		log.Debugf(ctx, "%s v%s already installed for all agents, skipping", name, meta.Version)
		return true
	})

	if opts.KeepHandPlacedSkills {
		if agent, name, found := handPlacedSkill(ctx, targetAgents, skillNames, params); found {
			return &RefusedError{Err: fmt.Errorf("kept the %s skill in the skills directory of %s, which an install would move aside; run 'databricks aitools install' to replace it", name, agent.DisplayName)}
		}
	}

	// Scoped to the loop, not the manifest fetch above, so earlier warnings print
	// as plain lines instead of racing the live spinner's redraw.
	sp := cmdio.NewSpinner(ctx)
	defer sp.Close()

	// Accumulate file provenance for skills we (re)fetch this run. Skipped
	// (already-installed) skills keep their existing records via the merge below.
	fileRecords := map[string]FileRecord{}
	var refetched []string
	failures := map[*agents.Agent]*AgentSkillsFailure{}

	for _, name := range skillNames {
		meta := targetSkills[name]

		sp.Update("Installing " + name + "...")
		records, failed, err := installSkillForAgents(ctx, name, meta, targetAgents, params)
		if err != nil {
			return err
		}
		for a, agentErr := range failed {
			f := failures[a]
			if f == nil {
				f = &AgentSkillsFailure{Agent: a, Total: len(targetSkills), Err: agentErr}
				failures[a] = f
			}
			f.Skills++
		}
		maps.Copy(fileRecords, records)
		refetched = append(refetched, name)
	}

	// Save state. Merge into existing state (loaded above) so skills from
	// previous installs (e.g., experimental skills from a prior run) are preserved.
	if state == nil {
		state = &InstallState{
			SchemaVersion: CurrentSchemaVersion,
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
	if opts.SkillsOnly {
		for _, a := range targetAgents {
			// Without its CLI on PATH the agent could not have had the plugin:
			// install suggests --skills-only for that case too, and update
			// should migrate the agent once the CLI is there.
			if a.Plugin != nil && a.HasBinary(ctx) && !slices.Contains(state.SkillsOnlyAgents, a.Name) {
				state.SkillsOnlyAgents = append(state.SkillsOnlyAgents, a.Name)
			}
		}
		slices.Sort(state.SkillsOnlyAgents)
	}
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

	// Stop the spinner before the summary so its transient line doesn't linger
	// above the persistent "Installed N skills." output.
	sp.Close()

	// When no agent got any skill, the summary would read as a success. An
	// agent that got some of them counts: its own line names what is missing.
	gotAny := slices.ContainsFunc(targetAgents, func(a *agents.Agent) bool {
		f := failures[a]
		return f == nil || f.Partial()
	})
	if gotAny {
		noun := "skills"
		if len(targetSkills) == 1 {
			noun = "skill"
		}
		// Name the directory so an agent can open the installed SKILL.md files
		// instead of searching the filesystem for them.
		cmdio.LogProgress(ctx, fmt.Sprintf("Installed %d %s in %s.", len(targetSkills), noun, filepath.ToSlash(baseDir)))
	}
	if len(failures) == 0 {
		return nil
	}
	agentsErr := &AgentSkillsError{}
	for _, a := range targetAgents {
		if f := failures[a]; f != nil {
			agentsErr.Failures = append(agentsErr.Failures, *f)
		}
	}
	return agentsErr
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
		if a.SupportsProjectSkills {
			compatible = append(compatible, a)
		} else {
			cmdio.LogProgress(ctx, "Skipped "+a.DisplayName+": does not support project-scoped skills.")
		}
	}
	return compatible
}

// incompatibleAgentNames returns the display names of agents that do not support project scope.
func incompatibleAgentNames(targetAgents []*agents.Agent) []string {
	var names []string
	for _, a := range targetAgents {
		if !a.SupportsProjectSkills {
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
	isDev := build.GetInfo().IsDevelopment()

	// Start with all skills or only the requested ones.
	var candidates map[string]SkillMeta
	if isSpecific {
		candidates = make(map[string]SkillMeta, len(opts.SpecificSkills))
		for _, name := range opts.SpecificSkills {
			meta, ok := skills[name]
			if !ok {
				return nil, &SkillError{Skill: name, Reason: ReasonSkillNotFound, Detail: "not found"}
			}
			candidates[name] = meta
		}
	} else {
		candidates = skills
	}

	result := make(map[string]SkillMeta, len(candidates))
	for name, meta := range candidates {
		if reason := unportableSkill(name, meta.Files); reason != "" {
			if isSpecific {
				return nil, &SkillError{Skill: name, Reason: ReasonUnportableSkill, Detail: "cannot be installed: " + reason}
			}
			log.Warnf(ctx, "Skipping skill %q: %s", name, reason)
			continue
		}

		if meta.IsExperimental() && !opts.IncludeExperimental {
			if isSpecific {
				return nil, &SkillError{Skill: name, Reason: ReasonExperimentalSkill, Detail: "is experimental; use --experimental to install"}
			}
			log.Debugf(ctx, "Skipping experimental skill %s", name)
			continue
		}

		if meta.MinCLIVer != "" && !isDev && semver.Compare("v"+cliVersion, "v"+meta.MinCLIVer) < 0 {
			if isSpecific {
				return nil, &SkillError{Skill: name, Reason: ReasonVersionIncompatible, Detail: fmt.Sprintf("requires CLI version %s (running %s)", meta.MinCLIVer, cliVersion)}
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
// Agents that did not get the skills are printed with a step to fix them,
// not returned.
func InstallAllSkills(ctx context.Context) error {
	installed := agents.DetectInstalled(ctx)
	if len(installed) == 0 {
		printNoAgentsDetected(ctx)
		return nil
	}

	PrintInstallingFor(ctx, installed)
	src := &GitHubManifestSource{}
	err := InstallSkillsForAgents(ctx, src, installed, InstallOptions{})
	// An optional setup step (apps init) has nothing to install; that is no
	// failure for it.
	if errors.Is(err, ErrNoSkillsResolved) {
		cmdio.LogString(ctx, NoSkillsResolvedMessage)
		return nil
	}
	// The caller's warning would name no agent and no step, so each agent
	// that did not get the skills is reported here with its own step.
	if agentsErr, ok := errors.AsType[*AgentSkillsError](err); ok {
		for _, f := range agentsErr.Failures {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, f.Notice()))
			cmdio.LogString(ctx, fmt.Sprintf("  Fix the error above, then run 'databricks aitools install --scope=global --agents=%s --skills-only'.", f.Agent.Name))
		}
		return nil
	}
	return err
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
	cmdio.LogString(ctx, "Supported agents: "+strings.Join(agents.SupportedNames(), ", "))
	cmdio.LogString(ctx, "Please install at least one coding agent first.")
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

// installSkillForAgents installs one skill in the canonical directory and
// exposes it to each agent. An agent that cannot get it is logged and returned
// in failed with its error; the other agents still get the skill.
func installSkillForAgents(ctx context.Context, skillName string, meta SkillMeta, detectedAgents []*agents.Agent, params installParams) (records map[string]FileRecord, failed map[*agents.Agent]error, err error) {
	canonicalDir := filepath.Join(params.baseDir, skillName)
	records, err = installSkillToDir(ctx, params.ref, meta.RepoDir, meta.SourceName, canonicalDir, meta.Files)
	if err != nil {
		return nil, nil, err
	}
	failed = map[*agents.Agent]error{}

	// For project scope, always symlink. For global, symlink when multiple agents.
	useSymlinks := params.scope == ScopeProject || len(detectedAgents) > 1

	for _, agent := range detectedAgents {
		agentSkillDir, err := agentSkillsDirForScope(ctx, agent, params.scope, params.cwd)
		if err != nil {
			log.Warnf(ctx, "Skipped %s: %v", agent.DisplayName, err)
			failed[agent] = err
			continue
		}

		destDir := filepath.Join(agentSkillDir, skillName)

		if err := backupThirdPartySkill(ctx, destDir, canonicalDir, skillName, agent.DisplayName); err != nil {
			log.Warnf(ctx, "Failed to back up existing skill for %s: %v", agent.DisplayName, err)
			failed[agent] = err
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
					failed[agent] = err
					continue
				}
			}
			log.Debugf(ctx, "Installed %q for %s (symlinked)", skillName, agent.DisplayName)
		} else {
			// Copy from canonical dir instead of re-downloading.
			if err := copyDir(canonicalDir, destDir); err != nil {
				log.Warnf(ctx, "Failed to install for %s: %v", agent.DisplayName, err)
				failed[agent] = err
				continue
			}
			log.Debugf(ctx, "Installed %q for %s", skillName, agent.DisplayName)
		}
	}

	return records, failed, nil
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
	_, err := os.Lstat(destDir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	// If it's a symlink to our canonical dir, no backup needed.
	if linksTo(destDir, canonicalDir) {
		return nil
	}

	backupDir, err := os.MkdirTemp("", fmt.Sprintf("databricks-skill-backup-%s-*", skillName))
	if err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupDest := filepath.Join(backupDir, skillName)
	if err := movePathWithCrossDeviceFallback(destDir, backupDest); err != nil {
		return fmt.Errorf("failed to move existing skill: %w", err)
	}

	log.Debugf(ctx, "Existing %q for %s moved to %s", skillName, agentName, backupDest)
	return nil
}

// handPlacedSkill returns the first target agent and skill whose entry in the
// agent's skills directory is neither a link to the canonical skill nor a copy
// of a subset of its files. installSkillForAgents would move such an entry
// aside; a link or a copy is what an earlier install that did not finish left.
func handPlacedSkill(ctx context.Context, targetAgents []*agents.Agent, skillNames []string, params installParams) (*agents.Agent, string, bool) {
	for _, agent := range targetAgents {
		agentSkillDir, err := agentSkillsDirForScope(ctx, agent, params.scope, params.cwd)
		if err != nil {
			continue
		}
		for _, name := range skillNames {
			path := filepath.Join(agentSkillDir, name)
			if _, err := os.Lstat(path); err != nil {
				continue
			}
			canonicalDir := filepath.Join(params.baseDir, name)
			if !linksTo(path, canonicalDir) && !isCopyOf(path, canonicalDir) {
				return agent, name, true
			}
		}
	}
	return nil, "", false
}

// linksTo reports whether path is a symlink whose target is canonicalDir.
func linksTo(path, canonicalDir string) bool {
	fi, err := os.Lstat(path)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(path)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Clean(filepath.Join(filepath.Dir(path), target))
	}
	return target == canonicalDir
}

// isCopyOf reports whether dir is a directory, not a link, whose every file is
// a regular file with the same contents as the file at the same path in
// canonicalDir.
func isCopyOf(dir, canonicalDir string) bool {
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return false
	}
	if fi, err := os.Stat(canonicalDir); err != nil || !fi.IsDir() {
		return false
	}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return errNotCopy
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		ours, err := os.ReadFile(filepath.Join(canonicalDir, rel))
		if err != nil {
			return err
		}
		theirs, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(ours, theirs) {
			return errNotCopy
		}
		return nil
	})
	return err == nil
}

var errNotCopy = errors.New("not a copy")

func movePathWithCrossDeviceFallback(src, dest string) error {
	if err := renamePathFn(src, dest); err != nil {
		if !isCrossDeviceRenameError(err) {
			return err
		}
		if err := copyPath(src, dest); err != nil {
			return err
		}
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("failed to remove original path after copy: %w", err)
		}
	}
	return nil
}

func isCrossDeviceRenameError(err error) bool {
	if errors.Is(err, syscall.EXDEV) {
		return true
	}
	if runtime.GOOS != "windows" {
		return false
	}

	errno, ok := errors.AsType[syscall.Errno](err)
	return ok && errno == syscall.Errno(17) // ERROR_NOT_SAME_DEVICE
}

// installSkillToDir downloads a skill's files into destDir and returns a
// FileRecord per file (keyed by "<skillName>/<file>", forward slashes) capturing
// the sha256 of the bytes written, so a later update can prune only the files we
// wrote that the user hasn't modified.
func installSkillToDir(ctx context.Context, ref, repoDir, skillName, destDir string, files []string) (map[string]FileRecord, error) {
	parentDir := filepath.Dir(destDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create skills directory: %w", err)
	}
	tmpDir, err := os.MkdirTemp(parentDir, "."+filepath.Base(destDir)+"-*.tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary skill directory: %w", err)
	}
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.RemoveAll(tmpDir)
		}
	}()

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
				return fmt.Errorf("failed to fetch %s/%s/%s at %s: %w", repoDir, skillName, filepath.ToSlash(file), ref, err)
			}
			destPath := filepath.Join(tmpDir, file)
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

	if err := replaceDir(destDir, tmpDir); err != nil {
		return nil, err
	}
	cleanupTmp = false
	return records, nil
}

func replaceDir(destDir, tmpDir string) error {
	parentDir := filepath.Dir(destDir)
	base := filepath.Base(destDir)

	var backupParent string
	var backupDest string
	if _, err := os.Lstat(destDir); err == nil {
		var mkErr error
		backupParent, mkErr = os.MkdirTemp(parentDir, "."+base+"-backup-*.tmp")
		if mkErr != nil {
			return fmt.Errorf("failed to create backup directory: %w", mkErr)
		}
		backupDest = filepath.Join(backupParent, base)
		if err := renamePathFn(destDir, backupDest); err != nil {
			_ = os.RemoveAll(backupParent)
			return fmt.Errorf("failed to move existing skill aside: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to inspect existing skill: %w", err)
	}

	if err := renamePathFn(tmpDir, destDir); err != nil {
		if backupDest != "" {
			if restoreErr := renamePathFn(backupDest, destDir); restoreErr != nil {
				return fmt.Errorf("failed to move skill into place: %w (also failed to restore existing skill: %v)", err, restoreErr) //nolint:errorlint // only the primary error belongs in the chain
			}
		}
		return fmt.Errorf("failed to move skill into place: %w", err)
	}

	if backupParent != "" {
		_ = os.RemoveAll(backupParent)
	}
	return nil
}

func copyPath(src, dest string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return os.Symlink(target, dest)
	}
	if info.IsDir() {
		return copyDir(src, dest)
	}
	return copyFile(src, dest, info.Mode())
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

func copyFile(src, dest string, mode fs.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, data, mode.Perm())
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
