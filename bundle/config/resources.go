package config

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
)

// Resources defines Databricks resources associated with the bundle.
type Resources struct {
	Jobs      map[string]*resources.Job      `json:"jobs,omitempty"`
	JobRuns   map[string]*resources.JobRun   `json:"job_runs,omitempty"`
	Pipelines map[string]*resources.Pipeline `json:"pipelines,omitempty"`

	Features                  map[string]*resources.Feature                  `json:"features,omitempty"`
	MaterializationSets       map[string]*resources.MaterializationSet       `json:"materialization_sets,omitempty"`
	FeatureSpecs              map[string]*resources.FeatureSpec              `json:"feature_specs,omitempty"`
	Models                    map[string]*resources.MlflowModel              `json:"models,omitempty"`
	Experiments               map[string]*resources.MlflowExperiment         `json:"experiments,omitempty"`
	ModelServingEndpoints     map[string]*resources.ModelServingEndpoint     `json:"model_serving_endpoints,omitempty"`
	ModelServices             map[string]*resources.ModelService             `json:"model_services,omitempty"`
	McpServices               map[string]*resources.McpService               `json:"mcp_services,omitempty"`
	ModelProviderServices     map[string]*resources.ModelProviderService     `json:"model_provider_services,omitempty"`
	RegisteredModels          map[string]*resources.RegisteredModel          `json:"registered_models,omitempty"`
	QualityMonitors           map[string]*resources.QualityMonitor           `json:"quality_monitors,omitempty"`
	DataQualityMonitors       map[string]*resources.DataQualityMonitor       `json:"data_quality_monitors,omitempty"`
	Catalogs                  map[string]*resources.Catalog                  `json:"catalogs,omitempty"`
	Schemas                   map[string]*resources.Schema                   `json:"schemas,omitempty"`
	Volumes                   map[string]*resources.Volume                   `json:"volumes,omitempty"`
	ExternalLocations         map[string]*resources.ExternalLocation         `json:"external_locations,omitempty"`
	Clusters                  map[string]*resources.Cluster                  `json:"clusters,omitempty"`
	Dashboards                map[string]*resources.Dashboard                `json:"dashboards,omitempty"`
	GenieSpaces               map[string]*resources.GenieSpace               `json:"genie_spaces,omitempty"`
	Apps                      map[string]*resources.App                      `json:"apps,omitempty"`
	SecretScopes              map[string]*resources.SecretScope              `json:"secret_scopes,omitempty"`
	Alerts                    map[string]*resources.Alert                    `json:"alerts,omitempty"`
	SqlWarehouses             map[string]*resources.SqlWarehouse             `json:"sql_warehouses,omitempty"`
	DatabaseInstances         map[string]*resources.DatabaseInstance         `json:"database_instances,omitempty"`
	DatabaseCatalogs          map[string]*resources.DatabaseCatalog          `json:"database_catalogs,omitempty"`
	SyncedDatabaseTables      map[string]*resources.SyncedDatabaseTable      `json:"synced_database_tables,omitempty"`
	PostgresProjects          map[string]*resources.PostgresProject          `json:"postgres_projects,omitempty"`
	PostgresBranches          map[string]*resources.PostgresBranch           `json:"postgres_branches,omitempty"`
	PostgresEndpoints         map[string]*resources.PostgresEndpoint         `json:"postgres_endpoints,omitempty"`
	PostgresCatalogs          map[string]*resources.PostgresCatalog          `json:"postgres_catalogs,omitempty"`
	PostgresDatabases         map[string]*resources.PostgresDatabase         `json:"postgres_databases,omitempty"`
	PostgresRoles             map[string]*resources.PostgresRole             `json:"postgres_roles,omitempty"`
	PostgresSyncedTables      map[string]*resources.PostgresSyncedTable      `json:"postgres_synced_tables,omitempty"`
	PostgresSnapshotSchedules map[string]*resources.PostgresSnapshotSchedule `json:"postgres_snapshot_schedules,omitempty"`
	PostgresDataApis          map[string]*resources.PostgresDataApi          `json:"postgres_data_apis,omitempty"`
	VectorSearchEndpoints     map[string]*resources.VectorSearchEndpoint     `json:"vector_search_endpoints,omitempty"`
	VectorSearchIndexes       map[string]*resources.VectorSearchIndex        `json:"vector_search_indexes,omitempty"`
	InstancePools             map[string]*resources.InstancePool             `json:"instance_pools,omitempty"`
	Secrets                   map[string]*resources.Secret                   `json:"secrets,omitempty"`
	ClusterPolicies           map[string]*resources.ClusterPolicy            `json:"cluster_policies,omitempty"`

	// Internal resources
	Snapshots map[string]*resources.Snapshot `json:"internal_immutable_snapshots,omitempty" bundle:"internal"`
}

