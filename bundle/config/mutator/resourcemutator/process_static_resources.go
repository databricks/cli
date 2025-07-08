package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
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

	// only YAML resources need to have paths normalized, before normalizing paths
	// we need to resolve variables because they can change path values:
	// - variable can be used a prefix
	// - path can be part of a complex variable value
	bundle.ApplySeqContext(
		ctx,
		b,
		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): resources.* (strings) (resolves variable references to their actual values)
		// Resolves variable references in 'resources' using bundle, workspace, and variables prefixes
		mutator.ResolveVariableReferencesOnlyResources(
			"bundle",
			"workspace",
			"variables",
		),
		mutator.NormalizePaths(),

		// Translate dashboard paths into paths in the workspace file system
		// This must occur after NormalizePaths but before NormalizeAndInitializeResources,
		// since the latter reads dashboard files and requires fully resolved paths.
		mutator.TranslatePathsDashboards(),
	)

	if logdiag.HasError(ctx) {
		return nil
	}

	NormalizeAndInitializeResources(ctx, b, addedResources)
	return nil
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
