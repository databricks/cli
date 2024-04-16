package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynvar"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

type resolveResourceReferences struct{}

func ResolveResourceReferences() bundle.Mutator {
	return &resolveResourceReferences{}
}

func (m *resolveResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	errs, errCtx := errgroup.WithContext(ctx)
	varPath := dyn.NewPath(dyn.Key("var"))
	p := dyn.NewPattern(
		dyn.Key("variables"),
		dyn.AnyKey(),
		dyn.Key("lookup"),
	)

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		normalized, _ := convert.Normalize(b.Config, v, convert.IncludeMissingFields)

		// Resolve all variable references in the lookup field.
		return dyn.MapByPattern(v, p, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			return dynvar.Resolve(v, func(path dyn.Path) (dyn.Value, error) {
				// Rewrite the shorthand path ${var.foo} into ${variables.foo.value}.
				if path.HasPrefix(varPath) && len(path) == 2 {
					path = dyn.NewPath(
						dyn.Key("variables"),
						path[1],
						dyn.Key("value"),
					)
				}

				return dyn.GetByPath(normalized, path)
			})
		})
	})

	if err != nil {
		return diag.FromErr(err)
	}

	for k := range b.Config.Variables {
		v := b.Config.Variables[k]
		if v == nil || v.Lookup == nil {
			continue
		}

		if v.HasValue() {
			log.Debugf(ctx, "Ignoring '%s' lookup for the variable '%s' because the value is set", v.Lookup, k)
			continue
		}

		errs.Go(func() error {
			id, err := v.Lookup.Resolve(errCtx, b.WorkspaceClient())
			if err != nil {
				return fmt.Errorf("failed to resolve %s, err: %w", v.Lookup, err)
			}

			v.Set(id)
			return nil
		})
	}

	return diag.FromErr(errs.Wait())
}

func (*resolveResourceReferences) Name() string {
	return "ResolveResourceReferences"
}
