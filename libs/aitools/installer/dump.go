package installer

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"slices"

	"github.com/databricks/cli/libs/cmdio"
)

// DumpSkillsToPath writes resolved skill files into destDir and returns the
// number of skills written. It is a dumb dump: no agents, no .state.json, no
// lifecycle (update/list/uninstall ignore it). It honors DATABRICKS_SKILLS_REF,
// the cli-compat version pin, --skills cherry-picks, and --experimental exactly
// as the agent install path does, since it reuses resolveSkills.
func DumpSkillsToPath(ctx context.Context, src ManifestSource, destDir string, opts InstallOptions) (int, error) {
	ref, explicit, err := GetSkillsRef(ctx)
	if err != nil {
		return 0, err
	}
	cmdio.LogString(ctx, "Using skills version "+DisplaySkillsVersion(ref))

	manifest, ref, err := FetchSkillsManifestWithFallback(ctx, src, ref, !explicit)
	if err != nil {
		return 0, err
	}

	targetSkills, err := resolveSkills(ctx, manifest.Skills, opts)
	if err != nil {
		return 0, err
	}

	names := slices.Sorted(maps.Keys(targetSkills))
	for _, name := range names {
		meta := targetSkills[name]
		if _, err := installSkillToDir(ctx, ref, meta.RepoDir, meta.SourceName, filepath.Join(destDir, name), meta.Files); err != nil {
			return 0, err
		}
	}

	noun := "skills"
	if len(names) == 1 {
		noun = "skill"
	}
	cmdio.LogString(ctx, fmt.Sprintf("Wrote %d %s to %s", len(names), noun, filepath.ToSlash(destDir)))
	return len(names), nil
}
