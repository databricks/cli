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
// The underlying struct is identical to PermissionT[L], enabling conversion to use generic methods.

// Permission is used for resources that use the generic iam.PermissionLevel (Alert, Dashboard, DatabaseInstance, PostgresProject).
type Permission PermissionT[iam.PermissionLevel]

func (p Permission) String() string {
	return PermissionT[iam.PermissionLevel](p).String()
}

type (
	AppPermission                  PermissionT[apps.AppPermissionLevel]
	ClusterPermission              PermissionT[compute.ClusterPermissionLevel]
	JobPermission                  PermissionT[jobs.JobPermissionLevel]
	MlflowExperimentPermission     PermissionT[ml.ExperimentPermissionLevel]
	MlflowModelPermission          PermissionT[ml.RegisteredModelPermissionLevel]
	ModelServingEndpointPermission PermissionT[serving.ServingEndpointPermissionLevel]
	PipelinePermission             PermissionT[pipelines.PipelinePermissionLevel]
	SqlWarehousePermission         PermissionT[sql.WarehousePermissionLevel]
)
