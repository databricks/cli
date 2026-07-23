package dresources

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/retries"
)

type StateLifecycle struct {
	Started *bool `json:"started,omitempty"`
}

// pathLifecycleStarted is the change path of the synthetic lifecycle.started field.
var pathLifecycleStarted = structpath.MustParsePath("lifecycle.started")

// offlineLifecycleTransition reports whether an offline (--planmode=offline)
// DoUpdate should fire a lifecycle Start/Stop. Offline has no live remote status
// to compare against, so the decision is driven purely by the local diff: fire
// only when lifecycle.started actually changed. This keeps offline from issuing
// a spurious transition (and from saving a lifecycle value it never applied).
func offlineLifecycleTransition(entry *PlanEntry) bool {
	return entry.Changes.HasChange(pathLifecycleStarted)
}

// isLifecycleRaceErr reports whether err is the backend's "already in that
// state" rejection of a lifecycle Start/Stop/Delete. Offline (--planmode=offline)
// fires a transition based on the config change alone, without reading remote, so
// the resource may already be in the desired state; the backend then rejects the
// call with INVALID_STATE. The same race is possible in full mode between the
// remote read and the transition.
func isLifecycleRaceErr(err error) bool {
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.ErrorCode == "INVALID_STATE"
}

// tolerateLifecycleRace swallows an "already in that state" error (see
// isLifecycleRaceErr) from a lifecycle Start/Stop/Delete, logging a warning and
// continuing. Used by resources whose transition is a single call.
func tolerateLifecycleRace(ctx context.Context, resourceKey string, err error) error {
	if err == nil || isLifecycleRaceErr(err) {
		if err != nil {
			log.Warnf(ctx, "%s: lifecycle transition rejected as already in desired state: %v", resourceKey, err)
		}
		return nil
	}
	return err
}

// This is copied from the retries package of the databricks-sdk-go. It should be made public,
// but for now, I'm copying it here.
func shouldRetry(err error) bool {
	if e, ok := errors.AsType[*retries.Err](err); ok {
		return !e.Halt
	}
	return false
}

// collectUpdatePathsWithPrefix extracts field paths from Changes that have action=Update,
// adding a prefix to each path. This is used when the state type has a flattened structure
// but the API expects paths relative to a nested object (e.g., "spec.display_name").
func collectUpdatePathsWithPrefix(changes Changes, prefix string) []string {
	var paths []string
	for path, change := range changes {
		if change.Action == deployplan.Update {
			paths = append(paths, prefix+path)
		}
	}
	return paths
}

// collectLeafUpdatePathsWithPrefix is like collectUpdatePathsWithPrefix but drops a parent
// path when a more specific child path is also being updated, and sorts the result.
//
// The Postgres Role PATCH endpoint rejects an update_mask that lists both a struct and one
// of its sub-fields, since the parent already implies the whole subtree. E.g. {"attributes",
// "attributes.createdb"} collapses to {"attributes.createdb"}. Sorting keeps the generated
// update_mask stable regardless of map iteration order.
func collectLeafUpdatePathsWithPrefix(changes Changes, prefix string) []string {
	var paths []string
	for path, change := range changes {
		if change.Action != deployplan.Update {
			continue
		}
		hasChild := false
		for other := range changes {
			if other == path || changes[other].Action != deployplan.Update {
				continue
			}
			if strings.HasPrefix(other, path+".") {
				hasChild = true
				break
			}
		}
		if !hasChild {
			paths = append(paths, prefix+path)
		}
	}
	slices.Sort(paths)
	return paths
}
