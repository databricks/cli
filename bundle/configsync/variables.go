package configsync

import (
	"context"
	"errors"
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
//
// Restoration steps that introduce a reference at a position that didn't
// previously hold one (the Replace fallback and sibling-based Add restoration)
// are additionally gated by crossTargetGuard: a ${var.X} reference is only
// introduced when X is resolvable in every target, or when the write
// destination is the current target's override section.
func RestoreVariableReferences(ctx context.Context, b *bundle.Bundle, fieldChanges []FieldChange) error {
	preResolved, guard := loadPreResolvedConfig(ctx, b)
	if !preResolved.IsValid() {
		return errors.New("pre-resolved config unavailable; variable-backed fields will be hardcoded")
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
		allow := guard.allowFor(fc.FieldCandidates)

		var newValue any
		switch fc.Change.Operation {
		case OperationReplace:
			fieldValue, ok := preResolvedValueAt(preResolved, fc.FieldCandidates)
			if !ok {
				continue
			}
			newValue = restoreOriginalRefs(fc.Change.Value, fieldValue, resolved, allow)
		case OperationAdd:
			siblings, ok := sequenceSiblings(preResolved, fc.FieldCandidates)
			if !ok {
				continue
			}
			newValue = restoreFromSiblings(fc.Change.Value, siblings, resolved, allow)
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

// loadPreResolvedConfig loads the bundle's configuration through the standard
// loader mutators (entry point, includes, target overrides) but without
// variable resolution. The resulting dyn.Value is fully merged across files
// and targets, yet retains ${...} references as literal strings. Returns
// InvalidValue if loading fails (restoration is then skipped).
//
// The cross-target guard is built just before target selection: that is the
// only point where the full targets.*.variables map is still available
// (SelectTarget merges the chosen target into the root and removes the
// targets section).
func loadPreResolvedConfig(ctx context.Context, b *bundle.Bundle) (dyn.Value, *crossTargetGuard) {
	fresh := &bundle.Bundle{
		BundleRootPath: b.BundleRootPath,
		BundleRoot:     b.BundleRoot,
	}
	mutator.DefaultMutators(ctx, fresh)
	guard := newCrossTargetGuard(&fresh.Config)
	if target := b.Config.Bundle.Target; target != "" {
		if _, ok := fresh.Config.Targets[target]; ok {
			bundle.ApplyContext(ctx, fresh, mutator.SelectTarget(target))
		}
	}
	return fresh.Config.Value(), guard
}

// crossTargetGuard decides whether restoration may introduce a ${var.X}
// reference at a position that didn't previously hold one. Restoration runs
// with a single target selected, so the resolved variables include values that
// may only exist for that target (targets.<t>.variables assignments,
// BUNDLE_VAR_* environment variables, variable-overrides.json). Writing such a
// reference into a shared part of the configuration breaks the other targets:
// a variable without a root default that some target doesn't assign makes that
// target fail to load. The guard only admits variables that are resolvable in
// every target — unless the write destination is the current target's override
// section, which other targets never read.
type crossTargetGuard struct {
	// targetsRoot is the merged configuration before target selection, with
	// the targets.* subtree still present. Used to classify write destinations.
	targetsRoot dyn.Value

	// multiTarget is false when the bundle has at most one target; restoration
	// can't create cross-target breakage then and all variables are allowed.
	multiTarget bool

	// safe holds the variables that are resolvable in every target: a default
	// or lookup in the root variables block, or an assignment in every
	// target's variables block.
	safe map[string]bool
}

// newCrossTargetGuard captures variable declarations from a config that has
// not had a target selected yet.
func newCrossTargetGuard(cfg *config.Root) *crossTargetGuard {
	g := &crossTargetGuard{
		targetsRoot: cfg.Value(),
		multiTarget: len(cfg.Targets) > 1,
		safe:        map[string]bool{},
	}
	// The nil checks below guard direct construction in unit tests; in the
	// production path InitializeVariables and the loader produce non-nil
	// entries.
	for name, v := range cfg.Variables {
		if v != nil && (v.HasDefault() || v.Lookup != nil) {
			g.safe[name] = true
			continue
		}
		assignedEverywhere := len(cfg.Targets) > 0
		for _, target := range cfg.Targets {
			if target == nil {
				assignedEverywhere = false
				break
			}
			tv := target.Variables[name]
			if tv == nil || (tv.Default == nil && tv.Lookup == nil) {
				assignedEverywhere = false
				break
			}
		}
		g.safe[name] = assignedEverywhere
	}
	return g
}

// allowFor returns the variable admission predicate for a field change with
// the given path candidates (plain path first, targets.<t>.-prefixed second;
// see ResolveChanges).
func (g *crossTargetGuard) allowFor(candidates []string) func(string) bool {
	if !g.multiTarget || g.destinationInTarget(candidates) {
		return allowAllVariables
	}
	return func(name string) bool { return g.safe[name] }
}

func allowAllVariables(string) bool { return true }

// destinationInTarget reports whether the change can only be written inside
// the current target's override section. applyChange tries the plain path
// first, so the destination is the target section only when the plain path
// does not exist in the pre-target-selection configuration while the
// targets.<t>.-prefixed path does. Classification failures fall back to false,
// which applies the stricter shared-destination rule.
//
// Known limitation: candidates carry file-local numeric indexes (see
// yamlFileIndex), so for a sequence element defined only in the target
// override section the plain path may collide with a different element of the
// shared sequence and classify as shared. That errs in the safe direction:
// the value stays hardcoded instead of becoming a target-only reference.
func (g *crossTargetGuard) destinationInTarget(candidates []string) bool {
	if len(candidates) < 2 {
		return false
	}
	return !patternExists(g.targetsRoot, candidates[0]) && patternExists(g.targetsRoot, candidates[1])
}

// patternExists reports whether the (possibly [*]-terminated) path pattern
// resolves in root. For new-element Adds the trailing [*] is dropped so the
// check applies to the parent sequence.
func patternExists(root dyn.Value, pattern string) bool {
	node, err := structpath.ParsePattern(pattern)
	if err != nil {
		return false
	}
	if node.BracketStar() {
		node = node.Parent()
	}
	p, err := dyn.NewPathFromString(node.String())
	if err != nil {
		return false
	}
	_, err = dyn.GetByPath(root, p)
	return err == nil
}

// varRefsAllowed reports whether every ${var.X} reference in s names a
// variable admitted by allow. References outside the var namespace
// (${bundle.X}, ${workspace.X}, ${resources.X.Y.id}) are exempt.
func varRefsAllowed(s string, allow func(string) bool) bool {
	ref, ok := dynvar.NewRef(dyn.V(s))
	if !ok {
		return true
	}
	for _, key := range ref.References() {
		p, err := dyn.NewPathFromString(key)
		if err != nil || !p.HasPrefix(varPrefix) || len(p) < 2 {
			continue
		}
		if !allow(p[1].Key()) {
			return false
		}
	}
	return true
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
	if err := db.Open(ctx, statePath, dstate.WithRecovery(false), dstate.WithWrite(false)); err != nil {
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
//
// Only the fallback consults allow: the other two steps re-emit a reference
// that already exists at this exact position, which can't introduce a
// cross-target leak.
func restoreOriginalRefs(value any, preResolved, resolved dyn.Value, allow func(string) bool) any {
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
			if ref, ok := matchAnyVariable(value, resolved); ok && varRefsAllowed(ref, allow) {
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
			v[key] = restoreOriginalRefs(val, childPre, resolved, allow)
		}
		return v

	case []any:
		preSeq, _ := preResolved.AsSequence()
		for i, val := range v {
			var childPre dyn.Value
			if i < len(preSeq) {
				childPre = preSeq[i]
			}
			v[i] = restoreOriginalRefs(val, childPre, resolved, allow)
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
//
// Every restored reference is checked against allow: sibling references come
// from other positions (possibly other files or the target override section),
// so substituting one always introduces a reference at a new position.
func restoreFromSiblings(value any, siblings []dyn.Value, resolved dyn.Value, allow func(string) bool) any {
	return restoreFromSiblingsAt(value, siblings, resolved, dyn.EmptyPath, allow)
}

func restoreFromSiblingsAt(value any, siblings []dyn.Value, resolved dyn.Value, relPath dyn.Path, allow func(string) bool) any {
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
		// The allow check runs after unique-match selection so that variables
		// rejected by the guard still count toward ambiguity rather than
		// promoting another candidate.
		if len(refs) == 1 {
			for ref := range refs {
				if varRefsAllowed(ref, allow) {
					return ref
				}
			}
		}
		return value

	case map[string]any:
		for key, val := range v {
			v[key] = restoreFromSiblingsAt(val, siblings, resolved, relPath.Append(dyn.Key(key)), allow)
		}
		return v

	case []any:
		for i, val := range v {
			v[i] = restoreFromSiblingsAt(val, siblings, resolved, relPath.Append(dyn.Index(i)), allow)
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
		case dyn.KindInvalid, dyn.KindMap, dyn.KindSequence, dyn.KindFloat, dyn.KindTime, dyn.KindNil:
			// Skip non-scalar / unsupported variable values.
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
