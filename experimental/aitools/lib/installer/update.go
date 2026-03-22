package installer

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/databricks/cli/experimental/aitools/lib/agents"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

// UpdateOptions controls the behavior of UpdateSkills.
type UpdateOptions struct {
	Force  bool
	NoNew  bool
	Check  bool     // dry run: show what would change without downloading
	Skills []string // empty = all installed
}

// UpdateResult describes what UpdateSkills did (or would do in check mode).
type UpdateResult struct {
	Updated   []SkillUpdate // skills that were updated
	Added     []SkillUpdate // new skills added (when NoNew is false)
	Unchanged []string      // skills at current version
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
	globalDir, err := GlobalSkillsDir(ctx)
	if err != nil {
		return nil, err
	}

	state, err := LoadState(globalDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load install state: %w", err)
	}

	if state == nil {
		if hasLegacyInstall(ctx, globalDir) {
			return nil, fmt.Errorf("found skills from a previous install without state tracking; run 'databricks experimental aitools install' to refresh before updating")
		}
		return nil, fmt.Errorf("no skills installed. Run 'databricks experimental aitools install' to install")
	}

	latestTag, authoritative, err := src.FetchLatestRelease(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest release: %w", err)
	}

	if !authoritative && !opts.Force {
		cmdio.LogString(ctx, "Could not check for updates (offline?). Use --force to update anyway.")
		return &UpdateResult{Unchanged: sortedKeys(state.Skills)}, nil
	}

	if state.Release == latestTag && !opts.Force {
		cmdio.LogString(ctx, "Already up to date.")
		return &UpdateResult{Unchanged: sortedKeys(state.Skills)}, nil
	}

	manifest, err := src.FetchManifest(ctx, latestTag)
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
	names := sortedKeys(skillSet)

	for _, name := range names {
		meta, inManifest := manifest.Skills[name]
		oldVersion := state.Skills[name]

		if !inManifest {
			_, wasInstalled := state.Skills[name]
			if wasInstalled {
				log.Warnf(ctx, "Warning: %q not found in manifest %s (keeping installed version).", name, latestTag)
			}
			result.Unchanged = append(result.Unchanged, name)
			continue
		}

		// Filter experimental skills unless state opted in.
		if meta.Experimental && !state.IncludeExperimental {
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

		// Check if this is a new skill (not in state).
		_, wasInstalled := state.Skills[name]

		if meta.Version == oldVersion && !opts.Force {
			result.Unchanged = append(result.Unchanged, name)
			continue
		}

		update := SkillUpdate{
			Name:       name,
			OldVersion: oldVersion,
			NewVersion: meta.Version,
		}

		if !wasInstalled {
			result.Added = append(result.Added, update)
		} else {
			result.Updated = append(result.Updated, update)
		}
	}

	if opts.Check {
		return result, nil
	}

	// Download and install updated/added skills.
	allChanges := append(result.Updated, result.Added...)
	for _, change := range allChanges {
		meta := manifest.Skills[change.Name]
		if err := installSkillForAgents(ctx, latestTag, change.Name, meta.Files, targetAgents, globalDir); err != nil {
			return nil, err
		}
	}

	// Update state.
	state.Release = latestTag
	state.LastUpdated = time.Now()
	for _, change := range allChanges {
		state.Skills[change.Name] = change.NewVersion
	}
	if err := SaveState(globalDir, state); err != nil {
		return nil, err
	}

	return result, nil
}

// buildUpdateSkillSet determines which skills to consider for update.
func buildUpdateSkillSet(state *InstallState, manifest *Manifest, opts UpdateOptions) map[string]bool {
	skillSet := make(map[string]bool)

	if len(opts.Skills) > 0 {
		// Only named skills.
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

// sortedKeys returns the keys of a map sorted alphabetically.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// FormatUpdateResult returns a human-readable summary of the update result.
// When check is true, output uses "Would update/add" instead of "Updated/Added".
func FormatUpdateResult(result *UpdateResult, check bool) string {
	var lines []string

	updateVerb := "updated"
	addVerb := "added"
	summaryVerb := "Updated"
	if check {
		updateVerb = "would update"
		addVerb = "would add"
		summaryVerb = "Would update"
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

	total := len(result.Updated) + len(result.Added)
	if total == 0 {
		return "No changes."
	}

	noun := "skills"
	if total == 1 {
		noun = "skill"
	}
	lines = append(lines, fmt.Sprintf("%s %d %s.", summaryVerb, total, noun))
	return strings.Join(lines, "\n")
}
