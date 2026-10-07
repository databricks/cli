package configsync

import (
	"context"
	"errors"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// varPrefix is the path prefix for the ${var.X} shorthand.
var varPrefix = structpath.NewStringKey(nil, "var")

// RestoreVariableReferences replaces hardcoded change values with variable
// references (${var.foo}, ${bundle.target}, ${resources.X.Y.id}) when the
// value can be traced back to a reference in the original YAML. Resource IDs
// are injected from state since they aren't materialized into the resolved
// config's tree.
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
// The caller supplies preResolved (see LoadPreResolvedConfig): the merged
// config where ${var.X} and ${resources.X.Y.id} references are still literal
// strings — enabling correct sibling lookup even for sequences split across
// files via target overrides.
// Restoration counts by mechanism are accumulated into stats (used for
// telemetry); pass nil when counters are not needed (the counter methods are
// nil-safe).
func RestoreVariableReferences(ctx context.Context, b *bundle.Bundle, fieldChanges []FieldChange, preResolved structvar.View, stats *RestoreStats) error {
	if !preResolved.IsValid() {
		return errors.New("pre-resolved config unavailable; variable-backed fields will be hardcoded")
	}
	resolved := resolvedConfig{view: b.Config.View(), overrides: map[string]any{}}

	// Mirror mutator.lookup's source-linked deployment override: when enabled,
	// ${workspace.file_path} resolves to b.SyncRootPath rather than the typed
	// workspace.file_path field (which still holds the default deploy path).
	// Without this, substring matching against the typed value misses the
	// actual deployed path and variables are lost on Replace. Keep this in
	// sync with mutator.lookup if new overrides are added there.
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		resolved.overrides["workspace.file_path"] = b.SyncRootPath
	}

	// Augment resolved with resource IDs from state — only when the config
	// actually uses ${resources.X.Y.id} references. The IDs aren't materialized
	// into b.Config (they live in the StateDB), so we inject them here
	// to enable sibling-based restoration. Skipped entirely for bundles with
	// no resource refs to avoid opening state DB files unnecessarily.
	resourceRefs := collectResourceIDRefs(preResolved)
	if len(resourceRefs) > 0 {
		if lookup := resourceIDLookup(b); lookup != nil {
			injectResourceIDs(ctx, resolved, resourceRefs, lookup)
		} else {
			log.Debugf(ctx, "variable restoration: state DB unavailable, skipping resource ID injection for %d refs", len(resourceRefs))
		}
	}

	for i := range fieldChanges {
		fc := &fieldChanges[i]

		var newValue any
		switch fc.Change.Operation {
		case OperationReplace:
			fieldValue, ok := preResolvedValueAt(preResolved, fc.originalPath)
			if !ok {
				continue
			}
			newValue = restoreOriginalRefs(fc.Change.Value, fieldValue, resolved, stats)
		case OperationAdd:
			siblings, ok := sequenceSiblings(preResolved, fc.originalPath)
			if !ok {
				continue
			}
			newValue = restoreFromSiblings(fc.Change.Value, siblings, resolved, stats)
		case OperationUnknown, OperationRemove, OperationSkip:
			continue
		}

		fc.Change = &ConfigChangeDesc{
			Operation: fc.Change.Operation,
			Value:     newValue,
		}
	}
	return nil
}

// LoadPreResolvedConfig loads the bundle's configuration through the standard
// loader mutators (entry point, includes, target overrides) but without
// variable resolution. The resulting configuration is fully merged across files
// and targets, yet retains ${...} references as literal strings. Returns
// an invalid value if loading fails (restoration is then skipped).
func LoadPreResolvedConfig(ctx context.Context, b *bundle.Bundle) structvar.View {
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

	// Keyed sequences merge in the initialize phase, which this reload skips. Without
	// them the sequences here stay in file order while the change paths address the
	// merged, key-sorted order, so a lookup would read a different element.
	bundle.ApplySeqContext(ctx, fresh,
		resourcemutator.MergeJobClusters(),
		resourcemutator.MergeJobParameters(),
		resourcemutator.MergeJobTasks(),
		resourcemutator.MergePipelineClusters(),
		resourcemutator.MergeApps(),
	)
	return fresh.Config.View()
}

// resolvedConfig is the bundle's resolved configuration plus values that are known
// only outside of it (the source-linked file path, resource IDs from state).
type resolvedConfig struct {
	view      structvar.View
	overrides map[string]any // by path
}

// lookup returns the value at path as a Go value, and whether it exists.
func (c resolvedConfig) lookup(path *structpath.PathNode) (any, bool) {
	if v, ok := c.overrides[path.String()]; ok {
		return v, true
	}
	v := c.view.Lookup(path)
	if !v.IsValid() {
		return nil, false
	}
	return v.AsAny(), true
}

