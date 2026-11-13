package testserver

import (
	"encoding/json"
	"strings"
)

// echoedSpec returns the spec the real API echoes back on GET after a create or
// update. Since 2026-11-12 the Lakebase API returns spec as stored: the fields the
// caller set (a field masked but absent from the body keeps its previous value),
// minus inputOnly fields that are consulted once and never echoed. Defaults the
// backend materializes appear only in status, not in spec.
//
// mask is the update_mask ("" on create, which takes the body's spec as is).
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
					parts := strings.Split(strings.TrimPrefix(m, "spec."), ".")
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

	// A oneof holds one member: whichever the body set wins over the stored sibling.
	for _, members := range oneofGroups {
		var set []string
		for _, m := range members {
			if _, ok := specMapGet(bodySpec, strings.Split(strings.TrimPrefix(m, "spec."), ".")); ok {
				set = append(set, m)
			}
		}
		if len(set) == 0 {
			continue
		}
		for _, m := range members {
			if m != set[0] {
				specMapDelete(result, strings.Split(strings.TrimPrefix(m, "spec."), "."))
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
