package mutator

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/config/walk"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

/*
For pathological cases, output and time grow exponentially.

On my laptop, timings for acceptance/bundle/variables/complex-cycle:
rounds           time

	 9          0.10s
	10          0.13s
	11          0.27s
	12          0.68s
	13          1.98s
	14          6.28s
	15         21.70s
	16         78.16s
*/
const maxResolutionRounds = 11

// List of prefixes to be used by default in ResolveVariableReferencesOnlyResources/ResolveVariableReferencesWithoutResources
// Prefixes specify which references are resolves, e.g. ${bundle...} and so on.
// This list does not include "artifacts" because this section will be modified in build phase and variable resolution happens in initialize phase.
// This list does not include "resources" because some of those references are known after resource is deployed.
var defaultPrefixes = []string{
	"bundle",
	"workspace",
	"variables",
}

var artifactPath = structpath.NewPath(nil, "artifacts")

// resolveScope is the part of the configuration a [resolveVariableReferences] resolves.
type resolveScope int

const (
	// scopeConfig is the whole configuration ("resources" only if includeResources).
	scopeConfig resolveScope = iota
	// scopeResources is "resources".
	scopeResources
	// scopeVariableLookups is "variables.*.lookup", each lookup resolved separately.
	scopeVariableLookups
)

type resolveVariableReferences struct {
	// prefixes are top-level config keys.
	prefixes    []string
	scope       resolveScope
	lookupFn    func(structvar.View, *structpath.PathNode, *bundle.Bundle) (structvar.View, error)
	allowPathFn func(*structpath.PathNode) bool
	extraRounds int

	// includeResources allows resolving variables in 'resources', otherwise, they are excluded.
	//
	// includeResources can be used with scopeResources to avoid resolving variables
	// outside of 'resources'.
	includeResources bool

	artifactsReferenceUsed bool

	// excludePaths lists variable reference paths (e.g. "workspace.file_path") whose
	// resolution should be skipped. References to these paths remain unresolved so a
	// later mutator can set the value and re-run resolution.
	excludePaths []string
}

func ResolveVariableReferencesOnlyResources(prefixes ...string) bundle.Mutator {
	if len(prefixes) == 0 {
		prefixes = defaultPrefixes
	}
	return &resolveVariableReferences{
		prefixes:         prefixes,
		lookupFn:         lookup,
		extraRounds:      maxResolutionRounds - 1,
		scope:            scopeResources,
		includeResources: true,
	}
}

func ResolveVariableReferencesWithoutResources(prefixes ...string) bundle.Mutator {
	if len(prefixes) == 0 {
		prefixes = defaultPrefixes
	}
	return &resolveVariableReferences{
		prefixes:    prefixes,
		lookupFn:    lookup,
		extraRounds: maxResolutionRounds - 1,
	}
}

func ResolveVariableReferencesInLookup() bundle.Mutator {
	return &resolveVariableReferences{
		prefixes:    defaultPrefixes,
		scope:       scopeVariableLookups,
		lookupFn:    lookupForVariables,
		extraRounds: maxResolutionRounds - 1,
	}
}

// ResolveVolumePathReferencesOnlyResources resolves only references to resources.volumes.*.volume_path.
func ResolveVolumePathReferencesOnlyResources() bundle.Mutator {
	return &resolveVariableReferences{
		prefixes:         []string{"resources"},
		lookupFn:         lookup,
		allowPathFn:      isVolumePathReferencePath,
		extraRounds:      maxResolutionRounds - 1,
		includeResources: true,
	}
}

func lookup(v structvar.View, path *structpath.PathNode, b *bundle.Bundle) (structvar.View, error) {
	if b.IsSourceLinkedDeployment() {
		if path.String() == "workspace.file_path" {
			return structvar.NewView(&b.SyncRootPath, nil, nil), nil
		}
	}
	// Future opportunity: if we lookup this path in both the given root
	// and the synthesized root, we know if it was explicitly set or implied to be empty.
	// Then we can emit a warning if it was not explicitly set.
	return lookupValue(v, path)
}

