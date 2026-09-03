package dresources

import (
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
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
//
// A parent path is dropped when a more specific child path is also being updated. Masking a
// message asks the API to replace it wholesale, so the API then requires every field under
// that message to be populated in the request body — verified against a real workspace on
// 2026-08-31, where update_mask=spec.default_endpoint_settings with a body carrying only
// the autoscaling limits was rejected with "Field
// 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in
// request". A bundle sends only the fields it declares, so only the leaf may be masked.
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
			masked := maskPath(path)
			// A change that lands on a message rather than on a field inside it masks the
			// leaves the body populates, not the message: see expandMaskedMessage.
			if leaves := expandMaskedMessage(masked, change.New); len(leaves) > 0 {
				for _, leaf := range leaves {
					paths = append(paths, prefix+oneofGroup(oneofGroups, leaf))
				}
				continue
			}
			paths = append(paths, prefix+oneofGroup(oneofGroups, masked))
		}
	}
	slices.Sort(paths)
	// Truncating subscripts can map two changed entries of the same map onto one path, and
	// two members of one oneof collapse onto their group.
	return slices.Compact(paths)
}

// oneofGroup returns the group a field is masked under, or the path itself when the field
// does not belong to a oneof.
func oneofGroup(oneofGroups map[string]string, path string) string {
	if group, ok := oneofGroups[path]; ok {
		return group
	}
	return path
}

// expandMaskedMessage returns the paths of the leaves that value populates, relative to
// path, or nil when value is not a nested message so that the caller keeps path as it is.
//
// Masking a message asks the API to replace it wholesale, and the API then requires every
// field under it to be populated in the request body — which a bundle that declares only
// some of them never satisfies. A change lands on the message itself, rather than on a
// field inside it, when the block is added to the configuration as a whole. Verified
// against a real workspace on 2026-09-03, where a config growing a
// default_endpoint_settings block with just an autoscaling limit produced
// update_mask=spec.default_endpoint_settings and was rejected with "Field
// 'spec.default_endpoint_settings.suspension' is in update_mask but not provided in
// request".
//
// A map or a repeated field is addressable only as a whole (see maskPath), and a wrapper
// such as duration.Duration is a struct in Go but a scalar on the wire. Neither expands.
func expandMaskedMessage(path string, value any) []string {
	typ := reflect.TypeOf(value)
	body, ok := messageBody(typ, value)
	if !ok {
		return nil
	}
	var paths []string
	appendPopulatedLeaves(&paths, path, derefType(typ), body)
	return paths
}

// messageBody returns value marshalled as a JSON object, reporting false when value is not
// a struct or does not marshal to an object.
func messageBody(typ reflect.Type, value any) (map[string]any, bool) {
	if typ == nil || derefType(typ).Kind() != reflect.Struct {
		return nil, false
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, false
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, false
	}
	return body, true
}

// appendPopulatedLeaves appends the path of every leaf present in body. A field is a leaf
// unless it is a struct in Go and an object on the wire, which is what a nested message is.
func appendPopulatedLeaves(paths *[]string, prefix string, typ reflect.Type, body map[string]any) {
	for name, value := range body {
		path := prefix + "." + name
		fieldType, found := jsonFieldType(typ, name)
		nested, isObject := value.(map[string]any)
		if found && isObject && derefType(fieldType).Kind() == reflect.Struct {
			appendPopulatedLeaves(paths, path, derefType(fieldType), nested)
			continue
		}
		*paths = append(*paths, path)
	}
}

// jsonFieldType returns the type of the field typ serializes under the given JSON name.
func jsonFieldType(typ reflect.Type, name string) (reflect.Type, bool) {
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Anonymous {
			embedded := derefType(field.Type)
			if embedded.Kind() != reflect.Struct {
				continue
			}
			if fieldType, ok := jsonFieldType(embedded, name); ok {
				return fieldType, true
			}
			continue
		}
		if structtag.JSONTag(field.Tag.Get("json")).Name() == name {
			return field.Type, true
		}
	}
	return nil, false
}

func derefType(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ
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
