package cmdio

import (
	"encoding/json"
	"fmt"
	"strings"
)

// applyInputOnlyMask returns v with the listed dotted paths removed. If
// paths is empty, v is returned unchanged. Otherwise v is round-tripped
// through JSON into a generic representation, the paths are deleted, and
// the masked value is returned for the caller to marshal in its preferred
// format.
//
// Paths use dotted notation (e.g. "stable_url.initial_workspace_id").
// Arrays are traversed transparently: a single path applies to every
// element of any array encountered along the way, so list responses share
// the same path expression as singletons.
func applyInputOnlyMask(v any, paths []string) (any, error) {
	if len(paths) == 0 {
		return v, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("input-only mask: marshal: %w", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("input-only mask: unmarshal: %w", err)
	}
	for _, p := range paths {
		deletePath(out, strings.Split(p, "."))
	}
	return out, nil
}

// deletePath walks v according to keys and removes the leaf key from any
// object it lands on. Missing intermediate keys are a no-op; arrays are
// traversed transparently with the same remaining key list.
func deletePath(v any, keys []string) {
	if len(keys) == 0 {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		if len(keys) == 1 {
			delete(t, keys[0])
			return
		}
		if child, ok := t[keys[0]]; ok {
			deletePath(child, keys[1:])
		}
	case []any:
		for _, el := range t {
			deletePath(el, keys)
		}
	}
}
