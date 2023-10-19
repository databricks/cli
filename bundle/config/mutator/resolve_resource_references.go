package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resolvers"
	"github.com/databricks/cli/libs/log"
)

var separator string = ":"

type resolveResourceReferences struct {
	resolvers map[string](resolvers.ResolverFunc)
}

func ResolveResourceReferences() bundle.Mutator {
	return &resolveResourceReferences{
		resolvers: resolvers.Resolvers(),
	}
}

func (m *resolveResourceReferences) Apply(ctx context.Context, b *bundle.Bundle) error {
	for k, v := range b.Config.Variables {
		if v.Lookup == "" {
			continue
		}

		if v.HasValue() {
			log.Debugf(ctx, "Ignoring '%s' lookup for the variable '%s' because the value is set", v.Lookup, k)
			continue
		}

		lookup := v.Lookup
		parts := strings.Split(lookup, separator)
		if len(parts) != 2 {
			return fmt.Errorf("incorrect lookup specified %s", lookup)
		}

		resource, name := parts[0], parts[1]
		resolver, ok := m.resolvers[resource]
		if !ok {
			return fmt.Errorf("unable to resolve resource reference %s, no resovler for %s", lookup, resource)
		}

		id, err := resolver(ctx, b, name)
		if err != nil {
			return fmt.Errorf("failed to resolve resource reference %s, err: %w", lookup, err)
		}

		v.Set(id)
	}

	return nil
}

func (*resolveResourceReferences) Name() string {
	return "ResolveResourceReferences"
}
