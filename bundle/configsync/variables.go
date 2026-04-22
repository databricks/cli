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
// For Replace operations, restoration requires the pre-resolved YAML at the
// exact field position. Pure variable references (e.g., ${var.catalog}) are
// restored when their resolved value matches. Compound interpolation strings
// (e.g., "/mnt/${var.account}/raw/landing") are reconstructed by preserving
// variables whose resolved values still appear at their expected positions
// and updating only the literal segments. The reverse map is NOT used for
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
	// The callback never returns an error, so WalkReadOnly always returns nil.
	_ = dyn.WalkReadOnly(preResolved, func(_ dyn.Path, v dyn.Value) error {
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

		resolvedV, lookupErr := dyn.GetByPath(resolved, resolvedPath)
		if lookupErr != nil {
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
// operations. For pure variable references, restores when the resolved value
// matches. For compound interpolation (e.g., "${var.X}_suffix"), preserves
// variables whose resolved values still appear at their expected positions.
func restoreOriginalRefs(value any, preResolved, resolved dyn.Value) any {
	switch v := value.(type) {
	case string, bool, int64:
		if ref, ok := matchOriginalRef(value, preResolved, resolved); ok {
			return ref
		}
		if s, ok := value.(string); ok {
			if restored, ok := restoreCompoundInterpolation(s, preResolved, resolved); ok {
				return restored
			}
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

// restoreCompoundInterpolation handles strings with mixed variable references
// and literal text, e.g., "/mnt/${var.account}/raw/landing". It checks whether
// each variable's resolved value still appears at its expected position in the
// remote string. Variables that match are preserved; changed literal segments
// are updated. Falls back to false if the template can't be aligned.
//
// Known limitation: variable matching is prefix-greedy. If ${var.X}="foo" and
// the remote starts with "footbar...", HasPrefix matches "foo" and the leftover
// "tbar" becomes literal garbage. Adjacent variables without a literal separator
// (e.g., "${var.A}${var.B}") cannot be reliably split if either value changes.
func restoreCompoundInterpolation(remoteValue string, preResolved, resolved dyn.Value) (string, bool) {
	if !preResolved.IsValid() {
		return "", false
	}
	template, ok := preResolved.AsString()
	if !ok || !dynvar.ContainsVariableReference(template) || dynvar.IsPureVariableReference(template) {
		return "", false
	}

	// Parse the template into segments: alternating literal and variable parts.
	segments := parseTemplateSegments(template, resolved)
	if segments == nil {
		return "", false
	}

	// Walk the remote string, aligning each segment against the template.
	// For each segment we try an exact match first. On mismatch we search
	// ahead for the next literal anchor to determine the boundary.
	// The last segment always consumes all remaining text.
	var result strings.Builder
	pos := 0

	for i, seg := range segments {
		if pos > len(remoteValue) {
			return "", false
		}

		remaining := remoteValue[pos:]
		isLast := i == len(segments)-1

		if seg.isVariable {
			if seg.resolvedValue == "" {
				return "", false
			}
			if isLast {
				// Last segment: variable must match the entire remainder.
				if remaining == seg.resolvedValue {
					result.WriteString(seg.raw)
				} else {
					result.WriteString(remaining)
				}
				pos = len(remoteValue)
			} else if strings.HasPrefix(remaining, seg.resolvedValue) {
				result.WriteString(seg.raw)
				pos += len(seg.resolvedValue)
			} else {
				end := findAnchorOffset(segments, i+1, remaining)
				if end < 0 {
					return "", false
				}
				result.WriteString(remaining[:end])
				pos += end
			}
		} else {
			if isLast {
				// Last literal: take the entire remainder (may include suffix changes).
				result.WriteString(remaining)
				pos = len(remoteValue)
			} else if strings.HasPrefix(remaining, seg.raw) {
				result.WriteString(seg.raw)
				pos += len(seg.raw)
			} else {
				end := findAnchorOffset(segments, i+1, remaining)
				if end < 0 {
					return "", false
				}
				result.WriteString(remaining[:end])
				pos += end
			}
		}
	}

	restored := result.String()
	if restored == template {
		// Nothing changed — keep the original template as-is.
		return template, true
	}
	if !dynvar.ContainsVariableReference(restored) {
		// All variables were lost — no benefit over hardcoding.
		return "", false
	}
	return restored, true
}

// templateSegment represents either a literal string or a variable reference
// within a template string.
type templateSegment struct {
	raw           string // as it appears in the template (literal text or "${var.X}")
	isVariable    bool
	resolvedValue string // only set for variable segments
}

// parseTemplateSegments splits a template string like "/mnt/${var.X}/raw"
// into alternating literal and variable segments, resolving each variable.
// Returns nil if any variable can't be resolved.
func parseTemplateSegments(template string, resolved dyn.Value) []templateSegment {
	ref, ok := dynvar.NewRef(dyn.V(template))
	if !ok {
		return nil
	}

	// Build full match strings: each Matches[i][0] is the "${...}" text.
	var segments []templateSegment
	cursor := 0

	for _, m := range ref.Matches {
		fullMatch := m[0] // e.g., "${var.catalog}"

		// Find this match in the template starting from cursor.
		idx := strings.Index(template[cursor:], fullMatch)
		if idx < 0 {
			return nil
		}

		// Literal before this variable.
		if idx > 0 {
			segments = append(segments, templateSegment{
				raw: template[cursor : cursor+idx],
			})
		}

		resolvedPath, ok := resolveReferencePath(fullMatch)
		if !ok {
			return nil
		}

		resolvedV, err := dyn.GetByPath(resolved, resolvedPath)
		if err != nil {
			return nil
		}

		resolvedStr, ok := resolvedV.AsString()
		if !ok {
			return nil
		}

		segments = append(segments, templateSegment{
			raw:           fullMatch,
			isVariable:    true,
			resolvedValue: resolvedStr,
		})

		cursor += idx + len(fullMatch)
	}

	// Trailing literal.
	if cursor < len(template) {
		segments = append(segments, templateSegment{
			raw: template[cursor:],
		})
	}

	return segments
}

// findAnchorOffset searches for the next literal segment's text in remaining
// and returns the offset where it starts. This is used to determine boundaries
// when a variable or literal segment doesn't match at the current position.
// Returns len(remaining) if no subsequent literal exists (last segment case).
// Returns -1 if a subsequent literal exists but can't be found.
func findAnchorOffset(segments []templateSegment, from int, remaining string) int {
	for i := from; i < len(segments); i++ {
		if !segments[i].isVariable {
			idx := strings.Index(remaining, segments[i].raw)
			if idx < 0 {
				return -1
			}
			return idx
		}
	}
	return len(remaining)
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
