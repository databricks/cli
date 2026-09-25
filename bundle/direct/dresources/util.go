package dresources

import (
	"errors"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structdiff"
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
// The plan records an edited block as a single block-level change, but masking a message
// asks the API to replace it wholesale, so the API then requires every field under that
// message to be populated in the request body — verified against a real workspace on
// 2026-08-31, where update_mask=spec.default_endpoint_settings with a body carrying only
// the autoscaling limits was rejected with "Field
// 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in
// request". A bundle sends only the fields it declares, so expandUpdatePaths decomposes a
// block change back into the changed leaves, and only those are masked.
//
// oneofGroups renames a change path to the group it belongs to, for fields the API only
// accepts under their oneof group name; see the per-resource maps below.
//
// Sorted, so the generated update_mask does not depend on map iteration order.
func collectUpdatePathsWithPrefix(changes Changes, prefix string, oneofGroups map[string]string) []string {
	var paths []string
	for path, change := range changes {
		if change.Action != deployplan.Update {
			continue
		}
		// The plan collapses a whole added/edited block to one block-level entry, but the
		// mask must name the changed leaves, not the enclosing message (masking a message
		// makes the API require every field under it in the body). Recover the leaves by
		// diffing the change's Old against New; a scalar change yields the path itself.
		for _, leaf := range expandUpdatePaths(path, change) {
			masked := maskPath(leaf)
			if group, ok := oneofGroups[masked]; ok {
				masked = group
			}
			paths = append(paths, prefix+masked)
		}
	}
	slices.Sort(paths)
	// Truncating subscripts can map two changed entries of the same map onto one path, and
	// two members of one oneof collapse onto their group.
	return slices.Compact(paths)
}

// expandUpdatePaths returns the changed leaf paths under a change at path. A block-level
// change (Old and New are structs/maps that differ at nested fields) expands to those
// fields; a scalar change, or a whole add/remove that does not descend, yields path itself.
func expandUpdatePaths(path string, change *deployplan.ChangeDesc) []string {
	// A whole add (Old nil) or whole remove (New nil) has nothing to descend into: the
	// block itself is the changed unit, so it masks as path. Only a block present on both
	// sides expands to the inner leaves that actually differ.
	if change.Old == nil || change.New == nil {
		return []string{path}
	}
	diffs, err := structdiff.GetStructDiff(change.Old, change.New, nil)
	if err != nil || len(diffs) == 0 {
		return []string{path}
	}
	out := make([]string, 0, len(diffs))
	for _, d := range diffs {
		sub := d.Path.String()
		switch {
		case sub == "":
			out = append(out, path)
		case strings.HasPrefix(sub, "["):
			out = append(out, path+sub)
		default:
			out = append(out, path+"."+sub)
		}
	}
	return out
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
