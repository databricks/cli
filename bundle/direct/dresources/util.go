package dresources

import (
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/retries"
)

type StateLifecycle struct {
	Started *bool `json:"started,omitempty"`
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
//
// A parent path is dropped when a more specific child path is also being updated. Masking a
// message asks the API to replace it wholesale, so the API then requires every field under
// that message to be populated in the request body — verified against a real workspace on
// 2026-08-31, where update_mask=spec.default_endpoint_settings with a body carrying only
// the autoscaling limits was rejected with "Field
// 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in
// request". A bundle sends only the fields it declares, so only the leaf may be masked.
//
// Sorted, so the generated update_mask does not depend on map iteration order.
func collectUpdatePathsWithPrefix(changes Changes, prefix string) []string {
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
			paths = append(paths, prefix+maskPath(path))
		}
	}
	slices.Sort(paths)
	// Truncating subscripts can map two changed entries of the same map onto one path.
	return slices.Compact(paths)
}

// maskPath converts a change path into the path the API accepts in update_mask. A map or
// repeated field is addressable only as a whole, so everything from the first subscript
// onwards is dropped: settings.pg_settings['work_mem'] is masked as settings.pg_settings.
// Verified against a real workspace on 2026-08-31, which answers the indexed form with
// "Unknown field path in update_mask: 'spec.settings.pg_settings['work_mem']'".
func maskPath(path string) string {
	if before, _, ok := strings.Cut(path, "["); ok {
		return before
	}
	return path
}
