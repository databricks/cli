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
	GenieSpaces           map[string]*resources.GenieSpace           `json:"genie_spaces,omitempty"`
	Apps                  map[string]*resources.App                  `json:"apps,omitempty"`
	SecretScopes          map[string]*resources.SecretScope          `json:"secret_scopes,omitempty"`
	Alerts                map[string]*resources.Alert                `json:"alerts,omitempty"`
	SqlWarehouses         map[string]*resources.SqlWarehouse         `json:"sql_warehouses,omitempty"`
	DatabaseInstances     map[string]*resources.DatabaseInstance     `json:"database_instances,omitempty"`
	DatabaseCatalogs      map[string]*resources.DatabaseCatalog      `json:"database_catalogs,omitempty"`
	SyncedDatabaseTables  map[string]*resources.SyncedDatabaseTable  `json:"synced_database_tables,omitempty"`
}

type ConfigResource interface {
	// Exists returns true if the resource exists in the workspace configured in
	// the input workspace client.
	Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

	// ResourceDescription returns a struct containing strings describing a resource
	ResourceDescription() resources.ResourceDescription

	// GetName returns the in-product name of the resource.
	GetName() string

	// GetURL returns the URL of the resource.
	GetURL() string

	// InitializeURL initializes the URL field of the resource.
	InitializeURL(baseURL url.URL)
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
	if description.PluralName == "" {
		panic("description of a resource group cannot be empty")
	}

	r := make(map[string]ConfigResource)
	for key, resource := range input {
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
		collectResourceMap(descriptions["genie_spaces"], r.GenieSpaces),
		collectResourceMap(descriptions["volumes"], r.Volumes),
		collectResourceMap(descriptions["apps"], r.Apps),
		collectResourceMap(descriptions["alerts"], r.Alerts),
		collectResourceMap(descriptions["secret_scopes"], r.SecretScopes),
		collectResourceMap(descriptions["sql_warehouses"], r.SqlWarehouses),
		collectResourceMap(descriptions["database_instances"], r.DatabaseInstances),
		collectResourceMap(descriptions["database_catalogs"], r.DatabaseCatalogs),
		collectResourceMap(descriptions["synced_database_tables"], r.SyncedDatabaseTables),
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

	for k := range r.GenieSpaces {
		if k == key {
			found = append(found, r.GenieSpaces[k])
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

	for k := range r.ModelServingEndpoints {
		if k == key {
			found = append(found, r.ModelServingEndpoints[k])
		}
	}

	for k := range r.SecretScopes {
		if k == key {
			found = append(found, r.SecretScopes[k])
		}
	}

	for k := range r.Alerts {
		if k == key {
			found = append(found, r.Alerts[k])
		}
	}

	for k := range r.SqlWarehouses {
		if k == key {
			found = append(found, r.SqlWarehouses[k])
		}
	}

	for k := range r.DatabaseInstances {
		if k == key {
			found = append(found, r.DatabaseInstances[k])
		}
	}

	for k := range r.DatabaseCatalogs {
		if k == key {
			found = append(found, r.DatabaseCatalogs[k])
		}
	}

	for k := range r.SyncedDatabaseTables {
		if k == key {
			found = append(found, r.SyncedDatabaseTables[k])
		}
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no such resource: %s", key)
	}

	if len(found) > 1 {
		keys := make([]string, 0, len(found))
		for _, r := range found {
			keys = append(keys, fmt.Sprintf("%s.%s", r.ResourceDescription().PluralName, key))
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
		"genie_spaces":            (&resources.GenieSpace{}).ResourceDescription(),
		"volumes":                 (&resources.Volume{}).ResourceDescription(),
		"apps":                    (&resources.App{}).ResourceDescription(),
		"secret_scopes":           (&resources.SecretScope{}).ResourceDescription(),
		"alerts":                  (&resources.Alert{}).ResourceDescription(),
		"sql_warehouses":          (&resources.SqlWarehouse{}).ResourceDescription(),
		"database_instances":      (&resources.DatabaseInstance{}).ResourceDescription(),
		"database_catalogs":       (&resources.DatabaseCatalog{}).ResourceDescription(),
		"synced_database_tables":  (&resources.SyncedDatabaseTable{}).ResourceDescription(),
	}
}
