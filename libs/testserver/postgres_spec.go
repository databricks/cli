package testserver

import (
	"encoding/json"
	"slices"
	"strings"
)

// echoedSpec returns the spec the real API echoes back on GET after a create or
// update. Since 2026-11-12 the API returns spec as stored for projects, branches,
// endpoints, databases and roles (catalogs, data APIs and synced tables were not
// observed to change and still return status only): the fields the
// caller set (a field masked but absent from the body keeps its previous value),
// minus inputOnly fields that are consulted once and never echoed. Defaults the
// backend materializes appear only in status, not in spec.
//
// mask is the update_mask. "" takes the body's spec as is: on create, and on an
// unmasked update, matching how the status of databases and roles is replaced.
// oneofGroups uses the same "spec."-prefixed paths as the update mask validators;
// setting one member of a group clears its siblings.
func echoedSpec[T any](stored *T, body []byte, mask string, oneofGroups map[string][]string, inputOnly ...string) *T {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return stored
	}
	bodySpec, ok := req["spec"].(map[string]any)
	if !ok {
		return stored
	}

	result := map[string]any{}
	if mask == "" {
		result = bodySpec
	} else {
		if stored != nil {
			raw, err := json.Marshal(stored)
			if err != nil {
				return stored
			}
			if err := json.Unmarshal(raw, &result); err != nil {
				return stored
			}
		}
		for path := range strings.SplitSeq(mask, ",") {
			path = strings.TrimSpace(path)
			if path == "spec" || path == "*" {
				mergeSpecMap(result, bodySpec)
				continue
			}
			rel, ok := strings.CutPrefix(path, "spec.")
			if !ok {
				continue
			}
			if members, isGroup := oneofGroups[path]; isGroup {
				for _, m := range members {
					parts := specPathParts(m)
					if v, ok := specMapGet(bodySpec, parts); ok {
						specMapSet(result, parts, v)
					}
				}
				continue
			}
			parts := strings.Split(rel, ".")
			if v, ok := specMapGet(bodySpec, parts); ok {
				specMapSet(result, parts, v)
			}
		}
	}

	// A oneof holds one member: when the request sets the group, its member replaces
	// the stored sibling. A group the mask does not cover keeps its stored member.
	for group, members := range oneofGroups {
		if mask != "" && !maskSetsGroup(mask, group, members) {
			continue
		}
		var set string
		for _, m := range members {
			if _, ok := specMapGet(bodySpec, specPathParts(m)); ok {
				set = m
				break
			}
		}
		if set == "" {
			continue
		}
		for _, m := range members {
			if m != set {
				specMapDelete(result, specPathParts(m))
			}
		}
	}

	for _, field := range inputOnly {
		delete(result, field)
	}
	if len(result) == 0 {
		return nil
	}

	raw, err := json.Marshal(result)
	if err != nil {
		return stored
	}
	var spec T
	if err := json.Unmarshal(raw, &spec); err != nil {
		return stored
	}
	return &spec
}

// maskSetsGroup reports whether update_mask covers the oneof group: the whole spec,
// the group itself, or one of its members.
func maskSetsGroup(mask, group string, members []string) bool {
	for path := range strings.SplitSeq(mask, ",") {
		path = strings.TrimSpace(path)
		if path == "spec" || path == "*" || path == group || slices.Contains(members, path) {
			return true
		}
	}
	return false
}

// specPathParts splits a "spec."-prefixed mask path into keys relative to the spec.
func specPathParts(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "spec."), ".")
}

// mergeSpecMap deep-merges src into dst; keys absent from src keep their dst value.
func mergeSpecMap(dst, src map[string]any) {
	for k, v := range src {
		srcChild, srcIsMap := v.(map[string]any)
		dstChild, dstIsMap := dst[k].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeSpecMap(dstChild, srcChild)
			continue
		}
		dst[k] = v
	}
}

// specMapGet returns the value at parts in m, if present.
func specMapGet(m map[string]any, parts []string) (any, bool) {
	var node any = m
	for _, p := range parts {
		mm, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		node, ok = mm[p]
		if !ok {
			return nil, false
		}
	}
	return node, true
}

// specMapSet sets the value at parts in m, creating intermediate maps.
func specMapSet(m map[string]any, parts []string, v any) {
	for _, p := range parts[:len(parts)-1] {
		child, ok := m[p].(map[string]any)
		if !ok {
			child = map[string]any{}
			m[p] = child
		}
		m = child
	}
	m[parts[len(parts)-1]] = v
}

// specMapDelete removes the value at parts from m, if present.
func specMapDelete(m map[string]any, parts []string) {
	for _, p := range parts[:len(parts)-1] {
		child, ok := m[p].(map[string]any)
		if !ok {
			return
		}
		m = child
	}
	delete(m, parts[len(parts)-1])
}

// Spec fields the API consults once and does not echo back on GET.
var (
	projectInputOnlySpecFields = []string{"pg_version"}
	branchInputOnlySpecFields  = []string{"source_branch", "source_branch_lsn", "source_branch_time"}
	roleInputOnlySpecFields    = []string{"postgres_role", "auth_method", "identity_type"}
)
