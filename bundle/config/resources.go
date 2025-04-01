package config

import (
	"context"
	"fmt"
	"net/url"

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
	Volumes               map[string]*resources.Volume               `json:"volumes,omitempty"`
	Clusters              map[string]*resources.Cluster              `json:"clusters,omitempty"`
	Dashboards            map[string]*resources.Dashboard            `json:"dashboards,omitempty"`
	Apps                  map[string]*resources.App                  `json:"apps,omitempty"`
}

type ConfigResource interface {
	// Exists returns true if the resource exists in the workspace configured in
	// the input workspace client.
	Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

	// ResourceDescription returns a struct containing strings describing a resource
	ResourceDescription() resources.ResourceDescription

	// TerraformResourceName returns an equivalent name of the resource. For example "databricks_job"
	// for jobs and "databricks_pipeline" for pipelines.
	TerraformResourceName() string

	// GetName returns the in-product name of the resource.
	GetName() string

	// GetURL returns the URL of the resource.
	GetURL() string

	// InitializeURL initializes the URL field of the resource.
	InitializeURL(baseURL url.URL)

	// IsNil returns true if the resource is nil, for example, when it was removed from the bundle.
	IsNil() bool
}

// ResourceGroup represents a group of resources of the same type.
// It includes a description of the resource type and a map of resources.
type ResourceGroup struct {
	Description resources.ResourceDescription
	Resources   map[string]ConfigResource
}

// collectResourceMap collects resources of a specific type into a ResourceGroup.
func collectResourceMap[T ConfigResource](
	description resources.ResourceDescription,
	input map[string]T,
) ResourceGroup {
	r := make(map[string]ConfigResource)
	for key, resource := range input {
		if resource.IsNil() {
			continue
		}
		r[key] = resource
	}
	return ResourceGroup{
		Description: description,
		Resources:   r,
	}
}

// AllResources returns all resources in the bundle grouped by their resource type.
func (r *Resources) AllResources() []ResourceGroup {
	descriptions := SupportedResources()
	return []ResourceGroup{
		collectResourceMap(descriptions["jobs"], r.Jobs),
		collectResourceMap(descriptions["pipelines"], r.Pipelines),
		collectResourceMap(descriptions["models"], r.Models),
		collectResourceMap(descriptions["experiments"], r.Experiments),
		collectResourceMap(descriptions["model_serving_endpoints"], r.ModelServingEndpoints),
		collectResourceMap(descriptions["registered_models"], r.RegisteredModels),
		collectResourceMap(descriptions["quality_monitors"], r.QualityMonitors),
		collectResourceMap(descriptions["schemas"], r.Schemas),
		collectResourceMap(descriptions["clusters"], r.Clusters),
		collectResourceMap(descriptions["dashboards"], r.Dashboards),
		collectResourceMap(descriptions["volumes"], r.Volumes),
		collectResourceMap(descriptions["apps"], r.Apps),
	}
}

func (r *Resources) FindResourceByConfigKey(key string) (ConfigResource, error) {
	var found []ConfigResource
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

	for k := range r.Apps {
		if k == key {
			found = append(found, r.Apps[k])
		}
	}

	for k := range r.Schemas {
		if k == key {
			found = append(found, r.Schemas[k])
		}
	}

	for k := range r.Experiments {
		if k == key {
			found = append(found, r.Experiments[k])
		}
	}

	for k := range r.Clusters {
		if k == key {
			found = append(found, r.Clusters[k])
		}
	}

	for k := range r.Volumes {
		if k == key {
			found = append(found, r.Volumes[k])
		}
	}

	for k := range r.Dashboards {
		if k == key {
			found = append(found, r.Dashboards[k])
		}
	}

	for k := range r.RegisteredModels {
		if k == key {
			found = append(found, r.RegisteredModels[k])
		}
	}

	for k := range r.QualityMonitors {
		if k == key {
			found = append(found, r.QualityMonitors[k])
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

// SupportedResources returns a map which keys correspond to the resource key in the bundle configuration.
func SupportedResources() map[string]resources.ResourceDescription {
	return map[string]resources.ResourceDescription{
		"jobs":                    (&resources.Job{}).ResourceDescription(),
		"pipelines":               (&resources.Pipeline{}).ResourceDescription(),
		"models":                  (&resources.MlflowModel{}).ResourceDescription(),
		"experiments":             (&resources.MlflowExperiment{}).ResourceDescription(),
		"model_serving_endpoints": (&resources.ModelServingEndpoint{}).ResourceDescription(),
		"registered_models":       (&resources.RegisteredModel{}).ResourceDescription(),
		"quality_monitors":        (&resources.QualityMonitor{}).ResourceDescription(),
		"schemas":                 (&resources.Schema{}).ResourceDescription(),
		"clusters":                (&resources.Cluster{}).ResourceDescription(),
		"dashboards":              (&resources.Dashboard{}).ResourceDescription(),
		"volumes":                 (&resources.Volume{}).ResourceDescription(),
		"apps":                    (&resources.App{}).ResourceDescription(),
	}
}
