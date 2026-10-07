package mutator

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
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

var artifactPath = structpath.MustParsePath("artifacts")

type resolveVariableReferences struct {
	// prefixes are top-level config keys.
	prefixes    []string
	pattern     *structpath.PatternNode
	lookupFn    func(structvar.View, *structpath.PathNode, *bundle.Bundle) (structvar.View, error)
	allowPathFn func(*structpath.PathNode) bool
	extraRounds int

	// includeResources allows resolving variables in 'resources', otherwise, they are excluded.
	//
	// includeResources can be used with appropriate pattern to avoid resolving variables
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
		pattern:          structpath.MustParsePattern("resources"),
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
		pattern:     structpath.MustParsePattern("variables.*.lookup"),
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
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
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
	varPath := structpath.MustParsePath("var")

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
// has a reference with one of the prefixes. It is conservative: the scope is all of
// the configuration outside "resources" (or only "resources"), regardless of the pattern.
// Like resolution itself, it records whether "artifacts" is referenced.
func (m *resolveVariableReferences) hasReferencesToResolve(b *bundle.Bundle, prefixes []*structpath.PathNode, varPath *structpath.PathNode) bool {
	onlyResources := m.includeResources && m.pattern != nil
	inScope := func(path string) bool {
		isResources := path == "resources" || strings.HasPrefix(path, "resources.")
		return (m.includeResources || !isResources) && (!onlyResources || isResources)
	}

	found := false
	check := func(s string) {
		ref, ok := structvar.NewRef(s)
		if !ok {
			return
		}
		for _, r := range ref.References() {
			path, err := structpath.ParsePath(r)
			if err != nil {
				// Let resolution report it.
				found = true
				return
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
	}

	root := reflect.ValueOf(&b.Config).Elem()
	rootType := root.Type()
	for i := range rootType.NumField() {
		name, _, _ := strings.Cut(rootType.Field(i).Tag.Get("json"), ",")
		if name == "" || name == "-" || !inScope(name) {
			continue
		}
		walkStrings(root.Field(i), check)
	}
	for path, ref := range b.Config.References() {
		if inScope(path.String()) {
			check(ref)
		}
	}
	return found
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

	// Resolve the references in each value matching the pattern (the whole configuration
	// if the pattern is nil). Template keys are relative to that value, like the paths
	// in cycle errors. The results are applied after all of them are resolved.
	type update struct {
		path  *structpath.PathNode
		value structvar.View
	}
	var updates []update
	resolveIn := func(p *structpath.PathNode, refs []referenceString) error {
		templates := map[string]structvar.Template{}
		paths := map[string]*structpath.PathNode{}
		for _, ref := range refs {
			key := ref.path.String()
			templates[key] = structvar.Template{Value: ref.value, Locations: ref.locs}
			paths[key] = structpath.Join(p, ref.path.AsSlice()...)
		}
		out, err := structvar.Resolve(templates, lookupFn)
		if err != nil {
			return err
		}
		for _, key := range slices.Sorted(maps.Keys(out)) {
			updates = append(updates, update{path: paths[key], value: out[key]})
		}
		return nil
	}

	var err error
	if m.pattern == nil {
		err = resolveIn(nil, m.referencesInScope(root))
	} else {
		err = structvar.ForEach(root, m.pattern, func(np *structpath.PathNode, v structvar.View) error {
			if !m.inScope(np) {
				return nil
			}
			return resolveIn(np, collectReferenceStrings(v))
		})
	}
	if err != nil {
		return hasUpdates, diags.Extend(resolveErrorDiags(err))
	}

	// Store the results in the typed configuration, converting them to the type of the
	// field (e.g. a variable reference resolved to an integer).
	for _, u := range updates {
		d, err := b.Config.Decode(u.path, u.value)
		diags = diags.Extend(d).Extend(diag.FromErr(err))
	}

	return hasUpdates, diags
}

// inScope reports whether path is in the part of the configuration this mutator resolves.
func (m *resolveVariableReferences) inScope(path *structpath.PathNode) bool {
	return m.includeResources || path.KeyAt(0) != "resources"
}

// referencesInScope returns the reference strings in the whole configuration,
// excluding "resources" unless resources are included.
func (m *resolveVariableReferences) referencesInScope(root structvar.View) []referenceString {
	if m.includeResources {
		return collectReferenceStrings(root)
	}
	var out []referenceString
	for k, v := range root.MapItems() {
		if !m.inScope(structpath.NewStringKey(nil, k)) {
			continue
		}
		p := structpath.NewStringKey(nil, k)
		for _, ref := range collectReferenceStrings(v) {
			ref.path = structpath.Join(p, ref.path.AsSlice()...)
			out = append(out, ref)
		}
	}
	return out
}

// referenceString is a string with a variable reference at path (relative to the
// value being resolved).
type referenceString struct {
	path  *structpath.PathNode
	value string
	locs  []diag.Location
}

// collectReferenceStrings returns the strings in v that contain variable references.
// Pure references in fields that cannot hold a string are strings in the view.
func collectReferenceStrings(v structvar.View) []referenceString {
	var out []referenceString
	_ = structvar.Walk(v, func(p *structpath.PathNode, v structvar.View) error {
		if s, ok := v.AsString(); ok {
			if _, ok := structvar.NewRef(s); ok {
				out = append(out, referenceString{path: p, value: s, locs: v.Locations()})
			}
		}
		return nil
	})
	return out
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
