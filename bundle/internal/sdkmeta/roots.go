package sdkmeta

import (
	"reflect"
	"sync"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/qualitymonitors"
	"github.com/databricks/cli/libs/sdktypes/aigateway/v1"
	"github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/sdktypes/apps/v1"
	"github.com/databricks/cli/libs/sdktypes/clusterlibraries/v2"
	"github.com/databricks/cli/libs/sdktypes/clusterpolicies/v2"
	"github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/sdktypes/database/v1"
	"github.com/databricks/cli/libs/sdktypes/experiments/v1"
	"github.com/databricks/cli/libs/sdktypes/instancepools/v2"
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	"github.com/databricks/cli/libs/sdktypes/modelregistry/v1"
	"github.com/databricks/cli/libs/sdktypes/modelserving/v1"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/sdktypes/secrets/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/grants/v1"
	"github.com/databricks/cli/libs/sdktypes/uc/registeredmodels/v1"
	ucsecrets "github.com/databricks/cli/libs/sdktypes/uc/secrets/v1"
	"github.com/databricks/cli/libs/sdktypes/vectorsearch/v1"
	"github.com/databricks/cli/libs/sdktypes/warehouses/v1"
	sdktypes "github.com/databricks/sdk-go/core/types"
)

// Pseudo schema names for sdk-go native types; they are not in cli.json and
// keep the definition names of the previous SDK's equivalents.
const (
	nativeTime     = "@time"
	nativeDuration = "@duration"
)

var nativeDefinitionNames = map[string]string{
	nativeTime:     "github.com/databricks/databricks-sdk-go/common/types/time.Time",
	nativeDuration: "github.com/databricks/databricks-sdk-go/common/types/duration.Duration",
}

