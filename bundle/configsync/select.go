package configsync

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/log"
)

// IndexDeployedResources indexes the deployed resources by "<type>:<id>",
// mapping to the plan key ("resources.<type>.<name>"). Indexing by the <type>
// component means a selector can only ever match a resource of that exact type,
// never an id that happens to collide across types.
func IndexDeployedResources(state *dstate.DeploymentState) map[string]string {
	byTypeID := make(map[string]string)
	for key := range state.Data.State {
		id := state.GetResourceID(key)
		if id == "" {
			continue
		}
		typeAndName, ok := strings.CutPrefix(key, "resources.")
		if !ok {
			continue
		}
		resourceType, _, ok := strings.Cut(typeAndName, ".")
		if !ok {
			continue
		}
		byTypeID[resourceType+":"+id] = key
	}
	return byTypeID
}

// ResolveResourceSelectors maps "<type>:<id>" selectors to their plan keys
// ("resources.<type>.<name>") using the open deployment state (see
// OpenDeploymentState).
//
// Selection uses both the resource type and the deployed resource id (a job id,
// pipeline id, dashboard id, ...) — the pair the workspace UI knows from a
// resource's page. The type is required because a resource id is only unique
// within a type: an id that happens to collide across types (e.g. a job and a
// warehouse) would otherwise select the wrong resource. <type> is the bundle
// resource type as it appears in the plan key, e.g. "jobs" or "pipelines". This
// is also why selection is independent from `bundle deploy --select`, which
// matches "type.name" keys.
//
// A selector that matches no deployed resource is skipped rather than failing
// the run — but only when at least one other selector did match. The workspace
// UI batches every edited resource into one sync, so a single stale selector
// (a resource deleted remotely, or whose deploy state has drifted) must not
// drop the valid resources' edits. When NO selector matches, the error is
// returned instead: silently reporting "no changes" would let the UI show a
// success while nothing synced, hiding a real state problem. A malformed
// selector (missing "<type>:<id>" shape) is always an error, because that is a
// caller mistake rather than drift.
// Duplicate selectors are deduplicated; the returned keys preserve the order in
// which their selectors first appear.
func ResolveResourceSelectors(ctx context.Context, state *dstate.DeploymentState, selectors []string) ([]string, error) {
	byTypeID := IndexDeployedResources(state)

	keys := make([]string, 0, len(selectors))
	var missing []string
	for _, selector := range selectors {
		resourceType, id, ok := strings.Cut(selector, ":")
		if !ok || resourceType == "" || id == "" {
			return nil, fmt.Errorf("invalid --select-ids value %q, expected <type>:<id> (e.g. jobs:123456789)", selector)
		}
		key, ok := byTypeID[selector]
		if !ok {
			missing = append(missing, selector)
			continue
		}
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	// No selector matched a deployed resource: fail loudly instead of syncing
	// nothing, so the caller does not report a spurious success.
	if len(keys) == 0 {
		resourceType, id, _ := strings.Cut(missing[0], ":")
		return nil, fmt.Errorf("no deployed %s resource with id %s", resourceType, id)
	}

	// Some selectors matched: skip the stale ones so the matched resources still
	// sync (the UI batches several resources into one run).
	for _, selector := range missing {
		log.Debugf(ctx, "config-remote-sync: skipping selector %q, no deployed resource with that id", selector)
	}
	return keys, nil
}

// FilterChanges returns the subset of changes that belong to the resources in
// selected, a list of plan keys ("resources.<type>.<name>") as returned by
// ResolveResourceSelectors. Change keys are plan keys too.
//
// Selection is at the resource level: a change node is kept when it is the
// selected resource itself or any node beneath it, matched by prefix on the "."
// path boundary. This groups a resource's permissions/grants sub-nodes
// ("resources.<type>.<name>.permissions") with their parent, so selecting a
// resource never silently skips its permissions. The "." boundary stops
// "resources.jobs.foo" from also matching an unrelated "resources.jobs.foobar".
//
// Only the selected resources' own nodes are kept: unlike deploy's plan
// filtering, the selection is not expanded to transitive dependencies, because
// dependencies matter only when planning (the plan always covers the full
// resource set so ${resources.*} references resolve), not when deciding which
// resources' configuration may be rewritten.
func FilterChanges(changes Changes, selected []string) Changes {
	filtered := make(Changes)
	for key, resourceChanges := range changes {
		for _, sel := range selected {
			if key == sel || strings.HasPrefix(key, sel+".") {
				filtered[key] = resourceChanges
				break
			}
		}
	}
	return filtered
}
