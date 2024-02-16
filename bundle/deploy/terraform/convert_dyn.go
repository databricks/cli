package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

func BundleToTerraformDyn(ctx context.Context, root dyn.Value) (*schema.Root, error) {
	tfroot := schema.NewRoot()
	tfroot.Provider = schema.NewProviders()
	tfroot.Resource = schema.NewResources()
	numResources := 0

	// Convert each resource in the bundle to the equivalent Terraform representation.
	resources, err := dyn.Get(root, "resources")
	if err != nil {
		// TODO check if key not exists, same condition as no resources.
		return nil, err
	}

	_, err = dyn.Walk(resources, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) < 2 {
			return v, nil
		}

		typ := p[0].Key()
		key := p[1].Key()

		// Lookup the converter based on the resource type.
		c, ok := tfdyn.GetConverter(typ)
		if !ok {
			return dyn.InvalidValue, fmt.Errorf("no converter for resource type %s", typ)
		}

		// Convert resource to Terraform representation.
		err := c.Convert(ctx, key, v, tfroot.Resource)
		if err != nil {
			return dyn.InvalidValue, err
		}

		numResources++

		// Skip traversal of the resource itself.
		return v, dyn.ErrSkip
	})
	if err != nil {
		return nil, err
	}

	// We explicitly set "resource" to nil to omit it from a JSON encoding.
	// This is required because the terraform CLI requires >= 1 resources defined
	// if the "resource" property is used in a .tf.json file.
	if numResources == 0 {
		tfroot.Resource = nil
	}

	return tfroot, nil
}
