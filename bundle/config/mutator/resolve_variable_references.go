package mutator

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/libs/dyn/merge"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
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

type resolveVariableReferences struct {
	prefixes    []string
	pattern     dyn.Pattern
	lookupFn    func(dyn.Value, dyn.Path, *bundle.Bundle) (dyn.Value, error)
	skipFn      func(dyn.Value) bool
	extraRounds int

	// includeResources allows resolving variables in 'resources', otherwise, they are excluded.
	//
	// includeResources can be used with appropriate pattern to avoid resolving variables
	// outside of 'resources'.
	includeResources bool
}

func ResolveVariableReferencesOnlyResources(prefixes ...string) bundle.Mutator {
	return &resolveVariableReferences{
		prefixes:         prefixes,
		lookupFn:         lookup,
		extraRounds:      maxResolutionRounds - 1,
		pattern:          dyn.NewPattern(dyn.Key("resources")),
		includeResources: true,
	}
}

func ResolveVariableReferencesWithoutResources(prefixes ...string) bundle.Mutator {
	return &resolveVariableReferences{
		prefixes:    prefixes,
		lookupFn:    lookup,
		extraRounds: maxResolutionRounds - 1,
	}
}

func ResolveVariableReferencesInLookup() bundle.Mutator {
	return &resolveVariableReferences{
		prefixes: []string{
			"bundle",
			"workspace",
			"variables",
		},
		pattern:     dyn.NewPattern(dyn.Key("variables"), dyn.AnyKey(), dyn.Key("lookup")),
		lookupFn:    lookupForVariables,
		extraRounds: maxResolutionRounds - 1,
	}
}

func lookup(v dyn.Value, path dyn.Path, b *bundle.Bundle) (dyn.Value, error) {
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		if path.String() == "workspace.file_path" {
			return dyn.V(b.SyncRootPath), nil
		}
	}
	// Future opportunity: if we lookup this path in both the given root
	// and the synthesized root, we know if it was explicitly set or implied to be empty.
	// Then we can emit a warning if it was not explicitly set.
	return dyn.GetByPath(v, path)
}

func lookupForVariables(v dyn.Value, path dyn.Path, b *bundle.Bundle) (dyn.Value, error) {
	if path[0].Key() != "variables" {
		return lookup(v, path, b)
	}

	varV, err := dyn.GetByPath(v, path[:len(path)-1])
	if err != nil {
		return dyn.InvalidValue, err
	}

	var vv variable.Variable
	err = convert.ToTyped(&vv, varV)
	if err != nil {
		return dyn.InvalidValue, err
	}

	if vv.Lookup != nil && vv.Lookup.String() != "" {
		return dyn.InvalidValue, errors.New("lookup variables cannot contain references to another lookup variables")
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
	prefixes := make([]dyn.Path, len(m.prefixes))
	for i, prefix := range m.prefixes {
		prefixes[i] = dyn.MustPathFromString(prefix)
	}

	// The path ${var.foo} is a shorthand for ${variables.foo.value}.
	// We rewrite it here to make the resolution logic simpler.
	varPath := dyn.NewPath(dyn.Key("var"))

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

		if round >= maxRounds-1 {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Variables references are too deep, stopping resolution after %d rounds. Unresolved variables may remain.", round+1),
				// Would be nice to include names of the variables there, but that would complicate things more
			})
			break
		}
	}
	return diags
}

func (m *resolveVariableReferences) resolveOnce(b *bundle.Bundle, prefixes []dyn.Path, varPath dyn.Path) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	hasUpdates := false
	err := m.selectivelyMutate(b, func(root dyn.Value) (dyn.Value, error) {
		// Synthesize a copy of the root that has all fields that are present in the type
		// but not set in the dynamic value set to their corresponding empty value.
		// This enables users to interpolate variable references to fields that haven't
		// been explicitly set in the dynamic value.
		//
		// For example: ${bundle.git.origin_url} should resolve to an empty string
		// if a bundle isn't located in a Git repository (yet).
		//
		// This is consistent with the behavior prior to using the dynamic value system.
		//
		// We can ignore the diagnostics return value because we know that the dynamic value
		// has already been normalized when it was first loaded from the configuration file.
		//
		normalized, _ := convert.Normalize(b.Config, root, convert.IncludeMissingFields)

		// If the pattern is nil, we resolve references in the entire configuration.
		root, err := dyn.MapByPattern(root, m.pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Resolve variable references in all values.
			return dynvar.Resolve(v, func(path dyn.Path) (dyn.Value, error) {
				// Rewrite the shorthand path ${var.foo} into ${variables.foo.value}.
				if path.HasPrefix(varPath) {
					newPath := dyn.NewPath(
						dyn.Key("variables"),
						path[1],
						dyn.Key("value"),
					)

					if len(path) > 2 {
						newPath = newPath.Append(path[2:]...)
					}

					path = newPath
				}

				// Perform resolution only if the path starts with one of the specified prefixes.
				for _, prefix := range prefixes {
					if path.HasPrefix(prefix) {
						// Skip resolution if there is a skip function and it returns true.
						if m.skipFn != nil && m.skipFn(v) {
							return dyn.InvalidValue, dynvar.ErrSkipResolution
						}
						hasUpdates = true
						return m.lookupFn(normalized, path, b)
					}
				}

				return dyn.InvalidValue, dynvar.ErrSkipResolution
			})
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Normalize the result because variable resolution may have been applied to non-string fields.
		// For example, a variable reference may have been resolved to a integer.
		root, normaliseDiags := convert.Normalize(b.Config, root)
		diags = diags.Extend(normaliseDiags)
		return root, nil
	})
	if err != nil {
		diags = diags.Extend(diag.FromErr(err))
	}

	return hasUpdates, diags
}

// selectivelyMutate applies a function to a subset of the configuration
func (m *resolveVariableReferences) selectivelyMutate(b *bundle.Bundle, fn func(value dyn.Value) (dyn.Value, error)) error {
	return b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		allKeys, err := getAllKeys(root)
		if err != nil {
			return dyn.InvalidValue, err
		}

		var included []string
		for _, key := range allKeys {
			if key == "resources" {
				if m.includeResources {
					included = append(included, key)
				}
			} else {
				included = append(included, key)
			}
		}

		includedRoot, err := merge.Select(root, included)
		if err != nil {
			return dyn.InvalidValue, err
		}

		excludedRoot, err := merge.AntiSelect(root, included)
		if err != nil {
			return dyn.InvalidValue, err
		}

		updatedRoot, err := fn(includedRoot)
		if err != nil {
			return dyn.InvalidValue, err
		}

		// merge is recursive, but it doesn't matter because keys are mutually exclusive
		return merge.Merge(updatedRoot, excludedRoot)
	})
}

func getAllKeys(root dyn.Value) ([]string, error) {
	var keys []string

	if mapping, ok := root.AsMap(); ok {
		for _, key := range mapping.Keys() {
			if keyString, ok := key.AsString(); ok {
				keys = append(keys, keyString)
			} else {
				return nil, fmt.Errorf("key is not a string: %v", key)
			}
		}
	}

	return keys, nil
}
