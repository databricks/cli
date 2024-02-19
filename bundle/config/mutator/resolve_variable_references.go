package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
)

type resolveVariableReferences struct {
	prefixes []string
}

func ResolveVariableReferences(prefixes ...string) bundle.Mutator {
	return &resolveVariableReferences{prefixes: prefixes}
}

func (*resolveVariableReferences) Name() string {
	return "ResolveVariableReferences"
}

func (m *resolveVariableReferences) Validate(ctx context.Context, b *bundle.Bundle) error {
	return nil
}

func (m *resolveVariableReferences) Apply(ctx context.Context, b *bundle.Bundle) error {
	prefixes := make([]dyn.Path, len(m.prefixes))
	for i, prefix := range m.prefixes {
		prefixes[i] = dyn.MustPathFromString(prefix)
	}

	// The path ${var.foo} is a shorthand for ${variables.foo.value}.
	// We rewrite it here to make the resolution logic simpler.
	varPath := dyn.NewPath(dyn.Key("var"))

	return b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
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
		// We can ignore the diagnostics return valuebecause we know that the dynamic value
		// has already been normalized when it was first loaded from the configuration file.
		//
		normalized, _ := convert.Normalize(b.Config, root, convert.IncludeMissingFields)
		lookup := func(path dyn.Path) (dyn.Value, error) {
			// Future opportunity: if we lookup this path in both the given root
			// and the synthesized root, we know if it was explicitly set or implied to be empty.
			// Then we can emit a warning if it was not explicitly set.
			return dyn.GetByPath(normalized, path)
		}

		// Resolve variable references in all values.
		root, err := dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
			// Rewrite the shorthand path ${var.foo} into ${variables.foo.value}.
			if path.HasPrefix(varPath) && len(path) == 2 {
				path = dyn.NewPath(
					dyn.Key("variables"),
					path[1],
					dyn.Key("value"),
				)
			}

			// Perform resolution only if the path starts with one of the specified prefixes.
			for _, prefix := range prefixes {
				if path.HasPrefix(prefix) {
					return lookup(path)
				}
			}

			return dyn.InvalidValue, dynvar.ErrSkipResolution
		})
		if err != nil {
			return dyn.InvalidValue, err
		}

		// Normalize the result because variable resolution may have been applied to non-string fields.
		// For example, a variable reference may have been resolved to a integer.
		root, diags := convert.Normalize(b.Config, root)
		for _, diag := range diags {
			// This occurs when a variable's resolved value is incompatible with the field's type.
			// Log a warning until we have a better way to surface these diagnostics to the user.
			log.Warnf(ctx, "normalization diagnostic: %s", diag.Summary)
		}
		return root, nil
	})
}
