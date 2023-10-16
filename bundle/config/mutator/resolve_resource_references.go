package mutator

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resolvers"
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
	for _, v := range b.Config.Variables {
		value := v.Value
		parts := strings.Split(*value, separator)
		if len(parts) != 2 {
			continue
		}

		resource, name := parts[0], parts[1]
		resolver, ok := m.resolvers[resource]
		if !ok {
			return fmt.Errorf("unable to resolve resource reference %s, no resovler for %s", *value, resource)
		}

		id, err := resolver(ctx, b, name)
		if err != nil {
			return fmt.Errorf("failed to resolve resource reference %s, err: %w", *value, err)
		}

		v.Replace(id)
	}

	return nil
}

func (*resolveResourceReferences) Name() string {
	return "ResolveResourceReferences"
}
