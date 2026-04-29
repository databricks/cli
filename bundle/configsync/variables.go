package configsync

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
)

// varPrefix is the dyn.Path prefix for the ${var.X} shorthand.
var varPrefix = dyn.NewPath(dyn.Key("var"))

// RestoreVariableReferences replaces hardcoded change values with variable
// references (${var.foo}, ${bundle.target}, ${resources.X.Y.id}) when the
// value can be traced back to a reference in the original YAML. Resource IDs
// are injected from state since they aren't materialized into the resolved
// config's dyn.Value tree.
//
// For Replace operations, restoration consults the pre-resolved YAML at the
// exact field position and tries three steps in order:
//  1. If the YAML had a pure ref (${var.X}, ${bundle.X}, ${resources.X.Y.id})
//     and its resolved value equals the new value, the original ref is kept.
//  2. If the YAML had a compound string (e.g., "/mnt/${var.account}/raw/X"),
//     the template is realigned: variables whose resolved values still appear
//     at their expected positions are kept, and only literal segments change.
//  3. Fallback for fields whose YAML was a pure ${var.X} but whose resolved
//     value doesn't match: search all bundle variables for a unique scalar
//     match. On a unique match, the field is re-targeted to that variable
//     (e.g., ${var.schema} → ${var.dev_schema}). Multiple matches are
//     ambiguous and skipped. The fallback is gated on the YAML field already
//     being a pure ${var.X}, so hardcoded literals are never promoted.
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

	// Mirror mutator.lookup's source-linked deployment override: when enabled,
	// ${workspace.file_path} resolves to b.SyncRootPath rather than the typed
	// workspace.file_path field (which still holds the default deploy path).
	// Without this, substring matching against the typed value misses the
	// actual deployed path and variables are lost on Replace. Keep this in
	// sync with mutator.lookup if new overrides are added there.
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		fpPath := dyn.NewPath(dyn.Key("workspace"), dyn.Key("file_path"))
		if updated, err := dyn.SetByPath(resolved, fpPath, dyn.V(b.SyncRootPath)); err == nil {
			resolved = updated
		}
	}

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
// When the original is a pure ${var.X} but its resolved value doesn't match the
// new value, falls back to a global lookup: if the new value uniquely matches
// a different variable, that variable is used instead. The field's prior use
// of a variable is the false-positive guard.
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
		if isPureVarRef(preResolved) {
			if ref, ok := matchAnyVariable(value, resolved); ok {
				return ref
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

// isPureVarRef reports whether the pre-resolved value at the field is a pure
// ${var.X} reference. Used to gate the fallback substitution: only fields that
// already used a variable can be re-targeted to a different variable.
func isPureVarRef(preResolved dyn.Value) bool {
	if !preResolved.IsValid() {
		return false
	}
	s, ok := preResolved.AsString()
	if !ok || !dynvar.IsPureVariableReference(s) {
		return false
	}
	p, ok := dynvar.PureReferenceToPath(s)
	if !ok {
		return false
	}
	return p.HasPrefix(varPrefix)
}

// matchAnyVariable searches all bundle variables for a unique scalar value that
// equals remoteValue. Returns the ${var.X} reference on a unique match, ""
// otherwise. Multiple matches are treated as ambiguous and skipped.
func matchAnyVariable(remoteValue any, resolved dyn.Value) (string, bool) {
	variables, err := dyn.GetByPath(resolved, dyn.NewPath(dyn.Key("variables")))
	if err != nil {
		return "", false
	}
	vmap, ok := variables.AsMap()
	if !ok {
		return "", false
	}
	var match string
	count := 0
	for _, pair := range vmap.Pairs() {
		name, ok := pair.Key.AsString()
		if !ok {
			continue
		}
		v, getErr := dyn.GetByPath(pair.Value, dyn.NewPath(dyn.Key("value")))
		if getErr != nil {
			continue
		}
		switch v.Kind() {
		case dyn.KindString, dyn.KindInt, dyn.KindBool:
			if v.AsAny() == remoteValue {
				match = pathToRef(varPrefix.Append(dyn.Key(name)))
				count++
			}
		}
	}
	if count == 1 {
		return match, true
	}
	return "", false
}

// pathToRef formats a dyn.Path as a "${...}" interpolation reference.
func pathToRef(p dyn.Path) string {
	return "${" + p.String() + "}"
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
// and literal text, e.g., "/mnt/${var.account}/raw/landing".
//
// Algorithm: for each variable in the template, find the first occurrence of
// its resolved value in the remote string and substitute it back to its raw
// ${...} form. Variables whose resolved value no longer appears are dropped
// (the user changed them); literal segments can grow, shrink, or disappear
// freely. Returns false if no variable ends up in the result (e.g., the user
// replaced everything with an unrelated string).
//
// Known limitation: substring-matching is unanchored. If ${var.X}="in" and the
// new value contains "in" inside an unrelated word, that occurrence is still
// rewritten to ${var.X}. Variables in the template are processed in order of
// appearance, which is usually what the user expects.
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

	result := remoteValue
	for _, seg := range segments {
		if !seg.isVariable || seg.resolvedValue == "" {
			continue
		}
		idx := strings.Index(result, seg.resolvedValue)
		if idx < 0 {
			continue
		}
		result = result[:idx] + seg.raw + result[idx+len(seg.resolvedValue):]
	}

	if !dynvar.ContainsVariableReference(result) {
		return "", false
	}
	return result, true
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
		node, err := structpath.ParsePattern(candidate)
		if err != nil {
			continue
		}
		_, hasIndex := node.Index()
		if !hasIndex && !node.BracketStar() {
			continue
		}
		p, err := dyn.NewPathFromString(node.Parent().String())
		if err != nil {
			continue
		}
		parentValue, err := dyn.GetByPath(preResolved, p)
		if err != nil {
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

