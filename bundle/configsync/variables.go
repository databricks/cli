package configsync

import (
	"context"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
)

// varPrefix is the dyn.Path prefix for the ${var.X} shorthand.
var varPrefix = dyn.NewPath(dyn.Key("var"))

// RestoreVariableReferences replaces hardcoded change values with variable
// references (${var.foo}, ${bundle.target}, ${resources.X.Y.id}) when the
// value can be traced back to a reference in the original YAML. Resource IDs
// are injected from state since they aren't materialized into the resolved
// config's dyn.Value tree.
//
// For Replace operations, restoration requires the pre-resolved YAML at the
// exact field position. Pure variable references (e.g., ${var.catalog}) are
// restored when their resolved value matches. Compound interpolation strings
// (e.g., "/mnt/${var.account}/raw/landing") are reconstructed by preserving
// variables whose resolved values still appear at their expected positions
// and updating only the literal segments.
//
// For Add operations, restoration is limited to new sequence elements (e.g.,
// a new task appended to the tasks array). Within the new element, a leaf is
// restored only when a sibling element in the same sequence has a pure
// variable reference at the exact same relative path whose resolved value
// matches the leaf value. Non-sequence Adds (new map fields) are left
// untouched.
//
// The pre-resolved config is obtained by re-loading the bundle from disk
// through the standard loader mutators (entry point + includes + target
// overrides) but skipping variable resolution. This gives a fully merged
// view where ${var.X} and ${resources.X.Y.id} references are still literal
// strings — enabling correct sibling lookup even for sequences split across
// files via target overrides.
func RestoreVariableReferences(ctx context.Context, b *bundle.Bundle, fieldChanges []FieldChange) {
	preResolved := loadPreResolvedConfig(ctx, b)
	if !preResolved.IsValid() {
		return
	}
	resolved := b.Config.Value()

	// Augment resolved with resource IDs from state — only when the config
	// actually uses ${resources.X.Y.id} references. The IDs aren't materialized
	// into b.Config.Value() (they live in the StateDB), so we inject them here
	// to enable sibling-based restoration. Skipped entirely for bundles with
	// no resource refs to avoid opening state DB files unnecessarily.
	resourceRefs := collectResourceIDRefs(preResolved)
	if len(resourceRefs) > 0 {
		if lookup := resourceIDLookup(ctx, b); lookup != nil {
			resolved = injectResourceIDs(ctx, resolved, resourceRefs, lookup)
		} else {
			log.Debugf(ctx, "variable restoration: state DB unavailable, skipping resource ID injection for %d refs", len(resourceRefs))
		}
	}

	for i := range fieldChanges {
		fc := &fieldChanges[i]

		var newValue any
		switch fc.Change.Operation {
		case OperationReplace:
			fieldValue, ok := preResolvedValueAt(preResolved, fc.FieldCandidates)
			if !ok {
				continue
			}
			newValue = restoreOriginalRefs(fc.Change.Value, fieldValue, resolved)
		case OperationAdd:
			siblings, ok := sequenceSiblings(preResolved, fc.FieldCandidates)
			if !ok {
				continue
			}
			newValue = restoreFromSiblings(fc.Change.Value, siblings, resolved)
		case OperationUnknown, OperationRemove, OperationSkip:
			continue
		}

		fc.Change = &ConfigChangeDesc{
			Operation: fc.Change.Operation,
			Value:     newValue,
		}
	}
}

// loadPreResolvedConfig loads the bundle's configuration through the standard
// loader mutators (entry point, includes, target overrides) but without
// variable resolution. The resulting dyn.Value is fully merged across files
// and targets, yet retains ${...} references as literal strings. Returns
// InvalidValue if loading fails (restoration is then skipped).
func loadPreResolvedConfig(ctx context.Context, b *bundle.Bundle) dyn.Value {
	fresh := &bundle.Bundle{
		BundleRootPath: b.BundleRootPath,
		BundleRoot:     b.BundleRoot,
	}
	mutator.DefaultMutators(ctx, fresh)
	if target := b.Config.Bundle.Target; target != "" {
		if _, ok := fresh.Config.Targets[target]; ok {
			bundle.ApplyContext(ctx, fresh, mutator.SelectTarget(target))
		}
	}
	return fresh.Config.Value()
}

