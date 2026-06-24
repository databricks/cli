package metrics

const (
	ExperimentalPythonWheelWrapperIsSet = "python_wheel_wrapper_is_set"
	ArtifactDynamicVersionIsSet         = "artifact_dynamic_version_is_set"
	ArtifactBuildCommandIsSet           = "artifact_build_command_is_set"
	ArtifactFilesIsSet                  = "artifact_files_is_set"
	PresetsNamePrefixIsSet              = "presets_name_prefix_is_set"
	AppLifecycleStarted                 = "app_lifecycle_started"
	ClusterLifecycleStarted             = "cluster_lifecycle_started"
	SqlWarehouseLifecycleStarted        = "sql_warehouse_lifecycle_started"
	SelectUsed                          = "select_used"
	LocalUsed                           = "local_used"

	// Whether workspace.state_path is under /Workspace/Shared.
	StatePathIsShared = "state_path_is_shared"

	// Whether this deploy is compatible with an automatic migration of the deployment
	// state to a dedicated state storage service (DMS). Deploying a bundle requires
	// write access (CAN_EDIT or higher) to the state folder; after migration that is
	// governed by the permissions on the deployment object instead.
	//
	// When the bundle has no permissions section, the migration can mirror the state
	// folder's ACLs onto the deployment (CAN_EDIT -> CAN_EDIT, CAN_MANAGE ->
	// CAN_MANAGE), preserving everyone's access wherever the state lives. When a
	// permissions section is set, the migration applies exactly those permissions, so
	// anyone with write access to the state folder who is not declared with
	// CAN_MANAGE would lose the ability to deploy.
	//
	// Exactly one of the three keys below is recorded per deploy:
	//   - auto: no permissions section (folder ACLs are mirrored), or every principal
	//           with write access to the state folder is declared.
	//   - only_self_undeclared: a permissions section is set and the only principal
	//           with undeclared write access is the deploying user. The migration
	//           grants the deploying user CAN_MANAGE on the deployment object, so this
	//           is auto-migratable if we choose to preserve that grant on future
	//           deploys. Recorded separately to measure how common this case is.
	//   - not:  a permissions section is set and the state folder has undeclared write
	//           access from a principal other than the deploying user.
	DMSCompatAuto               = "dms_compat_auto"
	DMSCompatOnlySelfUndeclared = "dms_compat_only_self_undeclared"
	DMSCompatNot                = "dms_compat_not"
)
