package terraform

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
)

// // Rewrite variable references to resources into Terraform compatible format.
// func interpolateTerraformResourceIdentifiers(path string, lookup map[string]string) (string, error) {
// 	parts := strings.Split(path, interpolation.Delimiter)
// 	if parts[0] == "resources" {
// 		switch parts[1] {
// 		case "pipelines":
// 			path = strings.Join(append([]string{"databricks_pipeline"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		case "jobs":
// 			path = strings.Join(append([]string{"databricks_job"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		case "models":
// 			path = strings.Join(append([]string{"databricks_mlflow_model"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		case "experiments":
// 			path = strings.Join(append([]string{"databricks_mlflow_experiment"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		case "model_serving_endpoints":
// 			path = strings.Join(append([]string{"databricks_model_serving"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		case "registered_models":
// 			path = strings.Join(append([]string{"databricks_registered_model"}, parts[2:]...), interpolation.Delimiter)
// 			return fmt.Sprintf("${%s}", path), nil
// 		default:
// 			panic("TODO: " + parts[1])
// 		}
// 	}

// 	return interpolation.DefaultLookup(path, lookup)
// }

func Interpolate() bundle.Mutator {
	return mutator.ResolveVariableReferences("foobar")
	// return interpolation.Interpolate(interpolateTerraformResourceIdentifiers)
}
