package installer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

// UpdateOptions controls the behavior of UpdateSkills.
type UpdateOptions struct {
	Force   bool
	NoNew   bool
	NoPrune bool     // keep skills that vanished from the manifest instead of removing them
	Check   bool     // dry run: show what would change without downloading
	Skills  []string // empty = all installed
	Scope   string   // ScopeGlobal or ScopeProject (default: global)
}

// UpdateResult describes what UpdateSkills did (or would do in check mode).
type UpdateResult struct {
	Updated   []SkillUpdate // skills that were updated
	Added     []SkillUpdate // new skills added (when NoNew is false)
	Removed   []SkillUpdate // skills pruned because they vanished from the manifest
	Unchanged []string      // skills at current version (or kept despite vanishing)
	Skipped   []string      // skills skipped (experimental, version constraint)
}

// SkillUpdate describes a single skill version change.
type SkillUpdate struct {
	Name       string
	OldVersion string
	NewVersion string
}

// UpdateSkills updates installed skills to the latest release.
func UpdateSkills(ctx context.Context, src ManifestSource, targetAgents []*agents.Agent, opts UpdateOptions) (*UpdateResult, error) {
	scope := opts.Scope
	if scope == "" {
		scope = ScopeGlobal
	}

	baseDir, err := skillsDir(ctx, scope)
	if err != nil {
		return nil, err
	}

	// For project scope, filter to compatible agents.
	var cwd string
	if scope == ScopeProject {
		cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to determine working directory: %w", err)
		}
		incompatible := incompatibleAgentNames(targetAgents)
		targetAgents = filterProjectAgents(ctx, targetAgents)
		if len(targetAgents) == 0 {
			return nil, fmt.Errorf("no agents support project-scoped skills. The following detected agents are global-only: %s", strings.Join(incompatible, ", "))
		}
	}

	state, err := LoadState(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load install state: %w", err)
	}

	if state == nil {
		if scope == ScopeGlobal && hasLegacyInstall(ctx, baseDir) {
			return nil, errors.New("found skills from a previous install without state tracking; run 'databricks aitools install' to refresh before updating")
		}
		return nil, errors.New("no skills or plugins installed. Run 'databricks aitools install' to install")
	}

	latestTag, explicit, err := GetSkillsRef(ctx)
	if err != nil {
		return nil, err
	}

	// Short-circuit only when pinned to an exact ref. When tracking latest (the
	// default), the ref ("main") never changes, so always reconcile against the
	// freshly fetched manifest instead of falsely reporting "already up to date".
	if explicit && state.Release == latestTag && !opts.Force {
		cmdio.LogString(ctx, "Already up to date.")
		return &UpdateResult{Unchanged: slices.Sorted(maps.Keys(state.Skills))}, nil
	}

	manifest, latestTag, err := FetchSkillsManifestWithFallback(ctx, src, latestTag, !explicit)
	if err != nil {
		if opts.Check {
			log.Warnf(ctx, "Could not fetch manifest: %v", err)
			return &UpdateResult{}, nil
		}
		return nil, err
	}

	// Determine the skill set to consider.
	skillSet := buildUpdateSkillSet(state, manifest, opts)

	result := &UpdateResult{}

	cliVersion := build.GetInfo().Version
	isDev := strings.HasPrefix(cliVersion, build.DefaultSemver)

	// Sort skill names for deterministic output.
	names := slices.Sorted(maps.Keys(skillSet))

	for _, name := range names {
		meta, inManifest := manifest.Skills[name]

		if !inManifest {
			if _, ok := state.Skills[name]; ok {
				// The skill vanished from the manifest. Prune it only when we
				// installed it and the user hasn't modified it since; otherwise
				// keep it with a warning so we never delete user-edited content.
				if !opts.NoPrune && skillPrunable(ctx, baseDir, name, scope, cwd, state) {
					result.Removed = append(result.Removed, SkillUpdate{Name: name, OldVersion: state.Skills[name]})
				} else {
					log.Warnf(ctx, "Warning: %q not found in manifest %s (keeping installed version).", name, latestTag)
					result.Unchanged = append(result.Unchanged, name)
				}
			}
			continue
		}

		// Filter experimental skills unless state opted in.
		if meta.IsExperimental() && !state.IncludeExperimental {
			log.Debugf(ctx, "Skipping experimental skill %s", name)
			result.Skipped = append(result.Skipped, name)
			continue
		}

		// Filter skills requiring a newer CLI version.
		if meta.MinCLIVer != "" && !isDev && semver.Compare("v"+cliVersion, "v"+meta.MinCLIVer) < 0 {
			log.Warnf(ctx, "Skipping %s: requires CLI version %s (running %s)", name, meta.MinCLIVer, cliVersion)
			result.Skipped = append(result.Skipped, name)
			continue
		}

		oldVersion, wasInstalled := state.Skills[name]

		if meta.Version == oldVersion && stateRepoDir(state, name) == meta.RepoDir && !opts.Force {
			result.Unchanged = append(result.Unchanged, name)
			continue
		}

		update := SkillUpdate{
			Name:       name,
			OldVersion: oldVersion,
			NewVersion: meta.Version,
		}

		if wasInstalled {
			result.Updated = append(result.Updated, update)
		} else {
			result.Added = append(result.Added, update)
		}
	}

	if opts.Check {
		return result, nil
	}

	// Download and install updated/added skills.
	allChanges := make([]SkillUpdate, 0, len(result.Updated)+len(result.Added))
	allChanges = append(allChanges, result.Updated...)
	allChanges = append(allChanges, result.Added...)

	params := installParams{
		baseDir: baseDir,
		scope:   scope,
		cwd:     cwd,
		ref:     latestTag,
	}

	fileRecords := map[string]FileRecord{}
	for _, change := range allChanges {
		meta := manifest.Skills[change.Name]
		records, err := installSkillForAgents(ctx, change.Name, meta, targetAgents, params)
		if err != nil {
			return nil, err
		}
		maps.Copy(fileRecords, records)
	}

	// Update state.
	state.Release = latestTag
	state.LastUpdated = time.Now()
	if state.RepoDirs == nil {
		state.RepoDirs = make(map[string]string, len(state.Skills)+len(allChanges))
	}
	if state.Files == nil {
		state.Files = make(map[string]FileRecord, len(fileRecords))
	}
	for _, change := range allChanges {
		meta := manifest.Skills[change.Name]
		state.Skills[change.Name] = change.NewVersion
		state.RepoDirs[change.Name] = meta.RepoDir
		// Drop stale provenance before recording the refetched files, so a file
		// removed/renamed in the new version doesn't leave an orphaned record.
		clearFileRecords(state.Files, change.Name)
	}
	maps.Copy(state.Files, fileRecords)

	// Prune skills that vanished from the manifest (and that the user hasn't
	// modified, as decided above). Removes our symlinks and unmodified copies
	// from every agent plus the canonical copy; user-managed dirs are left alone.
	for _, rem := range result.Removed {
		removeSkillExposures(ctx, baseDir, rem.Name, scope, cwd, state)
		delete(state.Skills, rem.Name)
		delete(state.RepoDirs, rem.Name)
		for path := range state.Files {
			if strings.HasPrefix(path, rem.Name+"/") {
				delete(state.Files, path)
			}
		}
	}

	if err := SaveState(baseDir, state); err != nil {
		return nil, err
	}

	return result, nil
}

