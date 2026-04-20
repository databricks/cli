// Package tmpdms is a temporary client library for the Deployment Metadata Service.
// It mirrors the structure that the Databricks Go SDK will eventually generate from
// the service's proto definitions. When the protos land in the SDK, migration should
// be a straightforward import path change.
package tmpdms

import "time"

// Enum types matching the proto definitions.
// Values are the proto enum name strings, which is how proto-over-HTTP serializes enums.

type (
	DeploymentStatus       string
	VersionStatus          string
	VersionComplete        string
	VersionType            string
	OperationStatus        string
	OperationActionType    string
	DeploymentResourceType string
)

const (
	DeploymentStatusUnspecified DeploymentStatus = "DEPLOYMENT_STATUS_UNSPECIFIED"
	DeploymentStatusActive      DeploymentStatus = "DEPLOYMENT_STATUS_ACTIVE"
	DeploymentStatusFailed      DeploymentStatus = "DEPLOYMENT_STATUS_FAILED"
	DeploymentStatusInProgress  DeploymentStatus = "DEPLOYMENT_STATUS_IN_PROGRESS"
	DeploymentStatusDeleted     DeploymentStatus = "DEPLOYMENT_STATUS_DELETED"
)

const (
	VersionStatusUnspecified VersionStatus = "VERSION_STATUS_UNSPECIFIED"
	VersionStatusInProgress  VersionStatus = "VERSION_STATUS_IN_PROGRESS"
	VersionStatusCompleted   VersionStatus = "VERSION_STATUS_COMPLETED"
)

const (
	VersionCompleteUnspecified  VersionComplete = "VERSION_COMPLETE_UNSPECIFIED"
	VersionCompleteSuccess      VersionComplete = "VERSION_COMPLETE_SUCCESS"
	VersionCompleteFailure      VersionComplete = "VERSION_COMPLETE_FAILURE"
	VersionCompleteForceAbort   VersionComplete = "VERSION_COMPLETE_FORCE_ABORT"
	VersionCompleteLeaseExpired VersionComplete = "VERSION_COMPLETE_LEASE_EXPIRED"
)

const (
	VersionTypeUnspecified VersionType = "VERSION_TYPE_UNSPECIFIED"
	VersionTypeDeploy      VersionType = "VERSION_TYPE_DEPLOY"
	VersionTypeDestroy     VersionType = "VERSION_TYPE_DESTROY"
)

const (
	OperationStatusUnspecified OperationStatus = "OPERATION_STATUS_UNSPECIFIED"
	OperationStatusSucceeded   OperationStatus = "OPERATION_STATUS_SUCCEEDED"
	OperationStatusFailed      OperationStatus = "OPERATION_STATUS_FAILED"
)

const (
	OperationActionTypeUnspecified   OperationActionType = "OPERATION_ACTION_TYPE_UNSPECIFIED"
	OperationActionTypeResize        OperationActionType = "OPERATION_ACTION_TYPE_RESIZE"
	OperationActionTypeUpdate        OperationActionType = "OPERATION_ACTION_TYPE_UPDATE"
	OperationActionTypeUpdateWithID  OperationActionType = "OPERATION_ACTION_TYPE_UPDATE_WITH_ID"
	OperationActionTypeCreate        OperationActionType = "OPERATION_ACTION_TYPE_CREATE"
	OperationActionTypeRecreate      OperationActionType = "OPERATION_ACTION_TYPE_RECREATE"
	OperationActionTypeDelete        OperationActionType = "OPERATION_ACTION_TYPE_DELETE"
	OperationActionTypeBind          OperationActionType = "OPERATION_ACTION_TYPE_BIND"
	OperationActionTypeBindAndUpdate OperationActionType = "OPERATION_ACTION_TYPE_BIND_AND_UPDATE"
	OperationActionTypeInitRegister  OperationActionType = "OPERATION_ACTION_TYPE_INITIAL_REGISTER"
)

const (
	ResourceTypeUnspecified      DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_UNSPECIFIED"
	ResourceTypeJob              DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_JOB"
	ResourceTypePipeline         DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_PIPELINE"
	ResourceTypeModel            DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_MODEL"
	ResourceTypeRegisteredModel  DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_REGISTERED_MODEL"
	ResourceTypeExperiment       DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_EXPERIMENT"
	ResourceTypeServingEndpoint  DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_MODEL_SERVING_ENDPOINT"
	ResourceTypeQualityMonitor   DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_QUALITY_MONITOR"
	ResourceTypeSchema           DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_SCHEMA"
	ResourceTypeVolume           DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_VOLUME"
	ResourceTypeCluster          DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_CLUSTER"
	ResourceTypeDashboard        DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_DASHBOARD"
	ResourceTypeApp              DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_APP"
	ResourceTypeCatalog          DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_CATALOG"
	ResourceTypeExternalLocation DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_EXTERNAL_LOCATION"
	ResourceTypeSecretScope      DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_SECRET_SCOPE"
	ResourceTypeAlert            DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_ALERT"
	ResourceTypeSQLWarehouse     DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_SQL_WAREHOUSE"
	ResourceTypeDatabaseInstance DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_DATABASE_INSTANCE"
	ResourceTypeDatabaseCatalog  DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_DATABASE_CATALOG"
	ResourceTypeSyncedDBTable    DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_SYNCED_DATABASE_TABLE"
	ResourceTypePostgresProject  DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_POSTGRES_PROJECT"
	ResourceTypePostgresBranch   DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_POSTGRES_BRANCH"
	ResourceTypePostgresEndpoint DeploymentResourceType = "DEPLOYMENT_RESOURCE_TYPE_POSTGRES_ENDPOINT"
)

