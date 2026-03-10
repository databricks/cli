package resources

import (
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

// Each resource defines its own permission type so that the JSON schema names them distinctly.
// Using non-alias type definitions (not =) makes them appear as named types in the schema.
// The underlying struct is identical to Permission[L], enabling conversion to use generic methods.

// IamPermission is used for resources that use the generic iam.PermissionLevel (Alert, Dashboard, DatabaseInstance, PostgresProject).
type IamPermission Permission[iam.PermissionLevel]

func (p IamPermission) String() string {
	return Permission[iam.PermissionLevel](p).String()
}

type AppPermission Permission[apps.AppPermissionLevel]
type ClusterPermission Permission[compute.ClusterPermissionLevel]
type JobPermission Permission[jobs.JobPermissionLevel]
type MlflowExperimentPermission Permission[ml.ExperimentPermissionLevel]
type MlflowModelPermission Permission[ml.RegisteredModelPermissionLevel]
type ModelServingEndpointPermission Permission[serving.ServingEndpointPermissionLevel]
type PipelinePermission Permission[pipelines.PipelinePermissionLevel]
type SqlWarehousePermission Permission[sql.WarehousePermissionLevel]
