package config

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
)

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Jobs      map[string]*resources.Job      `json:"jobs,omitempty"`
	Pipelines map[string]*resources.Pipeline `json:"pipelines,omitempty"`

	Models                map[string]*resources.MlflowModel          `json:"models,omitempty"`
	Experiments           map[string]*resources.MlflowExperiment     `json:"experiments,omitempty"`
	ModelServingEndpoints map[string]*resources.ModelServingEndpoint `json:"model_serving_endpoints,omitempty"`
	RegisteredModels      map[string]*resources.RegisteredModel      `json:"registered_models,omitempty"`
	QualityMonitors       map[string]*resources.QualityMonitor       `json:"quality_monitors,omitempty"`
	Schemas               map[string]*resources.Schema               `json:"schemas,omitempty"`
}

type resource struct {
	resource      ConfigResource
	resource_type string
	key           string
}

// TODO: Add UC schema resource here.
// TODO: Make this general and not specfix to specific resources?
func (r *Resources) allResources() []resource {
	all := make([]resource, 0)
	for k, e := range r.Jobs {
		all = append(all, resource{resource_type: "job", resource: e, key: k})
	}
	for k, e := range r.Pipelines {
		all = append(all, resource{resource_type: "pipeline", resource: e, key: k})
	}
	for k, e := range r.Models {
		all = append(all, resource{resource_type: "model", resource: e, key: k})
	}
	for k, e := range r.Experiments {
		all = append(all, resource{resource_type: "experiment", resource: e, key: k})
	}
	for k, e := range r.ModelServingEndpoints {
		all = append(all, resource{resource_type: "serving endpoint", resource: e, key: k})
	}
	for k, e := range r.RegisteredModels {
		all = append(all, resource{resource_type: "registered model", resource: e, key: k})
	}
	for k, e := range r.QualityMonitors {
		all = append(all, resource{resource_type: "quality monitor", resource: e, key: k})
	}
	return all
}

func (r *Resources) VerifyAllResourcesDefined() error {
	all := r.allResources()
	for _, e := range all {
		err := e.resource.Validate()
		if err != nil {
			return fmt.Errorf("%s %s is not defined", e.resource_type, e.key)
		}
	}

	return nil
}

type ConfigResource interface {
	// Function to assert if the resource exists in the workspace configured in
	// the input workspace client.
	Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

	// Terraform equivalent name of the resource. For example "databricks_job"
	// for jobs and "databricks_pipeline" for pipelines.
	TerraformResourceName() string

	// Validate the resource configuration.
	Validate() error
}

func (r *Resources) FindResourceByConfigKey(key string) (ConfigResource, error) {
	found := make([]ConfigResource, 0)
	for k := range r.Jobs {
		if k == key {
			found = append(found, r.Jobs[k])
		}
	}
	for k := range r.Pipelines {
		if k == key {
			found = append(found, r.Pipelines[k])
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no such resource: %s", key)
	}

	if len(found) > 1 {
		keys := make([]string, 0, len(found))
		for _, r := range found {
			keys = append(keys, fmt.Sprintf("%s:%s", r.TerraformResourceName(), key))
		}
		return nil, fmt.Errorf("ambiguous: %s (can resolve to all of %s)", key, keys)
	}

	return found[0], nil
}
