package metadata

import (
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// reverseInterpolate reverses the terraform.Interpolate transformation.
// It converts terraform-style resource references back to bundle-style.
// Example: ${databricks_pipeline.my_etl.id} → ${resources.pipelines.my_etl.id}
func reverseInterpolate(root dyn.Value) (dyn.Value, error) {
	return dynvar.Resolve(root, func(path dyn.Path) (dyn.Value, error) {
		// Need at least 2 components: resource_type.resource_name
		if len(path) < 2 {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		resourceType := path[0].Key()
		isAlreadyBundleFormat := resourceType == "resources"
		if isAlreadyBundleFormat {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		bundleGroup, ok := terraform.TerraformToGroupName[resourceType]
		if !ok {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}

		// Reconstruct path in bundle format:
		// databricks_pipeline.my_pipeline.id → resources.pipelines.my_pipeline.id
		bundlePath := dyn.NewPath(dyn.Key("resources"), dyn.Key(bundleGroup)).Append(path[1:]...)
		return dyn.V(fmt.Sprintf("${%s}", bundlePath.String())), nil
	})
}