// buildUpdateSkillSet determines which skills to consider for update.
func buildUpdateSkillSet(state *InstallState, manifest *Manifest, opts UpdateOptions) map[string]bool {
	skillSet := make(map[string]bool)

	if len(opts.Skills) > 0 {
		for _, name := range opts.Skills {
			skillSet[name] = true
		}
		return skillSet
	}

	// All installed skills.
	for name := range state.Skills {
		skillSet[name] = true
	}

	// Auto-add new skills from manifest (unless --no-new).
	if !opts.NoNew {
		for name := range manifest.Skills {
			skillSet[name] = true
		}
	}

	return skillSet
}

// hasLegacyInstall checks both canonical and legacy dirs for skills on disk without state.
func hasLegacyInstall(ctx context.Context, globalDir string) bool {
	if hasSkillsOnDisk(globalDir) {
		return true
	}
	homeDir, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	return hasSkillsOnDisk(filepath.Join(homeDir, ".databricks", "agent-skills"))
}

// skillFilesUnmodified reports whether every file the CLI recorded for a skill
// still exists in the canonical dir and matches its recorded sha256. Returns
// false when there is no recorded provenance (e.g. a legacy v1 install), so the
// caller keeps such skills rather than risk deleting user content.
// skillPrunable reports whether a vanished skill can be safely removed: the
// canonical copy and every agent exposure (symlink into canonical, or a copied
// dir) must be ours and unmodified. If anything is user-modified, has extra
// files, or has no recorded provenance, the skill is kept (the caller warns).
func skillPrunable(ctx context.Context, baseDir, skillName, scope, cwd string, state *InstallState) bool {
	canonicalDir := filepath.Join(baseDir, skillName)
	if !dirMatchesRecords(canonicalDir, skillName, state) {
		return false
	}
	for _, agent := range agents.Registry {
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}
		if !exposureRemovable(filepath.Join(agentDir, skillName), canonicalDir, skillName, state) {
			return false
		}
	}
	return true
}

