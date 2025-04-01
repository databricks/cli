package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type processYamlResources struct {
	resourceProcessor ResourceProcessor
}

// ProcessYamlResources is a mutator that processes all YAML resources in the bundle.
//
// Pre-condition:
// - Only YAML resources are loaded
//
// Post-condition:
// - ResourceProcessor is applied to all YAML resources
func ProcessYamlResources(resourceProcessor ResourceProcessor) bundle.Mutator {
	return &processYamlResources{resourceProcessor: resourceProcessor}
}

func (p processYamlResources) Name() string {
	return "ProcessYamlResources"
}

func (p processYamlResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	addedResources := NewResourceKeySet()

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		pattern := dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey())
		err := addedResources.AddPattern(pattern, root)
		if err != nil {
			return dyn.InvalidValue, err
		}

		return root, nil
	})
	if err != nil {
		return diag.Errorf("failed to collect resources: %s", err)
	}

	return p.resourceProcessor.Process(ctx, b, ResourceProcessorOpts{AddedResources: addedResources})
}
