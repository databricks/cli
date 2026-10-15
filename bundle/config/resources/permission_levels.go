package resources

// Permission levels accepted by the Permissions API for each resource type.
// The Permissions API itself is generic (sdk-go accessmanagement); these types
// give each resource its own enum in the bundle schema and validation.

// Permission level
type AppPermissionLevel string

const (
	AppPermissionLevelCanManage AppPermissionLevel = "CAN_MANAGE"
	AppPermissionLevelCanUse    AppPermissionLevel = "CAN_USE"
)

// Values returns all valid AppPermissionLevel values.
func (AppPermissionLevel) Values() []AppPermissionLevel {
	return []AppPermissionLevel{
		AppPermissionLevelCanManage,
		AppPermissionLevelCanUse,
	}
}

// Permission level
type ClusterPermissionLevel string

const (
	ClusterPermissionLevelCanAttachTo ClusterPermissionLevel = "CAN_ATTACH_TO"
	ClusterPermissionLevelCanManage   ClusterPermissionLevel = "CAN_MANAGE"
	ClusterPermissionLevelCanRestart  ClusterPermissionLevel = "CAN_RESTART"
)

// Values returns all valid ClusterPermissionLevel values.
func (ClusterPermissionLevel) Values() []ClusterPermissionLevel {
	return []ClusterPermissionLevel{
		ClusterPermissionLevelCanAttachTo,
		ClusterPermissionLevelCanManage,
		ClusterPermissionLevelCanRestart,
	}
}

// Permission level
type ClusterPolicyPermissionLevel string

const (
	ClusterPolicyPermissionLevelCanUse ClusterPolicyPermissionLevel = "CAN_USE"
)

// Values returns all valid ClusterPolicyPermissionLevel values.
func (ClusterPolicyPermissionLevel) Values() []ClusterPolicyPermissionLevel {
	return []ClusterPolicyPermissionLevel{
		ClusterPolicyPermissionLevelCanUse,
	}
}

// Permission level
type InstancePoolPermissionLevel string

const (
	InstancePoolPermissionLevelCanAttachTo InstancePoolPermissionLevel = "CAN_ATTACH_TO"
	InstancePoolPermissionLevelCanManage   InstancePoolPermissionLevel = "CAN_MANAGE"
)

// Values returns all valid InstancePoolPermissionLevel values.
func (InstancePoolPermissionLevel) Values() []InstancePoolPermissionLevel {
	return []InstancePoolPermissionLevel{
		InstancePoolPermissionLevelCanAttachTo,
		InstancePoolPermissionLevelCanManage,
	}
}

// Permission level
type JobPermissionLevel string

const (
	JobPermissionLevelCanManage    JobPermissionLevel = "CAN_MANAGE"
	JobPermissionLevelCanManageRun JobPermissionLevel = "CAN_MANAGE_RUN"
	JobPermissionLevelCanView      JobPermissionLevel = "CAN_VIEW"
	JobPermissionLevelIsOwner      JobPermissionLevel = "IS_OWNER"
)

// Values returns all valid JobPermissionLevel values.
func (JobPermissionLevel) Values() []JobPermissionLevel {
	return []JobPermissionLevel{
		JobPermissionLevelCanManage,
		JobPermissionLevelCanManageRun,
		JobPermissionLevelCanView,
		JobPermissionLevelIsOwner,
	}
}

// Permission level
type ExperimentPermissionLevel string

const (
	ExperimentPermissionLevelCanEdit   ExperimentPermissionLevel = "CAN_EDIT"
	ExperimentPermissionLevelCanManage ExperimentPermissionLevel = "CAN_MANAGE"
	ExperimentPermissionLevelCanRead   ExperimentPermissionLevel = "CAN_READ"
)

// Values returns all valid ExperimentPermissionLevel values.
func (ExperimentPermissionLevel) Values() []ExperimentPermissionLevel {
	return []ExperimentPermissionLevel{
		ExperimentPermissionLevelCanEdit,
		ExperimentPermissionLevelCanManage,
		ExperimentPermissionLevelCanRead,
	}
}

