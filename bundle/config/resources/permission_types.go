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

func (p IamPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[iam.PermissionLevel](p).ToAccessControlRequest()
}

func (p IamPermission) String() string {
	return Permission[iam.PermissionLevel](p).String()
}

type AppPermission Permission[apps.AppPermissionLevel]

func (p AppPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[apps.AppPermissionLevel](p).ToAccessControlRequest()
}

type ClusterPermission Permission[compute.ClusterPermissionLevel]

func (p ClusterPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[compute.ClusterPermissionLevel](p).ToAccessControlRequest()
}

type JobPermission Permission[jobs.JobPermissionLevel]

func (p JobPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[jobs.JobPermissionLevel](p).ToAccessControlRequest()
}

type MlflowExperimentPermission Permission[ml.ExperimentPermissionLevel]

func (p MlflowExperimentPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[ml.ExperimentPermissionLevel](p).ToAccessControlRequest()
}

type MlflowModelPermission Permission[ml.RegisteredModelPermissionLevel]

func (p MlflowModelPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[ml.RegisteredModelPermissionLevel](p).ToAccessControlRequest()
}

type ModelServingEndpointPermission Permission[serving.ServingEndpointPermissionLevel]

func (p ModelServingEndpointPermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[serving.ServingEndpointPermissionLevel](p).ToAccessControlRequest()
}

type PipelinePermission Permission[pipelines.PipelinePermissionLevel]

func (p PipelinePermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[pipelines.PipelinePermissionLevel](p).ToAccessControlRequest()
}

type SqlWarehousePermission Permission[sql.WarehousePermissionLevel]

func (p SqlWarehousePermission) ToAccessControlRequest() iam.AccessControlRequest {
	return Permission[sql.WarehousePermissionLevel](p).ToAccessControlRequest()
}
