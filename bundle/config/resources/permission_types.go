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

type (
	AlertPermission  Permission[iam.PermissionLevel]
	AlertPermissions []AlertPermission
)

func (ps AlertPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[iam.PermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	AppPermission  Permission[apps.AppPermissionLevel]
	AppPermissions []AppPermission
)

func (ps AppPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[apps.AppPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	ClusterPermission  Permission[compute.ClusterPermissionLevel]
	ClusterPermissions []ClusterPermission
)

func (ps ClusterPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[compute.ClusterPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	DashboardPermission  Permission[iam.PermissionLevel]
	DashboardPermissions []DashboardPermission
)

func (ps DashboardPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[iam.PermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	DatabaseInstancePermission  Permission[iam.PermissionLevel]
	DatabaseInstancePermissions []DatabaseInstancePermission
)

func (ps DatabaseInstancePermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[iam.PermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	DatabaseProjectPermission  Permission[iam.PermissionLevel]
	DatabaseProjectPermissions []DatabaseProjectPermission
)

func (ps DatabaseProjectPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[iam.PermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	JobPermission  Permission[jobs.JobPermissionLevel]
	JobPermissions []JobPermission
)

func (ps JobPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[jobs.JobPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	MlflowExperimentPermission  Permission[ml.ExperimentPermissionLevel]
	MlflowExperimentPermissions []MlflowExperimentPermission
)

func (ps MlflowExperimentPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[ml.ExperimentPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	MlflowModelPermission  Permission[ml.RegisteredModelPermissionLevel]
	MlflowModelPermissions []MlflowModelPermission
)

func (ps MlflowModelPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[ml.RegisteredModelPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	ModelServingEndpointPermission  Permission[serving.ServingEndpointPermissionLevel]
	ModelServingEndpointPermissions []ModelServingEndpointPermission
)

func (ps ModelServingEndpointPermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[serving.ServingEndpointPermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	PipelinePermission  Permission[pipelines.PipelinePermissionLevel]
	PipelinePermissions []PipelinePermission
)

func (ps PipelinePermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[pipelines.PipelinePermissionLevel](p).ToAccessControlRequest()
	}
	return result
}

type (
	SqlWarehousePermission  Permission[sql.WarehousePermissionLevel]
	SqlWarehousePermissions []SqlWarehousePermission
)

func (ps SqlWarehousePermissions) ToAccessControlRequests() []iam.AccessControlRequest {
	result := make([]iam.AccessControlRequest, len(ps))
	for i, p := range ps {
		result[i] = Permission[sql.WarehousePermissionLevel](p).ToAccessControlRequest()
	}
	return result
}
