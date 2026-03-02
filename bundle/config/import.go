package config

// ImportResource represents a single resource to import with its workspace ID.
type ImportResource struct {
	ID string `json:"id"`
}

// Import defines resources to import at the target level.
// Resources listed here will be bound to the bundle at deploy time.
// This field is only valid for the direct deployment engine.
type Import struct {
	Jobs                  map[string]ImportResource `json:"jobs,omitempty"`
	Pipelines             map[string]ImportResource `json:"pipelines,omitempty"`
	Models                map[string]ImportResource `json:"models,omitempty"`
	Experiments           map[string]ImportResource `json:"experiments,omitempty"`
	ModelServingEndpoints map[string]ImportResource `json:"model_serving_endpoints,omitempty"`
	RegisteredModels      map[string]ImportResource `json:"registered_models,omitempty"`
	QualityMonitors       map[string]ImportResource `json:"quality_monitors,omitempty"`
	Schemas               map[string]ImportResource `json:"schemas,omitempty"`
	Volumes               map[string]ImportResource `json:"volumes,omitempty"`
	Clusters              map[string]ImportResource `json:"clusters,omitempty"`
	Dashboards            map[string]ImportResource `json:"dashboards,omitempty"`
	Apps                  map[string]ImportResource `json:"apps,omitempty"`
	SecretScopes          map[string]ImportResource `json:"secret_scopes,omitempty"`
	Alerts                map[string]ImportResource `json:"alerts,omitempty"`
	SqlWarehouses         map[string]ImportResource `json:"sql_warehouses,omitempty"`
	DatabaseInstances     map[string]ImportResource `json:"database_instances,omitempty"`
	DatabaseCatalogs      map[string]ImportResource `json:"database_catalogs,omitempty"`
	SyncedDatabaseTables  map[string]ImportResource `json:"synced_database_tables,omitempty"`
	PostgresProjects      map[string]ImportResource `json:"postgres_projects,omitempty"`
	PostgresBranches      map[string]ImportResource `json:"postgres_branches,omitempty"`
	PostgresEndpoints     map[string]ImportResource `json:"postgres_endpoints,omitempty"`
}

// GetImportID returns the import ID for a given resource type and name.
// Returns empty string if no import is defined for the resource.
func (i *Import) GetImportID(resourceType, resourceName string) string {
	if i == nil {
		return ""
	}
	switch resourceType {
	case "jobs":
		if r, ok := i.Jobs[resourceName]; ok {
			return r.ID
		}
	case "pipelines":
		if r, ok := i.Pipelines[resourceName]; ok {
			return r.ID
		}
	case "models":
		if r, ok := i.Models[resourceName]; ok {
			return r.ID
		}
	case "experiments":
		if r, ok := i.Experiments[resourceName]; ok {
			return r.ID
		}
	case "model_serving_endpoints":
		if r, ok := i.ModelServingEndpoints[resourceName]; ok {
			return r.ID
		}
	case "registered_models":
		if r, ok := i.RegisteredModels[resourceName]; ok {
			return r.ID
		}
	case "quality_monitors":
		if r, ok := i.QualityMonitors[resourceName]; ok {
			return r.ID
		}
	case "schemas":
		if r, ok := i.Schemas[resourceName]; ok {
			return r.ID
		}
	case "volumes":
		if r, ok := i.Volumes[resourceName]; ok {
			return r.ID
		}
	case "clusters":
		if r, ok := i.Clusters[resourceName]; ok {
			return r.ID
		}
	case "dashboards":
		if r, ok := i.Dashboards[resourceName]; ok {
			return r.ID
		}
	case "apps":
		if r, ok := i.Apps[resourceName]; ok {
			return r.ID
		}
	case "secret_scopes":
		if r, ok := i.SecretScopes[resourceName]; ok {
			return r.ID
		}
	case "alerts":
		if r, ok := i.Alerts[resourceName]; ok {
			return r.ID
		}
	case "sql_warehouses":
		if r, ok := i.SqlWarehouses[resourceName]; ok {
			return r.ID
		}
	case "database_instances":
		if r, ok := i.DatabaseInstances[resourceName]; ok {
			return r.ID
		}
	case "database_catalogs":
		if r, ok := i.DatabaseCatalogs[resourceName]; ok {
			return r.ID
		}
	case "synced_database_tables":
		if r, ok := i.SyncedDatabaseTables[resourceName]; ok {
			return r.ID
		}
	case "postgres_projects":
		if r, ok := i.PostgresProjects[resourceName]; ok {
			return r.ID
		}
	case "postgres_branches":
		if r, ok := i.PostgresBranches[resourceName]; ok {
			return r.ID
		}
	case "postgres_endpoints":
		if r, ok := i.PostgresEndpoints[resourceName]; ok {
			return r.ID
		}
	}
	return ""
}

