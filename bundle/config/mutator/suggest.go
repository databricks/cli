package mutator

import (
	"math"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/textutil"
)

// suggestionThreshold returns the maximum edit distance for a given key length.
func suggestionThreshold(keyLen int) int {
	return min(3, max(1, keyLen/2))
}

// closestMatch finds the candidate with the smallest Levenshtein distance to key,
// within the suggestion threshold. Returns ("", math.MaxInt) if no match is close enough.
func closestMatch(key string, candidates []string) (string, int) {
	best := ""
	bestDist := math.MaxInt
	threshold := suggestionThreshold(len(key))
	for _, c := range candidates {
		d := textutil.LevenshteinDistance(key, c)
		if d < bestDist {
			bestDist = d
			best = c
			if d == 0 {
				break
			}
		}
	}
	if bestDist > threshold {
		return "", math.MaxInt
	}
	return best, bestDist
}

// structFieldNames enumerates all valid JSON field names for a struct type,
// including embedded/flattened fields and bundle:"readonly" fields.
// It excludes bundle:"internal", json:"-", unexported, and EmbeddedSlice fields.
func structFieldNames(t reflect.Type) []string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}

	var names []string
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue // unexported
		}
		if sf.Name == structaccess.EmbeddedSliceFieldName {
			continue
		}

		// For anonymous (embedded) structs without json tags, flatten their fields.
		jsonTag := sf.Tag.Get("json")
		if sf.Anonymous && jsonTag == "" {
			ft := sf.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			names = append(names, structFieldNames(ft)...)
			continue
		}

		name := structtag.JSONTag(jsonTag).Name()
		if name == "-" || name == "" {
			continue
		}

		// Exclude internal fields but include readonly fields (like resource IDs).
		btag := structtag.BundleTag(sf.Tag.Get("bundle"))
		if btag.Internal() {
			continue
		}

		names = append(names, name)
	}
	return names
}

// mapKeysFromDyn extracts string keys from a dyn.Value map.
func mapKeysFromDyn(v dyn.Value) []string {
	m, ok := v.AsMap()
	if !ok {
		return nil
	}
	var keys []string
	for _, k := range m.Keys() {
		if s, ok := k.AsString(); ok {
			keys = append(keys, s)
		}
	}
	return keys
}

// rewriteToVarShorthand converts "variables.X.value" back to "var.X".
func rewriteToVarShorthand(key string) string {
	if strings.HasPrefix(key, "variables.") && strings.HasSuffix(key, ".value") {
		middle := key[len("variables."):]
		middle = middle[:len(middle)-len(".value")]
		if !strings.Contains(middle, ".") {
			return "var." + middle
		}
	}
	return key
}

// makeSuggestFn builds a SuggestFn that uses Go type information and runtime
// dyn values to generate suggestions for unresolved variable references.
//
// The algorithm walks the entire path segment by segment, greedily correcting
// each wrong segment. This means multiple segments can be corrected in a single
// suggestion (e.g., ${bundl.nme} → ${bundle.name}).
//
// At each segment:
//   - Struct types: candidates come from the Go type via reflection (works even
//     for fields not in the config tree, like resource IDs).
//   - Map types: candidates come from the actual runtime dyn.Value keys.
func (m *resolveVariableReferences) makeSuggestFn(
	normalized dyn.Value,
) dynvar.SuggestFn {
	return func(key string) string {
		return m.suggest(key, normalized)
	}
}

func (m *resolveVariableReferences) suggest(
	key string,
	normalized dyn.Value,
) string {
	// Parse the key into path segments.
	path, err := dyn.NewPathFromString(key)
	if err != nil || len(path) == 0 {
		return ""
	}

	// Handle var.X → variables.X.value rewriting for internal lookup.
	// Also detect typos in the "var" prefix itself (e.g., "vr", "va").
	// Require at least 2 segments (e.g., "var.X") to avoid a panic on bare "var".
	isVar := len(path) >= 2 && path.HasPrefix(varPath)
	varPrefixCorrected := false
	if !isVar && len(path) >= 2 {
		if c, _ := closestMatch(path[0].Key(), []string{"var"}); c != "" {
			isVar = true
			varPrefixCorrected = true
		}
	}

	if isVar {
		newPath := dyn.NewPath(dyn.Key("variables"), path[1], dyn.Key("value"))
		if len(path) > 2 {
			newPath = newPath.Append(path[2:]...)
		}
		path = newPath
	}

	// Extract segment strings from the path. Only support simple key segments.
	segments := make([]string, len(path))
	for i, c := range path {
		segments[i] = c.Key()
	}

	suggestion := suggestPath(segments, normalized, varPrefixCorrected)
	if suggestion == "" {
		return ""
	}

	if isVar {
		suggestion = rewriteToVarShorthand(suggestion)
	}
	return suggestion
}

// suggestPath walks the config type tree and dyn value tree segment by segment,
// greedily correcting each wrong segment. Returns the corrected dot-separated
// path if any corrections were made, or "" if no suggestion can be produced.
func suggestPath(segments []string, normalized dyn.Value, initialHasFix bool) string {
	currentType := reflect.TypeOf(config.Root{})
	currentDyn := normalized
	corrected := make([]string, len(segments))
	hasFix := initialHasFix

	for i, seg := range segments {
		// Dereference pointer types.
		for currentType != nil && currentType.Kind() == reflect.Pointer {
			currentType = currentType.Elem()
		}

		if currentType == nil {
			return ""
		}

		var candidates []string
		var nextType reflect.Type

		switch currentType.Kind() {
		case reflect.Struct:
			candidates = structFieldNames(currentType)

			if slices.Contains(candidates, seg) {
				corrected[i] = seg
				nextType = fieldTypeByName(currentType, seg)
			}

		case reflect.Map:
			if currentType.Key().Kind() != reflect.String {
				return ""
			}
			candidates = mapKeysFromDyn(currentDyn)
			nextType = currentType.Elem()

			// Check if the key exists in the map.
			if currentDyn.IsValid() {
				if _, err := dyn.GetByPath(currentDyn, dyn.NewPath(dyn.Key(seg))); err == nil {
					corrected[i] = seg
				}
			}

		default:
			return ""
		}

		// If not matched, try fuzzy correction.
		if corrected[i] == "" {
			best, _ := closestMatch(seg, candidates)
			if best == "" {
				return ""
			}
			corrected[i] = best
			hasFix = true

			// Advance type using the corrected segment.
			if currentType.Kind() == reflect.Struct {
				nextType = fieldTypeByName(currentType, best)
			}
			// For maps, nextType is already set to the map element type.
		}

		// Advance dyn value for the next segment.
		if currentDyn.IsValid() {
			next, err := dyn.GetByPath(currentDyn, dyn.NewPath(dyn.Key(corrected[i])))
			if err == nil {
				currentDyn = next
			} else {
				currentDyn = dyn.InvalidValue
			}
		}

		currentType = nextType
	}

	if !hasFix {
		return ""
	}

	return strings.Join(corrected, ".")
}

// fieldTypeByName returns the reflect.Type of a struct field identified by its
// JSON name, searching embedded structs.
func fieldTypeByName(t reflect.Type, name string) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Anonymous && sf.Tag.Get("json") == "" {
			ft := sf.Type
			for ft.Kind() == reflect.Pointer {
				ft = ft.Elem()
			}
			if result := fieldTypeByName(ft, name); result != nil {
				return result
			}
			continue
		}
		jsonName := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if jsonName == name {
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() {
				continue
			}
			return sf.Type
		}
	}
	return nil
}
