package dresources

import (
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
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

// forceSendClearedFields returns ForceSendFields entries for the request fields the plan
// is clearing: the plan reports a change, but the value is zero, so omitempty would drop
// the field from the payload entirely.
//
// A partial-update API reads an absent field as "leave unchanged", so dropping it
// silently discards the clear and the next plan reports the same drift forever. Unity
// Catalog goes further and rejects a payload carrying no field at all with
// "<Operation> Nothing to update." (400), which made a schema, catalog or volume
// undeployable as soon as someone set a description on it outside the bundle (#6340).
//
// Every omitempty field assigned from config into a partial-update request shares the
// problem -- roughly 40 of them across the UC resources, not just comment -- so this is
// driven off the plan instead of naming individual fields. Resources whose update
// replaces the whole object (jobs, pipelines, model serving) need none of it: there an
// absent field already means cleared, and they converge on their own.
//
// Matching is by the request type's own JSON names, so a field the request does not
// expose, or names differently (e.g. new_name), is never force-sent. Skipped changes are
// excluded by HasChange, so a field the plan suppressed -- via backend_defaults or
// ignore_remote_changes -- is left alone rather than being cleared behind the plan's back.
func forceSendClearedFields(request any, changes Changes) []string {
	value := reflect.ValueOf(request).Elem()
	valueType := value.Type()

	var result []string
	for i := range valueType.NumField() {
		field := valueType.Field(i)
		name := structtag.JSONTag(field.Tag.Get("json")).Name()
		if name == "" || name == "-" || !value.Field(i).IsZero() {
			continue
		}
		if changes.HasChange(structpath.NewDotString(nil, name)) {
			result = append(result, field.Name)
		}
	}

	slices.Sort(result)
	return result
}
