package serving_endpoints

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

// maxDroppedFields caps how many dropped fields the warning lists.
const maxDroppedFields = 10

// warnDroppedFields prints a warning on stderr when the typed response that
// query prints is missing fields of the raw response.
func warnDroppedFields(ctx context.Context, name string, raw []byte, typed any) error {
	dropped := droppedFields(raw, typed)
	if len(dropped) == 0 {
		return nil
	}
	list := strings.Join(dropped[:min(len(dropped), maxDroppedFields)], ", ")
	if len(dropped) > maxDroppedFields {
		list += fmt.Sprintf(", and %d more", len(dropped)-maxDroppedFields)
	}
	return cmdio.RenderDiagnostics(ctx, diag.Diagnostics{{
		Severity: diag.Warning,
		Summary:  "the endpoint's response has fields that this output doesn't show: " + list,
		Detail:   fmt.Sprintf("To see the full response, send the same request with:\n  databricks api post /serving-endpoints/%s/invocations --json ...", name),
	}})
}

// droppedFields lists the paths (like "choices[].finish_reason") that are in
// raw but are missing or have a different JSON type in typed once it's
// marshaled back to JSON. Null and zero values are ignored, because typed
// fields with omitempty leave them out. It returns nil if either side can't be
// decoded: the response already decoded once, and a failed comparison must
// not fail the query.
func droppedFields(raw []byte, typed any) []string {
	rawValue, err := decodeJSON(raw)
	if err != nil {
		return nil
	}
	typedJSON, err := json.Marshal(typed)
	if err != nil {
		return nil
	}
	typedValue, err := decodeJSON(typedJSON)
	if err != nil {
		return nil
	}
	var paths []string
	collectDropped(rawValue, typedValue, "", &paths)
	slices.Sort(paths)
	return slices.Compact(paths)
}

func decodeJSON(data []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	var v any
	err := dec.Decode(&v)
	return v, err
}

func collectDropped(raw, typed any, path string, paths *[]string) {
	switch r := raw.(type) {
	case map[string]any:
		t, ok := typed.(map[string]any)
		if !ok {
			*paths = append(*paths, path)
			return
		}
		for k, rv := range r {
			child := joinFieldPath(path, k)
			tv, ok := t[k]
			if !ok {
				if !isZeroJSON(rv) {
					*paths = append(*paths, child)
				}
				continue
			}
			collectDropped(rv, tv, child, paths)
		}
	case []any:
		t, ok := typed.([]any)
		if !ok {
			*paths = append(*paths, path)
			return
		}
		for i, rv := range r {
			if i >= len(t) {
				*paths = append(*paths, path+"[]")
				return
			}
			collectDropped(rv, t[i], path+"[]", paths)
		}
	default:
		if raw != nil && typed != nil && jsonKind(raw) != jsonKind(typed) {
			*paths = append(*paths, path)
		}
	}
}

func joinFieldPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func jsonKind(v any) string {
	switch v.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	case string:
		return "string"
	case json.Number:
		return "number"
	case bool:
		return "bool"
	}
	return "null"
}

func isZeroJSON(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case string:
		return x == ""
	case bool:
		return !x
	case json.Number:
		f, err := x.Float64()
		return err == nil && f == 0
	case []any:
		return len(x) == 0
	case map[string]any:
		return len(x) == 0
	}
	return false
}
