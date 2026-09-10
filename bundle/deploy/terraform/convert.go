package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
)

// BundleToTerraformWithDynValue converts resources in a bundle configuration
// to the equivalent Terraform JSON representation.
//
// Resource types without a terraform converter are only supported by the direct
// engine. They are an error here, except when skipUnsupported is set: the state is
// migrated to the direct engine right after this deploy, so they are skipped by
// this run and deployed by the next one. See bundle.Bundle.MigratingToDirect.
func BundleToTerraformWithDynValue(ctx context.Context, root dyn.Value, skipUnsupported bool) (*schema.Root, error) {
	tfroot := schema.NewRoot()
	tfroot.Provider = schema.NewProviders()

	// Convert each resource in the bundle to the equivalent Terraform representation.
	dynResources, err := dyn.Get(root, "resources")
	if err != nil {
		// If the resources key is missing, return an empty root.
		if dyn.IsNoSuchKeyError(err) {
			return tfroot, nil
		}
		return nil, err
	}

	tfroot.Resource = schema.NewResources()

	numResources := 0
	_, err = dyn.Walk(dynResources, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if len(p) < 2 {
			return v, nil
		}

		// Skip resources that have been deleted locally.
		modifiedStatus, err := dyn.Get(v, "modified_status")
		if err == nil {
			modifiedStatusStr, ok := modifiedStatus.AsString()
			if ok && modifiedStatusStr == resources.ModifiedStatusDeleted {
				return v, dyn.ErrSkip
			}
		}

		typ := p[0].Key()
		key := p[1].Key()

		// Lookup the converter based on the resource type.
		c, ok := tfdyn.GetConverter(typ)
		if !ok {
			if !skipUnsupported {
				return dyn.InvalidValue, fmt.Errorf("no converter for resource type %s", typ)
			}
			log.Infof(ctx, "%s.%s: skipping, only supported by the direct engine; will be deployed after the state is migrated", typ, key)
			return v, dyn.ErrSkip
		}

		// Convert resource to Terraform representation.
		err = c.Convert(ctx, key, v, tfroot.Resource)
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