// removeSkillExposures removes a skill's exposure from every scoped agent (a
// symlink into canonical, or our unmodified copy) plus the canonical dir. Only
// call after skillPrunable has confirmed everything is removable.
func removeSkillExposures(ctx context.Context, baseDir, skillName, scope, cwd string, state *InstallState) {
	canonicalDir := filepath.Join(baseDir, skillName)
	for _, agent := range agents.Registry {
		if scope == ScopeProject && !agent.SupportsProjectScope {
			continue
		}
		agentDir, err := agentSkillsDirForScope(ctx, agent, scope, cwd)
		if err != nil {
			continue
		}
		entry := filepath.Join(agentDir, skillName)
		if !exposureRemovable(entry, canonicalDir, skillName, state) {
			continue
		}
		// RemoveAll on a symlink removes the link, not its target.
		if err := os.RemoveAll(entry); err != nil {
			log.Warnf(ctx, "Failed to remove %s: %v", entry, err)
		}
	}
	if err := os.RemoveAll(canonicalDir); err != nil {
		log.Warnf(ctx, "Failed to remove %s: %v", canonicalDir, err)
	}
}

// exposureRemovable reports whether an agent's skill entry is ours and safe to
// remove: absent, a symlink into canonicalDir, or a copied dir whose contents
// exactly match the recorded checksums.
func exposureRemovable(entry, canonicalDir, skillName string, state *InstallState) bool {
	fi, err := os.Lstat(entry)
	if errors.Is(err, fs.ErrNotExist) {
		return true
	}
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(entry)
		if err != nil {
			return false
		}
		abs := target
		if !filepath.IsAbs(target) {
			abs = filepath.Clean(filepath.Join(filepath.Dir(entry), target))
		}
		return abs == canonicalDir || strings.HasPrefix(abs, canonicalDir+string(os.PathSeparator))
	}
	if fi.IsDir() {
		return dirMatchesRecords(entry, skillName, state)
	}
	return false
}

// dirMatchesRecords reports whether dir contains exactly the files recorded for
// skillName, each with its recorded sha256 (no missing, modified, or extra
// files). Returns false when there is no recorded provenance, so unverifiable
// content is never deleted.
func dirMatchesRecords(dir, skillName string, state *InstallState) bool {
	if state == nil {
		return false
	}
	prefix := skillName + "/"
	recorded := map[string]string{}
	for path, rec := range state.Files {
		if after, ok := strings.CutPrefix(path, prefix); ok {
			recorded[after] = rec.SHA256
		}
	}
	if len(recorded) == 0 {
		return false
	}

	seen := 0
	matched := true
	walkErr := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		want, ok := recorded[filepath.ToSlash(rel)]
		if !ok {
			matched = false // a file we didn't write -> not ours, keep
			return filepath.SkipAll
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			matched = false
			return filepath.SkipAll
		}
		seen++
		return nil
	})
	if walkErr != nil || !matched {
		return false
	}
	return seen == len(recorded)
}