// Resource types (proto message equivalents).

type Deployment struct {
	Name          string           `json:"name,omitempty"`
	DisplayName   string           `json:"display_name,omitempty"`
	TargetName    string           `json:"target_name,omitempty"`
	Status        DeploymentStatus `json:"status,omitempty"`
	LastVersionID string           `json:"last_version_id,omitempty"`
	CreatedBy     string           `json:"created_by,omitempty"`
	CreateTime    *time.Time       `json:"create_time,omitempty"`
	UpdateTime    *time.Time       `json:"update_time,omitempty"`
	DestroyTime   *time.Time       `json:"destroy_time,omitempty"`
	DestroyedBy   string           `json:"destroyed_by,omitempty"`
}

type Version struct {
	Name             string          `json:"name,omitempty"`
	VersionID        string          `json:"version_id,omitempty"`
	CreatedBy        string          `json:"created_by,omitempty"`
	CreateTime       *time.Time      `json:"create_time,omitempty"`
	CompleteTime     *time.Time      `json:"complete_time,omitempty"`
	CliVersion       string          `json:"cli_version,omitempty"`
	Status           VersionStatus   `json:"status,omitempty"`
	VersionType      VersionType     `json:"version_type,omitempty"`
	CompletionReason VersionComplete `json:"completion_reason,omitempty"`
	CompletedBy      string          `json:"completed_by,omitempty"`
	DisplayName      string          `json:"display_name,omitempty"`
	TargetName       string          `json:"target_name,omitempty"`
}

type Operation struct {
	Name         string              `json:"name,omitempty"`
	ResourceKey  string              `json:"resource_key,omitempty"`
	ActionType   OperationActionType `json:"action_type,omitempty"`
	State        any                 `json:"state,omitempty"`
	ResourceID   string              `json:"resource_id,omitempty"`
	CreateTime   *time.Time          `json:"create_time,omitempty"`
	Status       OperationStatus     `json:"status,omitempty"`
	ErrorMessage string              `json:"error_message,omitempty"`
}

type Resource struct {
	Name           string                 `json:"name,omitempty"`
	ResourceKey    string                 `json:"resource_key,omitempty"`
	State          any                    `json:"state,omitempty"`
	ResourceID     string                 `json:"resource_id,omitempty"`
	LastActionType OperationActionType    `json:"last_action_type,omitempty"`
	LastVersionID  string                 `json:"last_version_id,omitempty"`
	ResourceType   DeploymentResourceType `json:"resource_type,omitempty"`
}

// Request types.

type CreateDeploymentRequest struct {
	DeploymentID string      `json:"deployment_id"`
	Deployment   *Deployment `json:"deployment"`
}

type GetDeploymentRequest struct {
	DeploymentID string `json:"-"`
}

type DeleteDeploymentRequest struct {
	DeploymentID string `json:"-"`
}

type CreateVersionRequest struct {
	DeploymentID string   `json:"-"`
	Parent       string   `json:"parent"`
	Version      *Version `json:"version"`
	VersionID    string   `json:"version_id"`
}

type GetVersionRequest struct {
	DeploymentID string `json:"-"`
	VersionID    string `json:"-"`
}

type HeartbeatRequest struct {
	DeploymentID string `json:"-"`
	VersionID    string `json:"-"`
}

type CompleteVersionRequest struct {
	DeploymentID     string          `json:"-"`
	VersionID        string          `json:"-"`
	Name             string          `json:"name"`
	CompletionReason VersionComplete `json:"completion_reason"`
	Force            bool            `json:"force,omitempty"`
}

type CreateOperationRequest struct {
	DeploymentID string     `json:"-"`
	VersionID    string     `json:"-"`
	Parent       string     `json:"parent"`
	ResourceKey  string     `json:"resource_key"`
	Operation    *Operation `json:"operation"`
}

type ListResourcesRequest struct {
	DeploymentID string `json:"-"`
	Parent       string `json:"parent"`
	PageSize     int    `json:"page_size,omitempty"`
	PageToken    string `json:"page_token,omitempty"`
}

// Response types.

type HeartbeatResponse struct {
	ExpireTime *time.Time `json:"expire_time,omitempty"`
}

type ListDeploymentsResponse struct {
	Deployments   []Deployment `json:"deployments"`
	NextPageToken string       `json:"next_page_token,omitempty"`
}

type ListVersionsResponse struct {
	Versions      []Version `json:"versions"`
	NextPageToken string    `json:"next_page_token,omitempty"`
}

type ListOperationsResponse struct {
	Operations    []Operation `json:"operations"`
	NextPageToken string      `json:"next_page_token,omitempty"`
}

type ListResourcesResponse struct {
	Resources     []Resource `json:"resources"`
	NextPageToken string     `json:"next_page_token,omitempty"`
}
