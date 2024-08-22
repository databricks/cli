package terraform

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	tfjson "github.com/hashicorp/terraform-json"
)

func conv(from any, to any) {
	buf, _ := json.Marshal(from)
	json.Unmarshal(buf, &to)
}

func convPermissions(acl []resources.Permission) *schema.ResourcePermissions {
	if len(acl) == 0 {
		return nil
	}

	resource := schema.ResourcePermissions{}
	for _, ac := range acl {
		resource.AccessControl = append(resource.AccessControl, convPermission(ac))
	}

	return &resource
}

func convPermission(ac resources.Permission) schema.ResourcePermissionsAccessControl {
	dst := schema.ResourcePermissionsAccessControl{
		PermissionLevel: ac.Level,
	}
	if ac.UserName != "" {
		dst.UserName = ac.UserName
	}
	if ac.GroupName != "" {
		dst.GroupName = ac.GroupName
	}
	if ac.ServicePrincipalName != "" {
		dst.ServicePrincipalName = ac.ServicePrincipalName
	}
	return dst
}

func convGrants(acl []resources.Grant) *schema.ResourceGrants {
	if len(acl) == 0 {
		return nil
	}

	resource := schema.ResourceGrants{}
	for _, ac := range acl {
		resource.Grant = append(resource.Grant, schema.ResourceGrantsGrant{
			Privileges: ac.Privileges,
			Principal:  ac.Principal,
		})
	}

	return &resource
}

// BundleToTerraform converts resources in a bundle configuration
// to the equivalent Terraform JSON representation.
//
// Note: This function is an older implementation of the conversion logic. It is
// no longer used in any code paths. It is kept around to be used in tests.
// New resources do not need to modify this function and can instead can define
// the conversion login in the tfdyn package.
func BundleToTerraform(config *config.Root) *schema.Root {
	tfroot := schema.NewRoot()
	tfroot.Provider = schema.NewProviders()
	tfroot.Resource = schema.NewResources()
	noResources := true

	for k, src := range config.Resources.Jobs {
		noResources = false
		var dst schema.ResourceJob
		conv(src, &dst)

		if src.JobSettings != nil {
			for _, v := range src.Tasks {
				var t schema.ResourceJobTask
				conv(v, &t)

				for _, v_ := range v.Libraries {
					var l schema.ResourceJobTaskLibrary
					conv(v_, &l)
					t.Library = append(t.Library, l)
				}

				// Convert for_each_task libraries
				if v.ForEachTask != nil {
					for _, v_ := range v.ForEachTask.Task.Libraries {
						var l schema.ResourceJobTaskForEachTaskTaskLibrary
						conv(v_, &l)
						t.ForEachTask.Task.Library = append(t.ForEachTask.Task.Library, l)
					}

				}

				dst.Task = append(dst.Task, t)
			}

			for _, v := range src.JobClusters {
				var t schema.ResourceJobJobCluster
				conv(v, &t)
				dst.JobCluster = append(dst.JobCluster, t)
			}

			// Unblock downstream work. To be addressed more generally later.
			if git := src.GitSource; git != nil {
				dst.GitSource = &schema.ResourceJobGitSource{
					Url:      git.GitUrl,
					Branch:   git.GitBranch,
					Commit:   git.GitCommit,
					Provider: string(git.GitProvider),
					Tag:      git.GitTag,
				}
			}

			for _, v := range src.Parameters {
				var t schema.ResourceJobParameter
				conv(v, &t)
				dst.Parameter = append(dst.Parameter, t)
			}
		}

		tfroot.Resource.Job[k] = &dst

		// Configure permissions for this resource.
		if rp := convPermissions(src.Permissions); rp != nil {
			rp.JobId = fmt.Sprintf("${databricks_job.%s.id}", k)
			tfroot.Resource.Permissions["job_"+k] = rp
		}
	}

	for k, src := range config.Resources.Pipelines {
		noResources = false
		var dst schema.ResourcePipeline
		conv(src, &dst)

		if src.PipelineSpec != nil {
			for _, v := range src.Libraries {
				var l schema.ResourcePipelineLibrary
				conv(v, &l)
				dst.Library = append(dst.Library, l)
			}

			for _, v := range src.Clusters {
				var l schema.ResourcePipelineCluster
				conv(v, &l)
				dst.Cluster = append(dst.Cluster, l)
			}

			for _, v := range src.Notifications {
				var l schema.ResourcePipelineNotification
				conv(v, &l)
				dst.Notification = append(dst.Notification, l)
			}
		}

		tfroot.Resource.Pipeline[k] = &dst

		// Configure permissions for this resource.
		if rp := convPermissions(src.Permissions); rp != nil {
			rp.PipelineId = fmt.Sprintf("${databricks_pipeline.%s.id}", k)
			tfroot.Resource.Permissions["pipeline_"+k] = rp
		}
	}

	for k, src := range config.Resources.Models {
		noResources = false
		var dst schema.ResourceMlflowModel
		conv(src, &dst)
		tfroot.Resource.MlflowModel[k] = &dst

		// Configure permissions for this resource.
		if rp := convPermissions(src.Permissions); rp != nil {
			rp.RegisteredModelId = fmt.Sprintf("${databricks_mlflow_model.%s.registered_model_id}", k)
			tfroot.Resource.Permissions["mlflow_model_"+k] = rp
		}
	}

	for k, src := range config.Resources.Experiments {
		noResources = false
		var dst schema.ResourceMlflowExperiment
		conv(src, &dst)
		tfroot.Resource.MlflowExperiment[k] = &dst

		// Configure permissions for this resource.
		if rp := convPermissions(src.Permissions); rp != nil {
			rp.ExperimentId = fmt.Sprintf("${databricks_mlflow_experiment.%s.id}", k)
			tfroot.Resource.Permissions["mlflow_experiment_"+k] = rp
		}
	}

	for k, src := range config.Resources.ModelServingEndpoints {
		noResources = false
		var dst schema.ResourceModelServing
		conv(src, &dst)
		tfroot.Resource.ModelServing[k] = &dst

		// Configure permissions for this resource.
		if rp := convPermissions(src.Permissions); rp != nil {
			rp.ServingEndpointId = fmt.Sprintf("${databricks_model_serving.%s.serving_endpoint_id}", k)
			tfroot.Resource.Permissions["model_serving_"+k] = rp
		}
	}

	for k, src := range config.Resources.RegisteredModels {
		noResources = false
		var dst schema.ResourceRegisteredModel
		conv(src, &dst)
		tfroot.Resource.RegisteredModel[k] = &dst

		// Configure permissions for this resource.
		if rp := convGrants(src.Grants); rp != nil {
			rp.Function = fmt.Sprintf("${databricks_registered_model.%s.id}", k)
			tfroot.Resource.Grants["registered_model_"+k] = rp
		}
	}

	for k, src := range config.Resources.QualityMonitors {
		noResources = false
		var dst schema.ResourceQualityMonitor
		conv(src, &dst)
		tfroot.Resource.QualityMonitor[k] = &dst
	}

	// We explicitly set "resource" to nil to omit it from a JSON encoding.
	// This is required because the terraform CLI requires >= 1 resources defined
	// if the "resource" property is used in a .tf.json file.
	if noResources {
		tfroot.Resource = nil
	}
	return tfroot
}

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

	return nil
}
