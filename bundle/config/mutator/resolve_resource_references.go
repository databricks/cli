package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

type resolveResourceReferences struct{}

func ResolveResourceReferences() bundle.Mutator {
	return &resolveResourceReferences{}
}

func (m *resolveResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	errs, errCtx := errgroup.WithContext(ctx)

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

			return v.Set(id)
		})
	}

	// Note, diags are lost from all goroutines except the first one to return diag
	return diag.FromErr(errs.Wait())
}

func (*resolveResourceReferences) Name() string {
	return "ResolveResourceReferences"
}
