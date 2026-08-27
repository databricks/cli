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

	// Outcome of the dry-run migration to the direct engine attempted after a
	// successful terraform deploy WHEN THE USER OPTED OUT of direct (direct is
	// the default, so this means engine: terraform). Only recorded when the state
	// conversion is truly a dry run (no auto-migrate).
	// DirectDryMigrateSuccess is false when the state could not be converted;
	// DirectDryMigrateWarnings is true when the conversion emitted warnings
	// (e.g. resources the direct engine can't represent).
	DirectDryMigrateSuccess  = "direct_drymigrate_success"
	DirectDryMigrateWarnings = "direct_drymigrate_warnings"

	// Outcome of an automatic post-deploy migration to the direct engine, which
	// runs unless the user opted out with engine: terraform. These replace the
	// direct_drymigrate_* keys on migrating deploys.
	//   - migrate_error:        state conversion itself errored.
	//   - migrate_commit_error: the state was converted, but committing it
	//                           (renaming files / pushing to workspace) failed.
	//   - migrate_warnings:     the conversion emitted warnings (see above).
	DirectMigrateError       = "direct_migrate_error"
	DirectMigrateCommitError = "direct_migrate_commit_error"
	DirectMigrateWarnings    = "direct_migrate_warnings"

	// Recorded when an automatic post-deploy migration to the direct engine
	// actually ran (state was rewritten). Exactly one of the three keys is true;
	// all are absent when auto-migration did not run. If both config and env
	// set direct, ConfigType wins per ResolveEngineSetting, so via_config
	// covers the "durable opt-in" population and via_env covers the
	// "env-var only" population.
	//   - via_config:  bundle.engine = "direct" was set in the bundle config.
	//   - via_env:     only DATABRICKS_BUNDLE_ENGINE=direct was set.
	//   - via_default: neither was set, so the migration came from the default.
	//                  This is the population that did not ask for anything.
	DirectAutoMigrateViaConfig  = "direct_migrated_via_config"
	DirectAutoMigrateViaEnv     = "direct_migrated_via_env"
	DirectAutoMigrateViaDefault = "direct_migrated_via_default"

	// Whether the (deprecated) terraform engine was explicitly opted into, and via
	// which source. Independent: both can be present when config and env both
	// request terraform. Emitted only when true; absence means "not opted in via
	// this source", so the population still on terraform can be sliced by source.
	//   - engine_terraform_config: bundle.engine = "terraform" in the bundle config.
	//   - engine_terraform_env:    DATABRICKS_BUNDLE_ENGINE=terraform in the environment.
	EngineTerraformConfig = "engine_terraform_config"
	EngineTerraformEnv    = "engine_terraform_env"

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

	// Breakdown dimensions recorded on every deploy alongside the verdict above, so the
	// DMS auto-migration population can be sliced without inferring it from the verdict.
	// Each is an independent boolean.

	// Whether a top-level permissions section is set. The no-permissions case is always
	// auto-migration compatible (folder ACLs are mirrored), so this separates the two
	// populations that both land on dms_compat_auto.
	PermissionsSectionSet = "permissions_section_set"

	// Where the deployment state folder lives. Exactly one of StatePathIsShared,
	// StatePathInDeployerHome, StatePathInOtherUserHome, and StatePathOther is true per
	// deploy. StatePathOther is any other /Workspace folder (not a user home or shared).
	StatePathInDeployerHome  = "state_path_in_deployer_home"
	StatePathInOtherUserHome = "state_path_in_other_user_home"
	StatePathOther           = "state_path_other"

	// Which principal types have undeclared write access to the state folder — the
	// access an auto-migration governed by the permissions section would drop. These can
	// co-occur; all false when the deploy is auto-migration compatible.
	DMSUndeclaredDeployingUser    = "dms_undeclared_deploying_user"
	DMSUndeclaredOtherUser        = "dms_undeclared_other_user"
	DMSUndeclaredServicePrincipal = "dms_undeclared_service_principal"
	DMSUndeclaredGroup            = "dms_undeclared_group"
)
