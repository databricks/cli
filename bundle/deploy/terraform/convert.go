package terraform

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
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

// BundleToTerraform converts resources in a bundle configuration
// to the equivalent Terraform JSON representation.
//
// NOTE: THIS IS CURRENTLY A HACK. WE NEED A BETTER WAY TO
// CONVERT TO/FROM TERRAFORM COMPATIBLE FORMAT.
func BundleToTerraform(config *config.Root) (*schema.Root, bool) {
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

	return tfroot, noResources
}

func TerraformToBundle(state *tfjson.State, config *config.Root) error {
	for _, resource := range state.Values.RootModule.Resources {
		// Limit to resources.
		if resource.Mode != tfjson.ManagedResourceMode {
			continue
		}

		switch resource.Type {
		case "databricks_job":
			var tmp schema.ResourceJob
			conv(resource.AttributeValues, &tmp)
			cur := config.Resources.Jobs[resource.Name]
			conv(tmp, &cur)
			config.Resources.Jobs[resource.Name] = cur
		case "databricks_pipeline":
			var tmp schema.ResourcePipeline
			conv(resource.AttributeValues, &tmp)
			cur := config.Resources.Pipelines[resource.Name]
			conv(tmp, &cur)
			config.Resources.Pipelines[resource.Name] = cur
		case "databricks_mlflow_model":
			var tmp schema.ResourceMlflowModel
			conv(resource.AttributeValues, &tmp)
			cur := config.Resources.Models[resource.Name]
			conv(tmp, &cur)
			config.Resources.Models[resource.Name] = cur
		case "databricks_mlflow_experiment":
			var tmp schema.ResourceMlflowExperiment
			conv(resource.AttributeValues, &tmp)
			cur := config.Resources.Experiments[resource.Name]
			conv(tmp, &cur)
			config.Resources.Experiments[resource.Name] = cur
		case "databricks_permissions":
			// Ignore; no need to pull these back into the configuration.
		default:
			return fmt.Errorf("missing mapping for %s", resource.Type)
		}
	}

	return nil
}
