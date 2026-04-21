package configsync

import (
	"bytes"
	"os"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

// varPrefix is the dyn.Path prefix for the ${var.X} shorthand.
var varPrefix = dyn.NewPath(dyn.Key("var"))

// RestoreVariableReferences replaces hardcoded change values with variable
// references (e.g., ${var.foo}) when the value can be traced back to a
// variable in the original YAML.
//
// For Replace operations, restoration only happens when the pre-resolved YAML
// at the exact field position was a pure variable reference whose resolved
// value matches the incoming remote value. The reverse map is NOT used for
// Replace — this prevents false positives where a sibling's variable context
// would incorrectly rewrite an unrelated hardcoded field.
//
// For Add operations (new fields), the reverse map is used: if the parent
// subtree uses variables and the new value matches exactly one variable, the
// variable reference is substituted.
func RestoreVariableReferences(b *bundle.Bundle, fieldChanges []FieldChange) {
	fileCache := map[string]dyn.Value{}
	resolved := b.Config.Value()

	reverseMap := buildVariableReverseMap(resolved, fileCache, fieldChanges)
	if len(reverseMap) == 0 {
		return
	}

	for i := range fieldChanges {
		fc := &fieldChanges[i]
		if fc.Change.Operation != OperationReplace && fc.Change.Operation != OperationAdd {
			continue
		}

		preResolved, hasContext := fieldVariableContext(fileCache, fc.FilePath, fc.FieldCandidates)

		var newValue any
		if fc.Change.Operation == OperationReplace {
			// Replace: only restore if the original field itself was a variable.
			if !preResolved.IsValid() {
				continue
			}
			newValue = restoreOriginalRefs(fc.Change.Value, preResolved, resolved)
		} else {
			// Add: use reverse map when parent context has variables.
			if !hasContext {
				continue
			}
			newValue = restoreFromReverseMap(fc.Change.Value, reverseMap)
		}

		fc.Change = &ConfigChangeDesc{
			Operation: fc.Change.Operation,
			Value:     newValue,
		}
	}
}

// buildVariableReverseMap discovers all pure variable references in the
// pre-resolved YAML files and pairs each with its resolved value from the
// bundle config. Returns a map from resolved value → reference strings.
func buildVariableReverseMap(resolved dyn.Value, fileCache map[string]dyn.Value, fieldChanges []FieldChange) map[any][]string {
	m := map[any][]string{}
	seen := map[string]bool{}

	files := map[string]bool{}
	for _, fc := range fieldChanges {
		files[fc.FilePath] = true
	}

	for filePath := range files {
		preResolved := loadCachedYAML(fileCache, filePath)
		if !preResolved.IsValid() {
			continue
		}
		collectReferences(preResolved, resolved, m, seen)
	}

	return m
}

// collectReferences walks a pre-resolved dyn.Value to find pure variable
// references (e.g., ${var.foo}, ${bundle.name}) and adds them to the reverse
// map keyed by their resolved value.
func collectReferences(preResolved, resolved dyn.Value, m map[any][]string, seen map[string]bool) {
	dyn.WalkReadOnly(preResolved, func(_ dyn.Path, v dyn.Value) error { //nolint:errcheck
		if v.Kind() != dyn.KindString {
			return nil
		}
		s := v.MustString()
		if !dynvar.IsPureVariableReference(s) || seen[s] {
			return nil
		}
		seen[s] = true

		resolvedPath, ok := resolveReferencePath(s)
		if !ok {
			return nil
		}

		resolvedV, err := dyn.GetByPath(resolved, resolvedPath)
		if err != nil {
			return nil
		}

		switch resolvedV.Kind() {
		case dyn.KindString, dyn.KindBool, dyn.KindInt:
			m[resolvedV.AsAny()] = append(m[resolvedV.AsAny()], s)
		case dyn.KindInvalid, dyn.KindMap, dyn.KindSequence, dyn.KindFloat, dyn.KindTime, dyn.KindNil:
			// Skip non-scalar and non-comparable types.
		}

		return nil
	})
}

// resolveReferencePath converts a variable reference string to the dyn.Path
// where its resolved value can be found in the bundle config. It applies the
// same ${var.X} → variables.X.value shorthand rewriting as the variable
// resolution mutator.
func resolveReferencePath(refStr string) (dyn.Path, bool) {
	p, ok := dynvar.PureReferenceToPath(refStr)
	if !ok {
		return nil, false
	}

	if p.HasPrefix(varPrefix) && len(p) >= 2 {
		newPath := dyn.NewPath(
			dyn.Key("variables"),
			p[1],
			dyn.Key("value"),
		)
		if len(p) > 2 {
			newPath = newPath.Append(p[2:]...)
		}
		return newPath, true
	}

	return p, true
}

// restoreOriginalRefs recursively restores variable references for Replace
// operations. Only restores a leaf when the pre-resolved config at the same
// position was a pure variable reference whose resolved value matches.
func restoreOriginalRefs(value any, preResolved, resolved dyn.Value) any {
	switch v := value.(type) {
	case string, bool, int64:
		if ref, ok := matchOriginalRef(value, preResolved, resolved); ok {
			return ref
		}
		return value

	case map[string]any:
		preMap, _ := preResolved.AsMap()
		for key, val := range v {
			var childPre dyn.Value
			if preMap.Len() > 0 {
				if p, ok := preMap.GetPairByString(key); ok {
					childPre = p.Value
				}
			}
			v[key] = restoreOriginalRefs(val, childPre, resolved)
		}
		return v

	case []any:
		preSeq, _ := preResolved.AsSequence()
		for i, val := range v {
			var childPre dyn.Value
			if i < len(preSeq) {
				childPre = preSeq[i]
			}
			v[i] = restoreOriginalRefs(val, childPre, resolved)
		}
		return v

	default:
		return value
	}
}

// restoreFromReverseMap recursively replaces leaf values with variable
// references for Add operations. Requires exactly one matching variable.
func restoreFromReverseMap(value any, reverseMap map[any][]string) any {
	switch v := value.(type) {
	case string, bool, int64:
		if refs := reverseMap[value]; len(refs) == 1 {
			return refs[0]
		}
		return value

	case map[string]any:
		for key, val := range v {
			v[key] = restoreFromReverseMap(val, reverseMap)
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = restoreFromReverseMap(val, reverseMap)
		}
		return v

	default:
		return value
	}
}

// matchOriginalRef checks if the pre-resolved config value at this position
// was a pure variable reference whose resolved value equals remoteValue.
func matchOriginalRef(remoteValue any, preResolved, resolved dyn.Value) (string, bool) {
	if !preResolved.IsValid() {
		return "", false
	}
	s, ok := preResolved.AsString()
	if !ok || !dynvar.IsPureVariableReference(s) {
		return "", false
	}

	resolvedPath, ok := resolveReferencePath(s)
	if !ok {
		return "", false
	}

	resolvedV, err := dyn.GetByPath(resolved, resolvedPath)
	if err != nil {
		return "", false
	}

	if resolvedV.AsAny() == remoteValue {
		return s, true
	}
	return "", false
}

// fieldVariableContext returns the pre-resolved dyn.Value at the field path
// and whether the field's parent subtree contains any variable reference.
// The returned dyn.Value is valid only when the field itself was found in the
// pre-resolved YAML (used for Replace). The bool is true when any ancestor
// uses variables (used for Add).
func fieldVariableContext(cache map[string]dyn.Value, filePath string, candidates []string) (dyn.Value, bool) {
	configValue := loadCachedYAML(cache, filePath)
	if !configValue.IsValid() {
		return dyn.InvalidValue, false
	}

	for _, candidate := range candidates {
		candidate = stripBracketStars(candidate)

		p, err := dyn.NewPathFromString(candidate)
		if err != nil {
			continue
		}

		v, err := dyn.GetByPath(configValue, p)
		if err == nil {
			if subtreeHasVariableRef(v) {
				return v, true
			}
		}

		if len(p) > 0 {
			parent, err := dyn.GetByPath(configValue, p[:len(p)-1])
			if err == nil && subtreeHasVariableRef(parent) {
				return dyn.InvalidValue, true
			}
		}
	}

	return dyn.InvalidValue, false
}

// stripBracketStars removes all [*] segments from a structpath string.
// resolveSelectors inserts [*] at any array position for Add operations
// where the target element doesn't exist yet.
func stripBracketStars(candidate string) string {
	return strings.ReplaceAll(candidate, "[*]", "")
}

// loadCachedYAML parses a YAML file and caches the result. Returns the
// pre-resolved dyn.Value (variable references are still literal strings).
func loadCachedYAML(cache map[string]dyn.Value, filePath string) dyn.Value {
	if v, ok := cache[filePath]; ok {
		return v
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		cache[filePath] = dyn.InvalidValue
		return dyn.InvalidValue
	}

	v, err := yamlloader.LoadYAML(filePath, bytes.NewBuffer(raw))
	if err != nil {
		cache[filePath] = dyn.InvalidValue
		return dyn.InvalidValue
	}

	cache[filePath] = v
	return v
}

// subtreeHasVariableRef recursively checks whether any string leaf in the
// dyn.Value subtree contains a variable reference. Short-circuits on first find.
func subtreeHasVariableRef(v dyn.Value) bool {
	switch v.Kind() {
	case dyn.KindString:
		return dynvar.ContainsVariableReference(v.MustString())
	case dyn.KindMap:
		m, _ := v.AsMap()
		for _, p := range m.Pairs() {
			if subtreeHasVariableRef(p.Value) {
				return true
			}
		}
	case dyn.KindSequence:
		s, _ := v.AsSequence()
		for _, elem := range s {
			if subtreeHasVariableRef(elem) {
				return true
			}
		}
	case dyn.KindInvalid, dyn.KindBool, dyn.KindInt, dyn.KindFloat, dyn.KindTime, dyn.KindNil:
		// Leaf types that cannot contain variable references.
	}
	return false
}
