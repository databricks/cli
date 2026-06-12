// Package inputonly drops fields marked INPUT_ONLY in the OpenAPI spec from
// SDK response values before they are rendered to the user.
//
// The Databricks Go SDK uses a single struct per resource for both request and
// response (transport-layer pattern). Some fields are REQUIRED on the request
// side — so their JSON tags have no omitempty — but INPUT_ONLY on the response
// side, meaning the server never populates them. When the CLI hands such a
// struct straight to encoding/json it emits the zero value (`"foo": ""`),
// which leaks API surface that isn't meant to round-trip.
//
// This package is consumed by the generated CLI command code in cmd/account/**
// and cmd/workspace/**: cligen reads the schemas in .codegen/cli.json, walks
// the response type, and emits a Strip call before the existing cmdio.Render.
// Keeping the logic out of libs/cmdio matches @pietern's guidance — cmdio
// stays a generic rendering pipeline, and the filtering policy lives where the
// metadata (cli.json) does.
package inputonly

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Strip returns v with the listed dotted paths removed. If paths is empty, v
// is returned unchanged. Otherwise v is round-tripped through JSON into a
// generic representation, the listed paths are deleted, and the masked value
// is returned for the caller to marshal in its preferred format.
//
// Paths use dotted notation (e.g. "stable_url.initial_workspace_id"). Arrays
// and dynamically-keyed maps (e.g. proto map<string, V>) are traversed
// transparently: a single path applies to every element of an array, and to
// every value of a map when no literal key matches the next path component.
// List responses and map-valued fields therefore share the same path
// expression as singletons.
func Strip(v any, paths []string) (any, error) {
	if len(paths) == 0 {
		return v, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("inputonly: marshal: %w", err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("inputonly: unmarshal: %w", err)
	}
	for _, p := range paths {
		deletePath(out, strings.Split(p, "."))
	}
	return out, nil
}

// deletePath walks v according to keys and removes the leaf key from any
// object it lands on.
//
// Both arrays and dynamically-keyed maps are traversed transparently:
//
//   - When v is a []any, every element is visited with the same key list.
//   - When v is a map[string]any but the next key is not a literal match,
//     every value is visited with the same key list — this handles proto
//     map<string, V> fields, whose JSON keys are user-provided strings and
//     whose values carry the field name from the path.
//
// Both struct fields and proto map<string, V> surface as map[string]any after
// json.Unmarshal, so a single corner case remains: if a map's user-provided
// key happens to equal an inner field name, the literal match wins and that
// entry is removed instead of the field inside each value. cligen emits paths
// from the schema, so this only fires for real-world key collisions and
// matches the expected behavior for any path the schema actually targets.
func deletePath(v any, keys []string) {
	if len(keys) == 0 {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		if child, ok := t[keys[0]]; ok {
			if len(keys) == 1 {
				delete(t, keys[0])
			} else {
				deletePath(child, keys[1:])
			}
			return
		}
		for _, child := range t {
			deletePath(child, keys)
		}
	case []any:
		for _, el := range t {
			deletePath(el, keys)
		}
	}
}