// lookupValue returns the value at path. Fields that are declared in the type but not
// set resolve to their zero value. This enables users to interpolate variable references
// to fields that haven't been set, e.g. ${bundle.git.origin_url} resolves to an empty
// string if a bundle isn't located in a Git repository (yet).
func lookupValue(x structvar.View, path *structpath.PathNode) (structvar.View, error) {
	return x.LookupWithDefaults(path)
}

func lookupForVariables(v structvar.View, path *structpath.PathNode, b *bundle.Bundle) (structvar.View, error) {
	if path.KeyAt(0) != "variables" {
		return lookup(v, path, b)
	}

	varV, err := lookupValue(v, path.Parent())
	if err != nil {
		return structvar.View{}, err
	}

	if lookupV := varV.Get("lookup"); lookupV.IsValid() {
		var vl variable.Lookup
		if _, err := (&structvar.StructVar{Value: &vl}).Assign(nil, lookupV); err != nil {
			return structvar.View{}, err
		}
		if vl.String() != "" {
			return structvar.View{}, errors.New("lookup variables cannot contain references to another lookup variables")
		}
	}

	return lookup(v, path, b)
}

func (m *resolveVariableReferences) Name() string {
	if m.includeResources {
		return "ResolveVariableReferences(resources)"
	} else {
		return "ResolveVariableReferences"
	}
}

func (m *resolveVariableReferences) Validate(ctx context.Context, b *bundle.Bundle) error {
	return nil
}

func (m *resolveVariableReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	prefixes := make([]*structpath.PathNode, len(m.prefixes))
	for i, prefix := range m.prefixes {
		prefixes[i] = structpath.NewPath(nil, prefix)
	}

	// The path ${var.foo} is a shorthand for ${variables.foo.value}.
	// We rewrite it here to make the resolution logic simpler.
	varPath := structpath.NewPath(nil, "var")

	// Resolution converts the whole configuration on every round; skip it if no
	// reference would be resolved.
	if !m.hasReferencesToResolve(b, prefixes, varPath) {
		if m.artifactsReferenceUsed {
			b.Metrics.SetBoolValue("artifacts_reference_used", true)
		}
		return nil
	}

	var diags diag.Diagnostics
	maxRounds := 1 + m.extraRounds

	for round := range maxRounds {
		hasUpdates, newDiags := m.resolveOnce(b, prefixes, varPath)

		diags = diags.Extend(newDiags)

		if diags.HasError() {
			break
		}

		if !hasUpdates {
			break
		}

		// Another round would only find out that nothing is left to resolve.
		if !m.hasReferencesToResolve(b, prefixes, varPath) {
			break
		}

		if round >= maxRounds-1 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Variables references are too deep, stopping resolution after %d rounds. Unresolved variables may remain.", round+1),
				// Would be nice to include names of the variables there, but that would complicate things more
			})
			break
		}
	}

	if m.artifactsReferenceUsed {
		b.Metrics.SetBoolValue("artifacts_reference_used", true)
	}

	return diags
}

// hasReferencesToResolve reports whether the configuration in scope of this mutator
// has a reference with one of the prefixes. Like resolution itself, it records whether
// "artifacts" is referenced.
func (m *resolveVariableReferences) hasReferencesToResolve(b *bundle.Bundle, prefixes []*structpath.PathNode, varPath *structpath.PathNode) bool {
	found := false
	m.walkStrings(&b.Config, func(_ *walk.Node, s string) (string, bool) {
		// Every string is checked: each can record the "artifacts" reference.
		found = m.referencesPrefix(s, prefixes, varPath) || found
		return "", false
	})
	for path, ref := range b.Config.References() {
		if m.inScope(path) {
			found = m.referencesPrefix(ref, prefixes, varPath) || found
		}
	}
	return found
}

// referencesPrefix reports whether s has a reference with one of the prefixes (or one
// that does not parse, to let resolution report it), and records an "artifacts" reference.
func (m *resolveVariableReferences) referencesPrefix(s string, prefixes []*structpath.PathNode, varPath *structpath.PathNode) bool {
	ref, ok := structvar.NewRef(s)
	if !ok {
		return false
	}
	found := false
	for _, r := range ref.References() {
		path, err := structpath.ParsePath(r)
		if err != nil {
			return true
		}
		if path.HasPrefix(varPath) {
			path = structpath.Join(structpath.NewStringKey(nil, "variables"), path.SkipPrefix(1).AsSlice()...)
		}
		if path.HasPrefix(artifactPath) {
			m.artifactsReferenceUsed = true
		}
		if slices.ContainsFunc(prefixes, path.HasPrefix) {
			found = true
		}
	}
	return found
}

