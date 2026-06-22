package configsync

import (
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/direct/dstate"
)

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
// A selector that matches no deployed resource is an error: only deployed
// resources have an id, so a selector matching nothing is a caller mistake.
// Duplicate selectors are deduplicated; the returned keys preserve the order in
// which their selectors first appear.
func ResolveResourceSelectors(state *dstate.DeploymentState, selectors []string) ([]string, error) {
	// Index deployed resources by "<type>:<id>". State keys have the form
	// "resources.<type>.<name>"; indexing by the <type> component means a
	// selector can only ever match a resource of that exact type, never an id
	// that happens to collide across types.
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

	keys := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		resourceType, id, ok := strings.Cut(selector, ":")
		if !ok || resourceType == "" || id == "" {
			return nil, fmt.Errorf("invalid --select-ids value %q, expected <type>:<id> (e.g. jobs:123456789)", selector)
		}
		key, ok := byTypeID[selector]
		if !ok {
			return nil, fmt.Errorf("no deployed %s resource with id %s", resourceType, id)
		}
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
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
// dependencies matter only when planning (DetectChanges always plans the full
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
