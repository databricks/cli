package installer

import (
	"context"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

// outdatedLookupTimeout bounds the release lookup of OutdatedTooling: one
// compatibility manifest fetch and, when the manifest tracks the latest
// release, one GitHub API call. The lookup runs once a coding agent's command
// has ended, and a machine that cannot reach GitHub waits this long, so it is
// kept short. Tests replace it.
var outdatedLookupTimeout = time.Second

// OutdatedTooling reports whether the Databricks AI tooling that the CLI
// installed for the agent in the global scope is older than the release
// `databricks aitools update` installs (GetSkillsRef), with both releases for
// display.
//
// The installed release comes from the global install state: the agent's
// plugin record when it has one, else the raw skills release. The plugin
// record holds the release of the last `aitools install` or `aitools update`,
// not what the agent's own plugin CLI delivered since, and the raw skills are
// shared by every agent on the machine. A plugin that the agent's own CLI
// installed (agents.Agent.DatabricksPluginConfigured) has no record, so its
// release is unknown. An installation whose release is unknown is never
// reported, and costs no lookup. The lookup can reach the network;
// outdatedLookupTimeout bounds it.
func OutdatedTooling(ctx context.Context, agentName string) (installed, available string, outdated bool) {
	dir, err := GlobalSkillsDir(ctx)
	if err != nil {
		return "", "", false
	}
	state, err := LoadState(dir)
	if err != nil {
		log.Debugf(ctx, "Could not load AI tools install state in %s: %v", dir, err)
		return "", "", false
	}
	if state == nil {
		return "", "", false
	}
	if rec, ok := state.Plugins[agentName]; ok {
		installed = semver.Canonical("v" + DisplaySkillsVersion(rec.Version))
	} else if a := agents.ByName(agentName); a != nil {
		// A plugin that the agent's own CLI installed has no record, and the
		// raw skills of the machine do not tell its release.
		if configured, _ := a.DatabricksPluginConfigured(ctx); !configured {
			installed, _ = installedSkillsRelease(state)
		}
	}
	if installed == "" {
		return "", "", false
	}

	lookupCtx, cancel := context.WithTimeout(ctx, outdatedLookupTimeout)
	defer cancel()
	ref, _, err := GetSkillsRef(lookupCtx)
	if err != nil {
		log.Debugf(ctx, "Could not resolve the skills release to compare the installed one with: %v", err)
		return "", "", false
	}
	// A manifest fetch that runs out of time falls back to a cached or embedded
	// manifest without an error, and that one can pin an older release. The
	// update would wait on the same network.
	if lookupCtx.Err() != nil {
		return "", "", false
	}
	// A branch ref has no release order. It is not semver, so it compares
	// lowest and the installation is never older than it.
	available = semver.Canonical("v" + DisplaySkillsVersion(ref))
	if semver.Compare(installed, available) >= 0 {
		return "", "", false
	}
	return DisplaySkillsVersion(installed), DisplaySkillsVersion(available), true
}