// ForEach calls fn for each import entry in the configuration.
func (i *Import) ForEach(fn func(resourceType, resourceName, importID string)) {
	if i == nil {
		return
	}
	for name, r := range i.Jobs {
		fn("jobs", name, r.ID)
	}
	for name, r := range i.Pipelines {
		fn("pipelines", name, r.ID)
	}
	for name, r := range i.Models {
		fn("models", name, r.ID)
	}
	for name, r := range i.Experiments {
		fn("experiments", name, r.ID)
	}
	for name, r := range i.ModelServingEndpoints {
		fn("model_serving_endpoints", name, r.ID)
	}
	for name, r := range i.RegisteredModels {
		fn("registered_models", name, r.ID)
	}
	for name, r := range i.QualityMonitors {
		fn("quality_monitors", name, r.ID)
	}
	for name, r := range i.Schemas {
		fn("schemas", name, r.ID)
	}
	for name, r := range i.Volumes {
		fn("volumes", name, r.ID)
	}
	for name, r := range i.Clusters {
		fn("clusters", name, r.ID)
	}
	for name, r := range i.Dashboards {
		fn("dashboards", name, r.ID)
	}
	for name, r := range i.Apps {
		fn("apps", name, r.ID)
	}
	for name, r := range i.SecretScopes {
		fn("secret_scopes", name, r.ID)
	}
	for name, r := range i.Alerts {
		fn("alerts", name, r.ID)
	}
	for name, r := range i.SqlWarehouses {
		fn("sql_warehouses", name, r.ID)
	}
	for name, r := range i.DatabaseInstances {
		fn("database_instances", name, r.ID)
	}
	for name, r := range i.DatabaseCatalogs {
		fn("database_catalogs", name, r.ID)
	}
	for name, r := range i.SyncedDatabaseTables {
		fn("synced_database_tables", name, r.ID)
	}
	for name, r := range i.PostgresProjects {
		fn("postgres_projects", name, r.ID)
	}
	for name, r := range i.PostgresBranches {
		fn("postgres_branches", name, r.ID)
	}
	for name, r := range i.PostgresEndpoints {
		fn("postgres_endpoints", name, r.ID)
	}
}

// IsEmpty returns true if no imports are defined.
func (i *Import) IsEmpty() bool {
	if i == nil {
		return true
	}
	return len(i.Jobs) == 0 &&
		len(i.Pipelines) == 0 &&
		len(i.Models) == 0 &&
		len(i.Experiments) == 0 &&
		len(i.ModelServingEndpoints) == 0 &&
		len(i.RegisteredModels) == 0 &&
		len(i.QualityMonitors) == 0 &&
		len(i.Schemas) == 0 &&
		len(i.Volumes) == 0 &&
		len(i.Clusters) == 0 &&
		len(i.Dashboards) == 0 &&
		len(i.Apps) == 0 &&
		len(i.SecretScopes) == 0 &&
		len(i.Alerts) == 0 &&
		len(i.SqlWarehouses) == 0 &&
		len(i.DatabaseInstances) == 0 &&
		len(i.DatabaseCatalogs) == 0 &&
		len(i.SyncedDatabaseTables) == 0 &&
		len(i.PostgresProjects) == 0 &&
		len(i.PostgresBranches) == 0 &&
		len(i.PostgresEndpoints) == 0
}
