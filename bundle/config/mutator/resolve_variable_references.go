package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
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
		lookup := dynvar.DefaultLookup(root)

		// Resolve variable references in all values.
		return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
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
	})
}
