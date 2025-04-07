package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type processStaticResources struct{}

// ProcessStaticResources is a mutator that processes all YAML resources in the bundle.
//
// Pre-condition:
// - Only static resources are loaded (e.g. YAML)
//
// Post-condition:
// - All static resources are initialized and normalized
func ProcessStaticResources() bundle.Mutator {
	return &processStaticResources{}
}

func (p processStaticResources) Name() string {
	return "ProcessStaticResources"
}

func (p processStaticResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	addedResources, err := getAllResources(b)
	if err != nil {
		return diag.FromErr(err)
	}

	return NormalizeAndInitializeResources(ctx, b, addedResources)
}

func getAllResources(b *bundle.Bundle) (ResourceKeySet, error) {
	set := NewResourceKeySet()
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
		err := set.AddPattern(pattern, root)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return root, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to collect resources: %s", err)
	}

	return set, nil
}
