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
package inputonly

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Strip returns v with the listed dotted paths removed. If paths is empty, v
// is returned unchanged. Otherwise v is round-tripped through JSON into a
// generic representation, the listed paths are deleted, and the masked value
// is returned for the caller to marshal in its preferred format.
//
// Paths use dotted notation (e.g. "stable_url.initial_workspace_id") and are
// matched literally at every segment. Arrays are traversed transparently: a
// path is applied to every element of any []any encountered along the way.
// Maps are not — see deletePath for the reasoning.
//
// Numbers are decoded via json.Number rather than float64 so values above
// 2^53 (e.g. SDK fields typed int64 like spark_context_id) re-marshal
// verbatim instead of silently losing precision.
//
// Side note: encoding/json marshals struct fields in declaration order but
// map[string]any keys alphabetically. Filtered responses therefore render
// with sorted JSON keys, while unfiltered ones keep SDK-struct order.
// Acceptance fixtures and downstream consumers should be tolerant of that.
func Strip(v any, paths []string) (any, error) {
	if len(paths) == 0 {
		return v, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("inputonly: marshal: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var out any
	if err := dec.Decode(&out); err != nil {
		return nil, fmt.Errorf("inputonly: unmarshal: %w", err)
	}
	for _, p := range paths {
		deletePath(out, strings.Split(p, "."))
	}
	return out, nil
}

// deletePath walks v according to keys and removes the leaf key from the
// object it lands on. Each segment is matched literally; if no literal match
// exists at a given level the path simply does not apply, and we return.
//
// Arrays are traversed transparently. After json.Unmarshal a JSON array is
// always []any, type-distinguishable from map[string]any, so descending into
// every element with the same key list is unambiguous.
//
// Maps are not traversed transparently, even though proto map<string, V>
// surfaces as map[string]any just like a struct does. Falling back to
// match-anywhere when the literal misses turns an anchored path into a
// match-anywhere expression: e.g. path "name" against
// {"id":"123","details":{"name":"x"}} would strip "details.name" instead of
// no-oping. Since INPUT_ONLY fields are always omitted by the server, the
// literal miss is the normal case and the fallback would fire on every
// Strip call; the failure mode is silent over-stripping. cligen emits
// paths anchored to specific schema locations and does not currently emit
// paths that descend through proto maps (cli.json carries one ref slot,
// populated for singleton message fields only). If that contract grows
// map value refs later, the path language should grow an explicit map
// marker (e.g. "*") rather than reintroducing implicit fallback.
func deletePath(v any, keys []string) {
	if len(keys) == 0 {
		return
	}
	switch t := v.(type) {
	case map[string]any:
		child, ok := t[keys[0]]
		if !ok {
			return
		}
		if len(keys) == 1 {
			delete(t, keys[0])
			return
		}
		deletePath(child, keys[1:])
	case []any:
		for _, el := range t {
			deletePath(el, keys)
		}
	}
}
