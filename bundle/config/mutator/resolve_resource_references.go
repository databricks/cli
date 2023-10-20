package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resolvers"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/sync/errgroup"
)

const separator string = ":"

type resolveResourceReferences struct {
	resolvers map[string]resolvers.ResolverFunc
}

func ResolveResourceReferences() bundle.Mutator {
	return &resolveResourceReferences{
		resolvers: resolvers.Resolvers(),
	}
}

func (m *resolveResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) error {
	errs, errCtx := errgroup.WithContext(ctx)

	for k := range b.Config.Variables {
		v := b.Config.Variables[k]
		if v.Lookup == "" {
			continue
		}

		if v.HasValue() {
			log.Debugf(ctx, "Ignoring '%s' lookup for the variable '%s' because the value is set", v.Lookup, k)
			continue
		}

		lookup := v.Lookup
		resource, name, ok := strings.Cut(lookup, separator)
		if !ok {
			return fmt.Errorf("incorrect lookup specified %s", lookup)
		}

		resolver, ok := m.resolvers[resource]
		if !ok {
			return fmt.Errorf("unable to resolve resource reference %s, no resolvers for %s", lookup, resource)
		}

		errs.Go(func() error {
			id, err := resolver(errCtx, b, name)
			if err != nil {
				return fmt.Errorf("failed to resolve %s reference %s, err: %w", resource, lookup, err)
			}

			v.Set(id)
			return nil
		})
	}

	return errs.Wait()
}

func (*resolveResourceReferences) Name() string {
	return "ResolveResourceReferences"
}