type ConfigResource interface {
	// Exists returns true if the resource exists in the workspace configured in
	// the input workspace client.
	Exists(ctx context.Context, w *databricks.WorkspaceClient, id string) (bool, error)

	// ResourceDescription returns a struct containing strings describing a resource
	ResourceDescription() resources.ResourceDescription

	// GetName returns the in-product name of the resource.
	GetName() string

	// GetURL returns the resource's workspace URL and whether this resource type
	// has one at all. A supported type that is not deployed yet returns ("", true);
	// a type that never has a URL returns ("", false).
	GetURL() (string, bool)

	// InitializeURL initializes the URL field of the resource.
	InitializeURL(baseURL url.URL)

	// GetLifecycle returns the lifecycle settings for the resource.
	GetLifecycle() resources.LifecycleConfig
}

// ResourceGroup represents a group of resources of the same type.
// It includes a description of the resource type and a map of resources.
type ResourceGroup struct {
	Description resources.ResourceDescription
	Resources   map[string]ConfigResource
}

// AllResources returns all resources in the bundle grouped by their resource type,
// with one group per resource type in the order of [AllResourceTypes].
func (r *Resources) AllResources() []ResourceGroup {
	rv := reflect.ValueOf(r).Elem()
	types := allResourceTypes()
	groups := make([]ResourceGroup, 0, len(types))
	for _, rt := range types {
		field := rv.Field(rt.fieldIndex)
		group := make(map[string]ConfigResource, field.Len())
		for it := field.MapRange(); it.Next(); {
			group[it.Key().String()] = it.Value().Interface().(ConfigResource)
		}
		groups = append(groups, ResourceGroup{
			Description: rt.Description,
			Resources:   group,
		})
	}
	return groups
}

// ResolveKey finds a resource by "key" or "type.key", e.g. "foo" or "jobs.foo", and returns it
// together with its key. A bare key must match exactly one resource across all types.
func (r *Resources) ResolveKey(ref string) (ConfigResource, string, error) {
	groups := r.AllResources()

	if typeName, key, ok := strings.Cut(ref, "."); ok {
		for _, group := range groups {
			if group.Description.PluralName != typeName {
				continue
			}
			if res, ok := group.Resources[key]; ok {
				return res, key, nil
			}
			break
		}
		return nil, "", fmt.Errorf("no such resource: %s", ref)
	}

	var found []ConfigResource
	var matches []string
	for _, group := range groups {
		if res, ok := group.Resources[ref]; ok {
			found = append(found, res)
			matches = append(matches, group.Description.PluralName+"."+ref)
		}
	}

	switch len(found) {
	case 0:
		return nil, "", fmt.Errorf("no such resource: %s", ref)
	case 1:
		return found[0], ref, nil
	default:
		return nil, "", fmt.Errorf("ambiguous resource: %s (can resolve to %s); use a qualified name to disambiguate", ref, strings.Join(matches, ", "))
	}
}

// ResolveDependsOn returns the key of the node that a lifecycle.depends_on entry refers to:
// "resources.jobs.foo" for "foo" or "jobs.foo", and "resources.schemas.foo.grants" for the
// grants or permissions of a qualified key, as "schemas.foo.grants".
func (r *Resources) ResolveDependsOn(ref string) (string, error) {
	resourceRef, sub := ref, ""
	if parts := strings.Split(ref, "."); len(parts) == 3 {
		if _, ok := ResourcesTypes()[parts[0]+"."+parts[2]]; !ok {
			return "", fmt.Errorf("no such resource: %s", ref)
		}
		resourceRef, sub = parts[0]+"."+parts[1], "."+parts[2]
	}

	res, key, err := r.ResolveKey(resourceRef)
	if err != nil {
		return "", err
	}
	return "resources." + res.ResourceDescription().PluralName + "." + key + sub, nil
}

func (r *Resources) HasInternalResources() bool {
	return len(r.Snapshots) > 0
}

// SupportedResources returns a map which keys correspond to the resource key in the bundle configuration.
func SupportedResources() map[string]resources.ResourceDescription {
	types := allResourceTypes()
	res := make(map[string]resources.ResourceDescription, len(types))
	for _, rt := range types {
		res[rt.Key] = rt.Description
	}
	return res
}