// resourceIDLookup returns a function that resolves resource keys to their
// deployed IDs from state. For the direct engine, the StateDB is already open
// on b.DeploymentBundle. For the terraform engine, the config snapshot is
// opened locally (it was downloaded by ensureSnapshotAvailable during
// DetectChanges). Returns nil if no state is available.
func resourceIDLookup(ctx context.Context, b *bundle.Bundle) func(string) string {
	if b.DeploymentBundle.StateDB.Path != "" {
		return b.DeploymentBundle.StateDB.GetResourceID
	}
	_, statePath := b.StateFilenameConfigSnapshot(ctx)
	db := &dstate.DeploymentState{}
	if err := db.Open(statePath); err != nil {
		log.Debugf(ctx, "variable restoration: failed to open state DB at %s: %v", statePath, err)
		return nil
	}
	return db.GetResourceID
}

// collectResourceIDRefs walks the pre-resolved merged config to find pure
// ${resources.<kind>.<name>.id} references. Returns the unique set of paths
// so the caller can inject IDs at those positions; returns nil if no such
// references exist.
func collectResourceIDRefs(preResolved dyn.Value) []dyn.Path {
	seen := map[string]bool{}
	var paths []dyn.Path
	_ = dyn.WalkReadOnly(preResolved, func(_ dyn.Path, v dyn.Value) error {
		s, ok := v.AsString()
		if !ok || !dynvar.IsPureVariableReference(s) || seen[s] {
			return nil
		}
		seen[s] = true
		p, ok := dynvar.PureReferenceToPath(s)
		if !ok {
			return nil
		}
		if len(p) != 4 || p[0].Key() != "resources" || p[3].Key() != "id" {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	return paths
}

// injectResourceIDs populates the resolved dyn.Value with IDs from state for
// the given resource reference paths. Skips references whose IDs aren't in
// state or that can't be written back into the dyn.Value tree.
func injectResourceIDs(ctx context.Context, resolved dyn.Value, paths []dyn.Path, lookupID func(string) string) dyn.Value {
	for _, p := range paths {
		resourceKey := p[:3].String()
		id := lookupID(resourceKey)
		if id == "" {
			log.Debugf(ctx, "variable restoration: no state entry for resource %q", resourceKey)
			continue
		}
		updated, err := dyn.SetByPath(resolved, p, dyn.V(id))
		if err != nil {
			log.Debugf(ctx, "variable restoration: SetByPath failed for %s: %v", p, err)
			continue
		}
		resolved = updated
	}
	return resolved
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

// restoreFromSiblings recursively restores variable references for new
// sequence elements. For each leaf, it consults sibling elements at the same
// relative path: if exactly one unique pure variable reference across siblings
// resolves to the leaf value, that reference is substituted. Multiple
// different matching references are treated as ambiguous and skipped.
func restoreFromSiblings(value any, siblings []dyn.Value, resolved dyn.Value) any {
	return restoreFromSiblingsAt(value, siblings, resolved, dyn.EmptyPath)
}

func restoreFromSiblingsAt(value any, siblings []dyn.Value, resolved dyn.Value, relPath dyn.Path) any {
	switch v := value.(type) {
	case string, bool, int64:
		refs := map[string]struct{}{}
		strVal, isStr := value.(string)
		for _, sib := range siblings {
			sv, err := dyn.GetByPath(sib, relPath)
			if err != nil {
				continue
			}
			s, ok := sv.AsString()
			if !ok {
				continue
			}
			if dynvar.IsPureVariableReference(s) {
				rp, ok := resolveReferencePath(s)
				if !ok {
					continue
				}
				rv, getErr := dyn.GetByPath(resolved, rp)
				if getErr != nil {
					continue
				}
				if rv.AsAny() == value {
					refs[s] = struct{}{}
				}
			} else if isStr && dynvar.ContainsVariableReference(s) {
				// Compound interpolation in sibling: try to align the new
				// value against the sibling's template. If all variables
				// match at their positions, the template (possibly with
				// updated literal segments) is used.
				if restored, ok := restoreCompoundInterpolation(strVal, sv, resolved); ok {
					refs[restored] = struct{}{}
				}
			}
		}
		if len(refs) == 1 {
			for ref := range refs {
				return ref
			}
		}
		return value

	case map[string]any:
		for key, val := range v {
			v[key] = restoreFromSiblingsAt(val, siblings, resolved, relPath.Append(dyn.Key(key)))
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = restoreFromSiblingsAt(val, siblings, resolved, relPath.Append(dyn.Index(i)))
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
		return template, true
	}
	if !dynvar.ContainsVariableReference(restored) {
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

	var segments []templateSegment
	cursor := 0

	for _, m := range ref.Matches {
		fullMatch := m[0]

		idx := strings.Index(template[cursor:], fullMatch)
		if idx < 0 {
			return nil
		}

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

	if cursor < len(template) {
		segments = append(segments, templateSegment{
			raw: template[cursor:],
		})
	}

	return segments
}

// findAnchorOffset searches for the next literal segment's text in remaining
// and returns the offset where it starts. Returns len(remaining) if no
// subsequent literal exists. Returns -1 if a subsequent literal exists but
// can't be found.
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

// preResolvedValueAt returns the pre-resolved dyn.Value at the field path,
// if the field exists in the merged pre-resolved config.
func preResolvedValueAt(preResolved dyn.Value, candidates []string) (dyn.Value, bool) {
	for _, candidate := range candidates {
		candidate = stripBracketStars(candidate)
		p, err := dyn.NewPathFromString(candidate)
		if err != nil {
			continue
		}
		v, err := dyn.GetByPath(preResolved, p)
		if err == nil {
			return v, true
		}
	}
	return dyn.InvalidValue, false
}

// sequenceSiblings returns the sibling elements of the parent sequence when
// the field change represents adding a new element to a sequence. The path's
// last component must be an index ([*] or [N]) and the parent must resolve
// to a sequence in the pre-resolved config. Returns false for non-sequence
// Adds (e.g., new map fields).
func sequenceSiblings(preResolved dyn.Value, candidates []string) ([]dyn.Value, bool) {
	for _, candidate := range candidates {
		parent, ok := extractSequenceParent(candidate)
		if !ok {
			continue
		}
		p, err := dyn.NewPathFromString(parent)
		if err != nil {
			continue
		}
		parentValue, err := dyn.GetByPath(preResolved, p)
		if err != nil {
			continue
		}
		if parentValue.Kind() != dyn.KindSequence {
			continue
		}
		seq, ok := parentValue.AsSequence()
		if !ok {
			continue
		}
		return seq, true
	}
	return nil, false
}

// extractSequenceParent returns the parent path if the candidate ends in an
// index (either [*] or [N]).
func extractSequenceParent(candidate string) (string, bool) {
	if before, ok := strings.CutSuffix(candidate, "[*]"); ok {
		return before, true
	}
	if !strings.HasSuffix(candidate, "]") {
		return "", false
	}
	idx := strings.LastIndex(candidate, "[")
	if idx < 0 {
		return "", false
	}
	inner := candidate[idx+1 : len(candidate)-1]
	if _, err := strconv.Atoi(inner); err != nil {
		return "", false
	}
	return candidate[:idx], true
}

// stripBracketStars removes all [*] segments from a structpath string.
// resolveSelectors inserts [*] at any array position for Add operations
// where the target element doesn't exist yet.
func stripBracketStars(candidate string) string {
	return strings.ReplaceAll(candidate, "[*]", "")
}