// PluginUpdate records that a plugin was updated for an agent.
type PluginUpdate struct {
	Agent   string
	Version string
}

// UpdateInstalledPlugins runs the plugin update for every plugin recorded in the
// given scope's state, bumping each record's version to ref. A plugin that can't
// be updated (CLI missing, etc.) is skipped with a warning, never failed, to
// keep the non-interactive update prompt-free and exit-0 on partial success.
func UpdateInstalledPlugins(ctx context.Context, scope, ref string) ([]PluginUpdate, error) {
	dir, err := skillsDir(ctx, scope)
	if err != nil {
		return nil, err
	}
	state, err := LoadState(dir)
	if err != nil {
		return nil, err
	}
	if state == nil || len(state.Plugins) == 0 {
		return nil, nil
	}

	version := DisplaySkillsVersion(ref)
	var updated []PluginUpdate
	for _, name := range slices.Sorted(maps.Keys(state.Plugins)) {
		agent := agents.ByName(name)
		if agent == nil {
			log.Warnf(ctx, "Skipping unknown agent %q in state", name)
			continue
		}
		if err := UpdatePluginForAgent(ctx, agent); err != nil {
			log.Warnf(ctx, "Skipped %s: %v", agent.DisplayName, err)
			continue
		}
		rec := state.Plugins[name]
		rec.Version = version
		state.Plugins[name] = rec
		updated = append(updated, PluginUpdate{Agent: agent.DisplayName, Version: version})
	}

	if len(updated) > 0 {
		state.LastUpdated = time.Now()
		if err := SaveState(dir, state); err != nil {
			return nil, err
		}
	}
	return updated, nil
}

// FormatUpdateResult returns a human-readable summary of the update result.
// When check is true, output uses "Would update/add" instead of "Updated/Added".
func FormatUpdateResult(result *UpdateResult, check bool) string {
	var lines []string

	updateVerb := "updated"
	addVerb := "added"
	removeVerb := "removed"
	summaryVerb := "Updated"
	removeSummaryVerb := "Removed"
	if check {
		updateVerb = "would update"
		addVerb = "would add"
		removeVerb = "would remove"
		summaryVerb = "Would update"
		removeSummaryVerb = "Would remove"
	}

	for _, u := range result.Updated {
		if u.OldVersion == "" {
			lines = append(lines, fmt.Sprintf("  %s %s -> v%s", updateVerb, u.Name, u.NewVersion))
		} else {
			lines = append(lines, fmt.Sprintf("  %s %s v%s -> v%s", updateVerb, u.Name, u.OldVersion, u.NewVersion))
		}
	}

	for _, a := range result.Added {
		lines = append(lines, fmt.Sprintf("  %s %s v%s", addVerb, a.Name, a.NewVersion))
	}

	for _, r := range result.Removed {
		lines = append(lines, fmt.Sprintf("  %s %s (no longer in this release)", removeVerb, r.Name))
	}

	total := len(result.Updated) + len(result.Added)
	if total == 0 && len(result.Removed) == 0 {
		return "No changes."
	}

	if total > 0 {
		lines = append(lines, fmt.Sprintf("%s %d %s.", summaryVerb, total, skillNoun(total)))
	}
	if len(result.Removed) > 0 {
		lines = append(lines, fmt.Sprintf("%s %d %s.", removeSummaryVerb, len(result.Removed), skillNoun(len(result.Removed))))
	}
	return strings.Join(lines, "\n")
}

func skillNoun(n int) string {
	if n == 1 {
		return "skill"
	}
	return "skills"
}