// resourceIDLookup returns a function that resolves resource keys to their
// deployed IDs from the direct StateDB already open on b.DeploymentBundle.
// Returns nil if no state is available.
func resourceIDLookup(b *bundle.Bundle) func(string) string {
	if b.DeploymentBundle.StateDB.Path != "" {
		return b.DeploymentBundle.StateDB.GetResourceID
	}
	return nil
}

// collectResourceIDRefs walks the pre-resolved merged config to find pure
// ${resources.<kind>.<name>.id} references. Returns the unique set of paths
// so the caller can inject IDs at those positions; returns nil if no such
// references exist.
func collectResourceIDRefs(preResolved structvar.View) []*structpath.PathNode {
	seen := map[string]bool{}
	var paths []*structpath.PathNode
	_ = structvar.Walk(preResolved, func(_ *structpath.PathNode, v structvar.View) error {
		s, ok := v.AsString()
		if !ok || !structvar.IsPureVariableReference(s) || seen[s] {
			return nil
		}
		seen[s] = true
		p, ok := structvar.PureReferenceToPath(s)
		if !ok || p.Len() != 4 || p.KeyAt(0) != "resources" || p.KeyAt(3) != "id" {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	return paths
}

// injectResourceIDs records IDs from state for the given resource reference paths in
// resolved. Skips references whose IDs aren't in state.
func injectResourceIDs(ctx context.Context, resolved resolvedConfig, paths []*structpath.PathNode, lookupID func(string) string) {
	for _, p := range paths {
		resourceKey := p.Prefix(3).String()
		id := lookupID(resourceKey)
		if id == "" {
			log.Debugf(ctx, "variable restoration: no state entry for resource %q", resourceKey)
			continue
		}
		resolved.overrides[p.String()] = id
	}
}

// resolveReferencePath converts a variable reference string to the path
// where its resolved value can be found in the bundle config. It applies the
// same ${var.X} → variables.X.value shorthand rewriting as the variable
// resolution mutator.
func resolveReferencePath(refStr string) (*structpath.PathNode, bool) {
	p, ok := structvar.PureReferenceToPath(refStr)
	if !ok {
		return nil, false
	}

	if p.HasPrefix(varPrefix) && p.Len() >= 2 {
		newPath := structpath.NewPath(nil, "variables", p.KeyAt(1), "value")
		return structpath.Join(newPath, p.AsSlice()[2:]...), true
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
func restoreOriginalRefs(value any, preResolved structvar.View, resolved resolvedConfig, stats *RestoreStats) any {
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
				stats.incRetargeted()
				return ref
			}
		}
		return value

	case map[string]any:
		for key, val := range v {
			v[key] = restoreOriginalRefs(val, preResolved.Get(key), resolved, stats)
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = restoreOriginalRefs(val, preResolved.Index(i), resolved, stats)
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
func restoreFromSiblings(value any, siblings []structvar.View, resolved resolvedConfig, stats *RestoreStats) any {
	return restoreFromSiblingsAt(value, siblings, resolved, nil, stats)
}

func restoreFromSiblingsAt(value any, siblings []structvar.View, resolved resolvedConfig, relPath *structpath.PathNode, stats *RestoreStats) any {
	switch v := value.(type) {
	case string, bool, int64:
		refs := map[string]struct{}{}
		strVal, isStr := value.(string)
		for _, sib := range siblings {
			sv := sib.Lookup(relPath)
			s, ok := sv.AsString()
			if !ok {
				continue
			}
			if structvar.IsPureVariableReference(s) {
				rp, ok := resolveReferencePath(s)
				if !ok {
					continue
				}
				rv, found := resolved.lookup(rp)
				if !found {
					continue
				}
				if rv == value {
					refs[s] = struct{}{}
				}
			} else if isStr && structvar.ContainsVariableReference(s) {
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
				stats.incFromSiblings()
				return ref
			}
		}
		return value

	case map[string]any:
		for key, val := range v {
			v[key] = restoreFromSiblingsAt(val, siblings, resolved, structpath.NewStringKey(relPath, key), stats)
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = restoreFromSiblingsAt(val, siblings, resolved, structpath.NewIndex(relPath, i), stats)
		}
		return v

	default:
		return value
	}
}

// isPureVarRef reports whether the pre-resolved value at the field is a pure
// ${var.X} reference. Used to gate the fallback substitution: only fields that
// already used a variable can be re-targeted to a different variable.
func isPureVarRef(preResolved structvar.View) bool {
	if !preResolved.IsValid() {
		return false
	}
	s, ok := preResolved.AsString()
	if !ok || !structvar.IsPureVariableReference(s) {
		return false
	}
	p, ok := structvar.PureReferenceToPath(s)
	return ok && p.HasPrefix(varPrefix)
}

// matchAnyVariable searches all bundle variables for a unique scalar value that
// equals remoteValue. Returns the ${var.X} reference on a unique match, ""
// otherwise. Multiple matches are treated as ambiguous and skipped.
func matchAnyVariable(remoteValue any, resolved resolvedConfig) (string, bool) {
	var match string
	count := 0
	for name, variable := range resolved.view.Get("variables").MapItems() {
		v := variable.Get("value")
		switch v.Kind() {
		case structvar.KindString, structvar.KindInt, structvar.KindBool:
			if v.AsAny() == remoteValue {
				match = pathToRef(structpath.NewStringKey(varPrefix, name))
				count++
			}
		case structvar.KindInvalid, structvar.KindMap, structvar.KindSequence, structvar.KindFloat, structvar.KindTime, structvar.KindNil:
			// Skip non-scalar / unsupported variable values.
		}
	}
	if count == 1 {
		return match, true
	}
	return "", false
}

// pathToRef formats a path as a "${...}" interpolation reference.
func pathToRef(p *structpath.PathNode) string {
	return "${" + p.String() + "}"
}

// matchOriginalRef checks if the pre-resolved config value at this position
// was a pure variable reference whose resolved value equals remoteValue.
func matchOriginalRef(remoteValue any, preResolved structvar.View, resolved resolvedConfig) (string, bool) {
	if !preResolved.IsValid() {
		return "", false
	}
	s, ok := preResolved.AsString()
	if !ok || !structvar.IsPureVariableReference(s) {
		return "", false
	}

	resolvedPath, ok := resolveReferencePath(s)
	if !ok {
		return "", false
	}

	resolvedV, found := resolved.lookup(resolvedPath)
	if !found {
		return "", false
	}

	if resolvedV == remoteValue {
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
func restoreCompoundInterpolation(remoteValue string, preResolved structvar.View, resolved resolvedConfig) (string, bool) {
	if !preResolved.IsValid() {
		return "", false
	}
	template, ok := preResolved.AsString()
	if !ok || !structvar.ContainsVariableReference(template) || structvar.IsPureVariableReference(template) {
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

	if !structvar.ContainsVariableReference(result) {
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
func parseTemplateSegments(template string, resolved resolvedConfig) []templateSegment {
	ref, ok := structvar.NewRef(template)
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

		resolvedV, found := resolved.lookup(resolvedPath)
		if !found {
			return nil
		}

		resolvedStr, ok := resolvedV.(string)
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

// preResolvedValueAt returns the pre-resolved value at the field path,
// if the field exists in the merged pre-resolved config.
func preResolvedValueAt(preResolved structvar.View, fieldPath string) (structvar.View, bool) {
	p, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return structvar.View{}, false
	}
	v := preResolved.Lookup(p)
	return v, v.IsValid()
}

// parentIsVariableReference reports whether the parent of fieldPath resolves to a
// scalar variable reference in the pre-resolved config (e.g. spark_conf:
// ${var.spark_conf}). A nested key or index cannot be written into such a scalar,
// so config-remote-sync skips the change. fieldPath is a merged-index path, the
// same space as FieldChange.originalPath.
func parentIsVariableReference(preResolved structvar.View, fieldPath string) bool {
	node, err := structpath.ParsePattern(fieldPath)
	if err != nil {
		return false
	}
	parent := node.Parent()
	if parent == nil {
		return false
	}
	v, ok := preResolvedValueAt(preResolved, parent.String())
	if !ok {
		return false
	}
	s, ok := v.AsString()
	return ok && structvar.ContainsVariableReference(s)
}

// sequenceSiblings returns the sibling elements of the parent sequence when
// the field change represents adding a new element to a sequence. The path's
// last component must be an index ([*] or [N]) and the parent must resolve
// to a sequence in the pre-resolved config. Returns false for non-sequence
// Adds (e.g., new map fields).
func sequenceSiblings(preResolved structvar.View, fieldPath string) ([]structvar.View, bool) {
	node, err := structpath.ParsePattern(fieldPath)
	if err != nil {
		return nil, false
	}
	_, hasIndex := node.Index()
	if !hasIndex && !node.BracketStar() {
		return nil, false
	}
	p, err := structpath.ParsePath(node.Parent().String())
	if err != nil {
		return nil, false
	}
	parentValue := preResolved.Lookup(p)
	if parentValue.Kind() != structvar.KindSequence {
		return nil, false
	}
	var seq []structvar.View
	for _, elem := range parentValue.Sequence() {
		seq = append(seq, elem)
	}
	return seq, true
}