// walkStrings calls fn for the strings with references in the scope of this mutator.
func (m *resolveVariableReferences) walkStrings(r *config.Root, fn walk.StringFunc) {
	switch m.scope {
	case scopeConfig:
		if m.includeResources {
			walk.Strings(r, walk.WithReferences, fn)
		} else {
			walk.StringsWithoutResources(r, walk.WithReferences, fn)
		}
	case scopeResources:
		walk.ResourceStrings(r, walk.WithReferences, fn)
	case scopeVariableLookups:
		walk.VariableLookupStrings(r, walk.WithReferences, fn)
	}
}

// inScope reports whether path is in the part of the configuration this mutator resolves.
func (m *resolveVariableReferences) inScope(path *structpath.PathNode) bool {
	switch m.scope {
	case scopeResources:
		return path.KeyAt(0) == "resources"
	case scopeVariableLookups:
		return path.KeyAt(0) == "variables" && path.KeyAt(2) == "lookup"
	default:
		return m.includeResources || path.KeyAt(0) != "resources"
	}
}

func (m *resolveVariableReferences) resolveOnce(b *bundle.Bundle, prefixes []*structpath.PathNode, varPath *structpath.PathNode) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	hasUpdates := false
	root := b.Config.View()

	lookupFn := func(sp *structpath.PathNode) (structvar.View, error) {
		path := sp
		// Rewrite the shorthand path ${var.foo} into ${variables.foo.value}.
		if path.HasPrefix(varPath) {
			path = structpath.Join(structpath.NewPath(nil, "variables", path.KeyAt(1), "value"), path.SkipPrefix(2).AsSlice()...)
		}

		// If the path starts with "artifacts", we need to add a metric to track if this reference is used.
		if path.HasPrefix(artifactPath) {
			m.artifactsReferenceUsed = true
		}

		// Perform resolution only if the path starts with one of the specified prefixes.
		if slices.ContainsFunc(prefixes, path.HasPrefix) {
			if slices.Contains(m.excludePaths, path.String()) {
				return structvar.View{}, structvar.ErrSkipResolution
			}
			if m.allowPathFn != nil && !m.allowPathFn(path) {
				return structvar.View{}, structvar.ErrSkipResolution
			}
			value, err := m.lookupFn(root, path, b)
			hasUpdates = hasUpdates || (err == nil && value.IsValid())
			return value, err
		}

		return structvar.View{}, structvar.ErrSkipResolution
	}

	// Resolve the references in each value of the scope: the whole configuration,
	// "resources", or each variable lookup. Template keys are relative to that value, like
	// the paths in cycle errors. The results are applied after all of them are resolved.
	var updates []resolvedValue
	var err error
	for _, refs := range m.referenceStrings(&b.Config) {
		if updates, err = resolveReferenceStrings(updates, refs, lookupFn); err != nil {
			break
		}
	}
	if err != nil {
		return hasUpdates, diags.Extend(resolveErrorDiags(err))
	}

	// Store the results in the typed configuration, converting them to the type of the
	// field (e.g. a variable reference resolved to an integer).
	// A value that cannot be converted is dropped, which moves the elements after it in
	// a sequence: apply the updates from the last element to the first.
	slices.SortStableFunc(updates, func(a, b resolvedValue) int { return -comparePaths(a.path, b.path) })
	var updateDiags []diag.Diagnostics
	for _, u := range updates {
		d, err := b.Config.Decode(u.path, u.value)
		updateDiags = append(updateDiags, d.Extend(diag.FromErr(err)))
	}
	for _, d := range slices.Backward(updateDiags) {
		diags = diags.Extend(d)
	}

	return hasUpdates, diags
}