// Permission level
type RegisteredModelPermissionLevel string

const (
	RegisteredModelPermissionLevelCanEdit                     RegisteredModelPermissionLevel = "CAN_EDIT"
	RegisteredModelPermissionLevelCanManage                   RegisteredModelPermissionLevel = "CAN_MANAGE"
	RegisteredModelPermissionLevelCanManageProductionVersions RegisteredModelPermissionLevel = "CAN_MANAGE_PRODUCTION_VERSIONS"
	RegisteredModelPermissionLevelCanManageStagingVersions    RegisteredModelPermissionLevel = "CAN_MANAGE_STAGING_VERSIONS"
	RegisteredModelPermissionLevelCanRead                     RegisteredModelPermissionLevel = "CAN_READ"
)

// Values returns all valid RegisteredModelPermissionLevel values.
func (RegisteredModelPermissionLevel) Values() []RegisteredModelPermissionLevel {
	return []RegisteredModelPermissionLevel{
		RegisteredModelPermissionLevelCanEdit,
		RegisteredModelPermissionLevelCanManage,
		RegisteredModelPermissionLevelCanManageProductionVersions,
		RegisteredModelPermissionLevelCanManageStagingVersions,
		RegisteredModelPermissionLevelCanRead,
	}
}

// Permission level
type ServingEndpointPermissionLevel string

const (
	ServingEndpointPermissionLevelCanManage ServingEndpointPermissionLevel = "CAN_MANAGE"
	ServingEndpointPermissionLevelCanQuery  ServingEndpointPermissionLevel = "CAN_QUERY"
	ServingEndpointPermissionLevelCanView   ServingEndpointPermissionLevel = "CAN_VIEW"
)

// Values returns all valid ServingEndpointPermissionLevel values.
func (ServingEndpointPermissionLevel) Values() []ServingEndpointPermissionLevel {
	return []ServingEndpointPermissionLevel{
		ServingEndpointPermissionLevelCanManage,
		ServingEndpointPermissionLevelCanQuery,
		ServingEndpointPermissionLevelCanView,
	}
}

// Permission level
type PipelinePermissionLevel string

const (
	PipelinePermissionLevelCanManage PipelinePermissionLevel = "CAN_MANAGE"
	PipelinePermissionLevelCanRun    PipelinePermissionLevel = "CAN_RUN"
	PipelinePermissionLevelCanView   PipelinePermissionLevel = "CAN_VIEW"
	PipelinePermissionLevelIsOwner   PipelinePermissionLevel = "IS_OWNER"
)

// Values returns all valid PipelinePermissionLevel values.
func (PipelinePermissionLevel) Values() []PipelinePermissionLevel {
	return []PipelinePermissionLevel{
		PipelinePermissionLevelCanManage,
		PipelinePermissionLevelCanRun,
		PipelinePermissionLevelCanView,
		PipelinePermissionLevelIsOwner,
	}
}

// Permission level
type WarehousePermissionLevel string

const (
	WarehousePermissionLevelCanManage  WarehousePermissionLevel = "CAN_MANAGE"
	WarehousePermissionLevelCanMonitor WarehousePermissionLevel = "CAN_MONITOR"
	WarehousePermissionLevelCanUse     WarehousePermissionLevel = "CAN_USE"
	WarehousePermissionLevelCanView    WarehousePermissionLevel = "CAN_VIEW"
	WarehousePermissionLevelIsOwner    WarehousePermissionLevel = "IS_OWNER"
)

// Values returns all valid WarehousePermissionLevel values.
func (WarehousePermissionLevel) Values() []WarehousePermissionLevel {
	return []WarehousePermissionLevel{
		WarehousePermissionLevelCanManage,
		WarehousePermissionLevelCanMonitor,
		WarehousePermissionLevelCanUse,
		WarehousePermissionLevelCanView,
		WarehousePermissionLevelIsOwner,
	}
}

