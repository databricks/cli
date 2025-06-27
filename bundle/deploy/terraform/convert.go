package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
)

// BundleToTerraformWithDynValue converts resources in a bundle configuration
// to the equivalent Terraform JSON representation.
func BundleToTerraformWithDynValue(ctx context.Context, root dyn.Value) (*schema.Root, error) {
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
			return dyn.InvalidValue, fmt.Errorf("no converter for resource type %s", typ)
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

func TerraformToBundle(ctx context.Context, state ExportedResourcesMap, config *config.Root) error {
	return config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		for groupName, group := range state {
			for resourceName, attrs := range group {
				path := dyn.Path{dyn.Key("resources"), dyn.Key(groupName), dyn.Key(resourceName)}
				resource, err := dyn.GetByPath(v, path)
				if !resource.IsValid() {
					m := dyn.NewMapping()
					m.SetLoc("id", nil, dyn.V(attrs.ID))
					m.SetLoc("modified_status", nil, dyn.V(resources.ModifiedStatusDeleted))
					v, err = dyn.SetByPath(v, path, dyn.V(m))
					if err != nil {
						return dyn.InvalidValue, err
					}
				} else if err != nil {
					return dyn.InvalidValue, err
				} else {
					v, err = dyn.SetByPath(v, dyn.Path{dyn.Key("resources"), dyn.Key(groupName), dyn.Key(resourceName), dyn.Key("id")}, dyn.V(attrs.ID))
					if err != nil {
						return dyn.InvalidValue, err
					}
				}
			}
		}

		return dyn.MapByPattern(v, dyn.Pattern{dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()}, func(p dyn.Path, inner dyn.Value) (dyn.Value, error) {
			idPath := dyn.Path{dyn.Key("id")}
			statusPath := dyn.Path{dyn.Key("modified_status")}
			id, _ := dyn.GetByPath(inner, idPath)
			status, _ := dyn.GetByPath(inner, statusPath)
			if !id.IsValid() && !status.IsValid() {
				return dyn.SetByPath(inner, statusPath, dyn.V(resources.ModifiedStatusCreated))
			}
			return inner, nil
		})
	})
}