// roots maps the API types that the bundle configuration references directly
// (embedded in resources or used by CLI-owned fields) to their cli.json schema.
// Types nested inside them are mapped by following cli.json refs. CLI-owned
// stand-ins for API types (resources.CreateCatalog, permission levels) are
// listed here too so they keep their previous schema definition names.
var roots = sync.OnceValue(func() map[reflect.Type]string {
	return map[reflect.Type]string{
		reflect.TypeFor[sdktypes.Time]():     nativeTime,
		reflect.TypeFor[sdktypes.Duration](): nativeDuration,

		reflect.TypeFor[alerts.Alert]():                                  "sql.AlertV2",
		reflect.TypeFor[apps.App]():                                      "apps.App",
		reflect.TypeFor[resources.CreateCatalog]():                       "catalog.CreateCatalog",
		reflect.TypeFor[clusterpolicies.CreatePolicyRequest]():           "compute.CreatePolicy",
		reflect.TypeFor[clusters.CreateClusterRequest]():                 "compute.ClusterSpec",
		reflect.TypeFor[clusterlibraries.Library]():                      "compute.Library",
		reflect.TypeFor[lakeview.LifecycleState]():                       "dashboards.LifecycleState",
		reflect.TypeFor[database.DatabaseCatalog]():                      "database.DatabaseCatalog",
		reflect.TypeFor[database.DatabaseInstance]():                     "database.DatabaseInstance",
		reflect.TypeFor[database.SyncedDatabaseTable]():                  "database.SyncedDatabaseTable",
		reflect.TypeFor[resources.CreateExternalLocation]():              "catalog.CreateExternalLocation",
		reflect.TypeFor[instancepools.CreateInstancePoolRequest]():       "compute.CreateInstancePool",
		reflect.TypeFor[jobs.JobSettings]():                              "jobs.JobSettings",
		reflect.TypeFor[jobs.RunNowRequest]():                            "jobs.RunNow",
		reflect.TypeFor[aigateway.McpServiceConfig]():                    "catalog.McpServiceConfig",
		reflect.TypeFor[aigateway.ModelProviderServiceConfig]():          "catalog.ModelProviderServiceConfig",
		reflect.TypeFor[aigateway.ModelServiceConfig]():                  "catalog.ModelServiceConfig",
		reflect.TypeFor[experiments.CreateExperimentRequest]():           "ml.CreateExperiment",
		reflect.TypeFor[modelregistry.CreateRegisteredModelRequest]():    "ml.CreateModelRequest",
		reflect.TypeFor[modelserving.CreateInferenceEndpointRequest]():   "serving.CreateServingEndpoint",
		reflect.TypeFor[pipelines.CreatePipelineRequest]():               "pipelines.CreatePipeline",
		reflect.TypeFor[postgres.BranchSpec]():                           "postgres.BranchSpec",
		reflect.TypeFor[postgres.Catalog_CatalogSpec]():                  "postgres.CatalogCatalogSpec",
		reflect.TypeFor[postgres.Database_DatabaseSpec]():                "postgres.DatabaseDatabaseSpec",
		reflect.TypeFor[postgres.EndpointSpec]():                         "postgres.EndpointSpec",
		reflect.TypeFor[postgres.ProjectSpec]():                          "postgres.ProjectSpec",
		reflect.TypeFor[postgres.Role_RoleSpec]():                        "postgres.RoleRoleSpec",
		reflect.TypeFor[postgres.ScheduleCadence]():                      "postgres.ScheduleCadence",
		reflect.TypeFor[postgres.SyncedTable_SyncedTableSpec]():          "postgres.SyncedTableSyncedTableSpec",
		reflect.TypeFor[qualitymonitors.CreateMonitor]():                 "catalog.CreateMonitor",
		reflect.TypeFor[registeredmodels.CreateRegisteredModelRequest](): "catalog.CreateRegisteredModelRequest",
		reflect.TypeFor[resources.CreateSchema]():                        "catalog.CreateSchema",
		reflect.TypeFor[ucsecrets.Secret]():                              "catalog.Secret",
		reflect.TypeFor[secrets.ScopeBackendType]():                      "workspace.ScopeBackendType",
		reflect.TypeFor[secrets.AzureKeyVaultSecretScopeMetadata]():      "workspace.AzureKeyVaultSecretScopeMetadata",
		reflect.TypeFor[warehouses.CreateWarehouseRequest]():             "sql.CreateWarehouseRequest",
		reflect.TypeFor[vectorsearch.CreateEndpointRequest]():            "vectorsearch.CreateEndpoint",
		reflect.TypeFor[vectorsearch.CreateVectorIndexRequest]():         "vectorsearch.CreateVectorIndexRequest",
		reflect.TypeFor[resources.CreateVolumeRequestContent]():          "catalog.CreateVolumeRequestContent",
		reflect.TypeFor[grants.PrivilegeAssignment]():                    "catalog.PrivilegeAssignment",

		reflect.TypeFor[resources.AppPermissionLevel]():                  "apps.AppPermissionLevel",
		reflect.TypeFor[resources.ClusterPermissionLevel]():              "compute.ClusterPermissionLevel",
		reflect.TypeFor[resources.ClusterPolicyPermissionLevel]():        "compute.ClusterPolicyPermissionLevel",
		reflect.TypeFor[resources.InstancePoolPermissionLevel]():         "compute.InstancePoolPermissionLevel",
		reflect.TypeFor[resources.JobPermissionLevel]():                  "jobs.JobPermissionLevel",
		reflect.TypeFor[resources.ExperimentPermissionLevel]():           "ml.ExperimentPermissionLevel",
		reflect.TypeFor[resources.RegisteredModelPermissionLevel]():      "ml.RegisteredModelPermissionLevel",
		reflect.TypeFor[resources.ServingEndpointPermissionLevel]():      "serving.ServingEndpointPermissionLevel",
		reflect.TypeFor[resources.PipelinePermissionLevel]():             "pipelines.PipelinePermissionLevel",
		reflect.TypeFor[resources.WarehousePermissionLevel]():            "sql.WarehousePermissionLevel",
		reflect.TypeFor[resources.VectorSearchEndpointPermissionLevel](): "vectorsearch.VectorSearchEndpointPermissionLevel",
		reflect.TypeFor[resources.PermissionLevel]():                     "iam.PermissionLevel",
	}
})
