package terraform

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
	tfjson "github.com/hashicorp/terraform-json"
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

func TerraformToBundle(state *resourcesState, config *config.Root) error {
	for _, resource := range state.Resources {
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}
		for _, instance := range resource.Instances {
			switch resource.Type {
			case "databricks_job":
				if config.Resources.Jobs == nil {
					config.Resources.Jobs = make(map[string]*resources.Job)
				}
				cur := config.Resources.Jobs[resource.Name]
				if cur == nil {
					cur = &resources.Job{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Jobs[resource.Name] = cur
			case "databricks_pipeline":
				if config.Resources.Pipelines == nil {
					config.Resources.Pipelines = make(map[string]*resources.Pipeline)
				}
				cur := config.Resources.Pipelines[resource.Name]
				if cur == nil {
					cur = &resources.Pipeline{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Pipelines[resource.Name] = cur
			case "databricks_mlflow_model":
				if config.Resources.Models == nil {
					config.Resources.Models = make(map[string]*resources.MlflowModel)
				}
				cur := config.Resources.Models[resource.Name]
				if cur == nil {
					cur = &resources.MlflowModel{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Models[resource.Name] = cur
			case "databricks_mlflow_experiment":
				if config.Resources.Experiments == nil {
					config.Resources.Experiments = make(map[string]*resources.MlflowExperiment)
				}
				cur := config.Resources.Experiments[resource.Name]
				if cur == nil {
					cur = &resources.MlflowExperiment{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Experiments[resource.Name] = cur
			case "databricks_model_serving":
				if config.Resources.ModelServingEndpoints == nil {
					config.Resources.ModelServingEndpoints = make(map[string]*resources.ModelServingEndpoint)
				}
				cur := config.Resources.ModelServingEndpoints[resource.Name]
				if cur == nil {
					cur = &resources.ModelServingEndpoint{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.ModelServingEndpoints[resource.Name] = cur
			case "databricks_registered_model":
				if config.Resources.RegisteredModels == nil {
					config.Resources.RegisteredModels = make(map[string]*resources.RegisteredModel)
				}
				cur := config.Resources.RegisteredModels[resource.Name]
				if cur == nil {
					cur = &resources.RegisteredModel{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.RegisteredModels[resource.Name] = cur
			case "databricks_quality_monitor":
				if config.Resources.QualityMonitors == nil {
					config.Resources.QualityMonitors = make(map[string]*resources.QualityMonitor)
				}
				cur := config.Resources.QualityMonitors[resource.Name]
				if cur == nil {
					cur = &resources.QualityMonitor{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.QualityMonitors[resource.Name] = cur
			case "databricks_schema":
				if config.Resources.Schemas == nil {
					config.Resources.Schemas = make(map[string]*resources.Schema)
				}
				cur := config.Resources.Schemas[resource.Name]
				if cur == nil {
					cur = &resources.Schema{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Schemas[resource.Name] = cur
			case "databricks_volume":
				if config.Resources.Volumes == nil {
					config.Resources.Volumes = make(map[string]*resources.Volume)
				}
				cur := config.Resources.Volumes[resource.Name]
				if cur == nil {
					cur = &resources.Volume{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Volumes[resource.Name] = cur
			case "databricks_cluster":
				if config.Resources.Clusters == nil {
					config.Resources.Clusters = make(map[string]*resources.Cluster)
				}
				cur := config.Resources.Clusters[resource.Name]
				if cur == nil {
					cur = &resources.Cluster{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Clusters[resource.Name] = cur
			case "databricks_dashboard":
				if config.Resources.Dashboards == nil {
					config.Resources.Dashboards = make(map[string]*resources.Dashboard)
				}
				cur := config.Resources.Dashboards[resource.Name]
				if cur == nil {
					cur = &resources.Dashboard{ModifiedStatus: resources.ModifiedStatusDeleted}
				}
				cur.ID = instance.Attributes.ID
				config.Resources.Dashboards[resource.Name] = cur
			case "databricks_app":
				if config.Resources.Apps == nil {
					config.Resources.Apps = make(map[string]*resources.App)
				}
				cur := config.Resources.Apps[resource.Name]
				if cur == nil {
					cur = &resources.App{ModifiedStatus: resources.ModifiedStatusDeleted, App: &apps.App{}}
				} else {
					// If the app exists in terraform and bundle, we always set modified status to updated
					// because we don't really know if the app source code was updated or not.
					cur.ModifiedStatus = resources.ModifiedStatusUpdated
				}
				cur.Name = instance.Attributes.Name
				config.Resources.Apps[resource.Name] = cur
			case "databricks_permissions":
			case "databricks_grants":
				// Ignore; no need to pull these back into the configuration.
			default:
				return fmt.Errorf("missing mapping for %s", resource.Type)
			}
		}
	}

	for _, src := range config.Resources.Jobs {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Pipelines {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Models {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Experiments {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.ModelServingEndpoints {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.RegisteredModels {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.QualityMonitors {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Schemas {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Volumes {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Clusters {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Dashboards {
		if src.ModifiedStatus == "" && src.ID == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}
	for _, src := range config.Resources.Apps {
		if src.ModifiedStatus == "" {
			src.ModifiedStatus = resources.ModifiedStatusCreated
		}
	}

	return nil
}
