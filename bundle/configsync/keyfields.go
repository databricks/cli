package configsync

import (
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/registry"
)

// A keyed-slice element is addressed in a change path as [='value']: the key field is
// omitted because resolution is by value. configsync works on dynamic values without a
// Go type, so it recovers the key field from the element itself — the member the
// registry recognises as a key field. These helpers implement that lookup for the two
// element representations configsync deals with: a dyn.Value (the merged config) and a
// decoded map (a change payload).

// dynElementKeyValue returns the value of the element's key field, for matching an
// element against the value in a [='value'] selector. ok is false when the element is
// not a mapping or has no registered key field set.
func dynElementKeyValue(elem dyn.Value) (value string, ok bool) {
	m, isMap := elem.AsMap()
	if !isMap {
		return "", false
	}
	for _, name := range sortedMappingKeys(m) {
		if !registry.IsKeyField(name) {
			continue
		}
		if pair, ok := m.GetPairByString(name); ok {
			if s, ok := pair.Value.AsString(); ok {
				return s, true
			}
		}
	}
	return "", false
}

// mapKeyField returns the name of the element's key field for a decoded change payload.
// It returns "" when value is not a map or has no registered key field.
func mapKeyField(value any) string {
	m, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		if registry.IsKeyField(name) {
			return name
		}
	}
	return ""
}

// sortedMappingKeys returns the mapping's string keys in sorted order, so a lookup that
// scans them is deterministic when several key fields are present.
func sortedMappingKeys(m dyn.Mapping) []string {
	keys := make([]string, 0, m.Len())
	for _, k := range m.Keys() {
		if s, ok := k.AsString(); ok {
			keys = append(keys, s)
		}
	}
	slices.Sort(keys)
	return keys
}