// Permission level
type VectorSearchEndpointPermissionLevel string

const (
	VectorSearchEndpointPermissionLevelCanCreate VectorSearchEndpointPermissionLevel = "CAN_CREATE"
	VectorSearchEndpointPermissionLevelCanManage VectorSearchEndpointPermissionLevel = "CAN_MANAGE"
	VectorSearchEndpointPermissionLevelCanUse    VectorSearchEndpointPermissionLevel = "CAN_USE"
)

// Values returns all valid VectorSearchEndpointPermissionLevel values.
func (VectorSearchEndpointPermissionLevel) Values() []VectorSearchEndpointPermissionLevel {
	return []VectorSearchEndpointPermissionLevel{
		VectorSearchEndpointPermissionLevelCanCreate,
		VectorSearchEndpointPermissionLevelCanManage,
		VectorSearchEndpointPermissionLevelCanUse,
	}
}

// Permission level
type PermissionLevel string

const (
	PermissionLevelCanAttachTo                 PermissionLevel = "CAN_ATTACH_TO"
	PermissionLevelCanBind                     PermissionLevel = "CAN_BIND"
	PermissionLevelCanCreate                   PermissionLevel = "CAN_CREATE"
	PermissionLevelCanCreateApp                PermissionLevel = "CAN_CREATE_APP"
	PermissionLevelCanEdit                     PermissionLevel = "CAN_EDIT"
	PermissionLevelCanEditMetadata             PermissionLevel = "CAN_EDIT_METADATA"
	PermissionLevelCanManage                   PermissionLevel = "CAN_MANAGE"
	PermissionLevelCanManageProductionVersions PermissionLevel = "CAN_MANAGE_PRODUCTION_VERSIONS"
	PermissionLevelCanManageRun                PermissionLevel = "CAN_MANAGE_RUN"
	PermissionLevelCanManageStagingVersions    PermissionLevel = "CAN_MANAGE_STAGING_VERSIONS"
	PermissionLevelCanMonitor                  PermissionLevel = "CAN_MONITOR"
	PermissionLevelCanMonitorOnly              PermissionLevel = "CAN_MONITOR_ONLY"
	PermissionLevelCanQuery                    PermissionLevel = "CAN_QUERY"
	PermissionLevelCanRead                     PermissionLevel = "CAN_READ"
	PermissionLevelCanRestart                  PermissionLevel = "CAN_RESTART"
	PermissionLevelCanRun                      PermissionLevel = "CAN_RUN"
	PermissionLevelCanUse                      PermissionLevel = "CAN_USE"
	PermissionLevelCanView                     PermissionLevel = "CAN_VIEW"
	PermissionLevelCanViewMetadata             PermissionLevel = "CAN_VIEW_METADATA"
	PermissionLevelIsOwner                     PermissionLevel = "IS_OWNER"
)

// Values returns all valid PermissionLevel values.
func (PermissionLevel) Values() []PermissionLevel {
	return []PermissionLevel{
		PermissionLevelCanAttachTo,
		PermissionLevelCanBind,
		PermissionLevelCanCreate,
		PermissionLevelCanCreateApp,
		PermissionLevelCanEdit,
		PermissionLevelCanEditMetadata,
		PermissionLevelCanManage,
		PermissionLevelCanManageProductionVersions,
		PermissionLevelCanManageRun,
		PermissionLevelCanManageStagingVersions,
		PermissionLevelCanMonitor,
		PermissionLevelCanMonitorOnly,
		PermissionLevelCanQuery,
		PermissionLevelCanRead,
		PermissionLevelCanRestart,
		PermissionLevelCanRun,
		PermissionLevelCanUse,
		PermissionLevelCanView,
		PermissionLevelCanViewMetadata,
		PermissionLevelIsOwner,
	}
}
