package sdkmeta

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/require"
)

type testRoot struct {
	Jobs                      map[string]*resources.Job                      `json:"jobs,omitempty"`
	JobRuns                   map[string]*resources.JobRun                   `json:"job_runs,omitempty"`
	Pipelines                 map[string]*resources.Pipeline                 `json:"pipelines,omitempty"`
	Models                    map[string]*resources.MlflowModel              `json:"models,omitempty"`
	Experiments               map[string]*resources.MlflowExperiment         `json:"experiments,omitempty"`
	ModelServingEndpoints     map[string]*resources.ModelServingEndpoint     `json:"model_serving_endpoints,omitempty"`
	ModelServices             map[string]*resources.ModelService             `json:"model_services,omitempty"`
	McpServices               map[string]*resources.McpService               `json:"mcp_services,omitempty"`
	ModelProviderServices     map[string]*resources.ModelProviderService     `json:"model_provider_services,omitempty"`
	RegisteredModels          map[string]*resources.RegisteredModel          `json:"registered_models,omitempty"`
	QualityMonitors           map[string]*resources.QualityMonitor           `json:"quality_monitors,omitempty"`
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
	VectorSearchEndpoints     map[string]*resources.VectorSearchEndpoint     `json:"vector_search_endpoints,omitempty"`
	VectorSearchIndexes       map[string]*resources.VectorSearchIndex        `json:"vector_search_indexes,omitempty"`
	InstancePools             map[string]*resources.InstancePool             `json:"instance_pools,omitempty"`
	Secrets                   map[string]*resources.Secret                   `json:"secrets,omitempty"`
	ClusterPolicies           map[string]*resources.ClusterPolicy            `json:"cluster_policies,omitempty"`
	Snapshots                 map[string]*resources.Snapshot                 `json:"internal_immutable_snapshots,omitempty" bundle:"internal"`
}

func TestBuild(t *testing.T) {
	doc, err := clijson.Parse("../../../.codegen/cli.json")
	require.NoError(t, err)
	_, err = Build(reflect.TypeFor[testRoot](), doc.Schemas)
	require.NoError(t, err)
}

func TestDump(t *testing.T) {
	doc, _ := clijson.Parse("../../../.codegen/cli.json")
	m, _ := Build(reflect.TypeFor[testRoot](), doc.Schemas)
	byOld := map[string][]string{}
	for typ, n := range m.Types() {
		byOld[n] = append(byOld[n], typeName(typ))
	}
	for n, ts := range byOld {
		if len(ts) > 1 {
			t.Logf("SHARED %s <- %v", n, ts)
		}
		if _, ok := doc.Schemas[n]; !ok {
			t.Logf("NOSCHEMA %s <- %v", n, ts)
		}
	}
	t.Logf("mapped %d", len(m.Types()))
}

func TestEnvVar(t *testing.T) {
	doc, _ := clijson.Parse("../../../.codegen/cli.json")
	m, _ := Build(reflect.TypeFor[testRoot](), doc.Schemas)
	for typ, n := range m.Types() {
		if typ.Name() == "EnvVar" {
			t.Logf("%s %s", typeName(typ), n)
		}
	}
}

func TestPipelinesStorage(t *testing.T) {
	doc, _ := clijson.Parse("../../../.codegen/cli.json")
	m, _ := Build(reflect.TypeFor[testRoot](), doc.Schemas)
	for typ, n := range m.Types() {
		if strings.Contains(typeName(typ), "pipelines/v2") && strings.Contains(typ.Name(), "Info") {
			t.Logf("%s %s %v", typeName(typ), n, m.Required(typ))
		}
	}
}
