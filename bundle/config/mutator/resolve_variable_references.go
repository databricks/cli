package mutator

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

type resolveVariableReferences struct {
	prefixes []string
	pattern  dyn.Pattern
	lookupFn func(dyn.Value, dyn.Path, *bundle.Bundle) (dyn.Value, error)
	skipFn   func(dyn.Value) bool
}

func ResolveVariableReferences(prefixes ...string) bundle.Mutator {
	return &resolveVariableReferences{prefixes: prefixes, lookupFn: lookup}
}

func ResolveVariableReferencesInLookup() bundle.Mutator {
	return &resolveVariableReferences{prefixes: []string{
		"bundle",
		"workspace",
		"variables",
	}, pattern: dyn.NewPattern(dyn.Key("variables"), dyn.AnyKey(), dyn.Key("lookup")), lookupFn: lookupForVariables}
}

func ResolveVariableReferencesInComplexVariables() bundle.Mutator {
	return &resolveVariableReferences{
		prefixes: []string{
			"bundle",
			"workspace",
			"variables",
		},
		pattern:  dyn.NewPattern(dyn.Key("variables"), dyn.AnyKey(), dyn.Key("value")),
		lookupFn: lookupForComplexVariables,
		skipFn:   skipResolvingInNonComplexVariables,
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

func lookupForComplexVariables(v dyn.Value, path dyn.Path, b *bundle.Bundle) (dyn.Value, error) {
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

	if vv.Type == variable.VariableTypeComplex {
		return dyn.InvalidValue, errors.New("complex variables cannot contain references to another complex variables")
	}

	return lookup(v, path, b)
}

func skipResolvingInNonComplexVariables(v dyn.Value) bool {
	switch v.Kind() {
	case dyn.KindMap, dyn.KindSequence:
		return false
	default:
		return true
	}
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

func (*resolveVariableReferences) Name() string {
	return "ResolveVariableReferences"
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

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
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
	return diags
}