// comparePaths orders paths by their components; indices are compared as numbers.
func comparePaths(a, b *structpath.PathNode) int {
	as, bs := a.AsSlice(), b.AsSlice()
	for i := 0; i < len(as) && i < len(bs); i++ {
		ai, aok := as[i].Index()
		bi, bok := bs[i].Index()
		if aok && bok {
			if c := cmp.Compare(ai, bi); c != 0 {
				return c
			}
			continue
		}
		if c := strings.Compare(as[i].String(), bs[i].String()); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(as), len(bs))
}

// resolvedValue is the value a reference string at path resolved to.
type resolvedValue struct {
	path  *structpath.PathNode
	value structvar.View
}

// resolveReferenceStrings resolves the reference strings of one base together and
// appends the results to out, in path order. Template keys are relative to the base.
func resolveReferenceStrings(out []resolvedValue, refs []referenceString, lookup structvar.Lookup) ([]resolvedValue, error) {
	templates := map[string]structvar.Template{}
	paths := map[string]*structpath.PathNode{}
	for _, ref := range refs {
		key := ref.path.SkipPrefix(ref.base.Len()).String()
		templates[key] = structvar.Template{Value: ref.value, Locations: ref.locs}
		paths[key] = ref.path
	}
	resolved, err := structvar.Resolve(templates, lookup)
	if err != nil {
		return out, err
	}
	for _, key := range slices.Sorted(maps.Keys(resolved)) {
		out = append(out, resolvedValue{path: paths[key], value: resolved[key]})
	}
	return out, nil
}

// referenceString is a string with a variable reference at path. Strings with the same
// base (the scope value they are in) are resolved together.
type referenceString struct {
	base  *structpath.PathNode
	path  *structpath.PathNode
	value string
	locs  []diag.Location
}

// referenceStrings returns the strings with variable references in the scope of this
// mutator, grouped by base in the order of the bases. Pure references in fields that
// cannot hold a string are included as their reference strings.
func (m *resolveVariableReferences) referenceStrings(r *config.Root) [][]referenceString {
	var out []referenceString
	m.walkStrings(r, func(n *walk.Node, s string) (string, bool) {
		if _, ok := structvar.NewRef(s); ok {
			out = append(out, referenceString{path: n.Path(), value: s, locs: n.Locations().Get()})
		}
		return "", false
	})
	for path, ref := range r.References() {
		if m.inScope(path) {
			out = append(out, referenceString{path: path, value: ref, locs: r.GetLocationsOfExact(path)})
		}
	}

	groups := map[string][]referenceString{}
	for _, ref := range out {
		ref.base = m.base(ref.path)
		key := ref.base.String()
		groups[key] = append(groups[key], ref)
	}
	result := make([][]referenceString, 0, len(groups))
	for _, key := range slices.Sorted(maps.Keys(groups)) {
		result = append(result, groups[key])
	}
	return result
}

// base returns the scope value path is in.
func (m *resolveVariableReferences) base(path *structpath.PathNode) *structpath.PathNode {
	switch m.scope {
	case scopeResources:
		return path.Prefix(1)
	case scopeVariableLookups:
		return path.Prefix(3)
	default:
		return nil
	}
}

// resolveErrorDiags renders "did you mean" suggestions as a diagnostic Detail so
// libs/diag owns the multi-line formatting.
func resolveErrorDiags(err error) diag.Diagnostics {
	refErr, ok := errors.AsType[*structvar.ReferenceError](err)
	if !ok || len(refErr.Suggestions) == 0 {
		return diag.FromErr(err)
	}

	header := "did you mean:"
	if len(refErr.Suggestions) > 1 {
		header = "did you mean one of:"
	}
	var detail strings.Builder
	detail.WriteString(header)
	for _, ref := range refErr.Suggestions {
		detail.WriteString("\n  ${" + ref + "}")
	}

	return diag.Diagnostics{{
		Severity: diag.Error,
		Summary:  refErr.Error(),
		Detail:   detail.String(),
	}}
}

func isVolumePathReferencePath(path *structpath.PathNode) bool {
	if path.Len() != 4 {
		return false
	}
	return path.KeyAt(0) == "resources" &&
		path.KeyAt(1) == "volumes" &&
		path.KeyAt(3) == "volume_path"
}
