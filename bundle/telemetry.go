package bundle

import (
	"reflect"

	"github.com/databricks/cli/libs/telemetry/protos"
)

// Telemetry holds the boolean flags reported with a deploy. It is the single
// source of the deploy event's bool_values (see BoolValues) and of the
// telemetry.* bits of the bundle bitmap (see bundle/bitmap). It lives on Bundle,
// not on the config, so users cannot set it.
//
// The field type says what was measured:
//   - bool: a flag reported only when true; absence means "not seen".
//   - *bool: a measured value; nil when the measurement did not run.
//   - []bool: one value per measurement (e.g. per artifact), or each value seen.
//
// The `metric` tag is the key in bool_values. Fields have no json tags: the
// bitmap uses the Go field names (e.g. "telemetry.SelectUsed").
//
// Not here: the has_<task_type> keys, which the bitmap already records as config
// fields (resources.jobs.*.tasks[*].<type>_task), and the local.cache.* keys,
// which describe the local user cache rather than the bundle. Both are written
// to Metrics.BoolValues directly.
type Telemetry struct {
	PythonWheelWrapperIsSet     *bool  `metric:"python_wheel_wrapper_is_set"`
	ArtifactDynamicVersionIsSet []bool `metric:"artifact_dynamic_version_is_set"`
	ArtifactBuildCommandIsSet   []bool `metric:"artifact_build_command_is_set"`
	ArtifactFilesIsSet          []bool `metric:"artifact_files_is_set"`
	PresetsNamePrefixIsSet      *bool  `metric:"presets_name_prefix_is_set"`
	// The *LifecycleStarted fields hold the lifecycle.started values set across
	// the resources of that type, each value once: true if any resource sets
	// true, false if any sets false.
	AppLifecycleStarted          []bool `metric:"app_lifecycle_started"`
	ClusterLifecycleStarted      []bool `metric:"cluster_lifecycle_started"`
	DeploymentHistoryEnabled     *bool  `metric:"deployment_history_enabled"`
	SqlWarehouseLifecycleStarted []bool `metric:"sql_warehouse_lifecycle_started"`
	SelectUsed                   bool   `metric:"select_used"`

	// Whether the deploy registered an immutable snapshot. Derived from the snapshot
	// resource, not from experimental.immutable_folder: the flag may change, and the
	// resource is what the deploy actually uploads.
	ImmutableSnapshotUsed *bool `metric:"immutable_snapshot_used"`

	// Outcome of an automatic migration of terraform state to the direct engine.
	//   - migrate_error:        state conversion itself errored.
	//   - migrate_warnings:     the conversion emitted warnings (e.g. resources the direct engine can't represent).
	DirectMigrateError    bool `metric:"direct_migrate_error"`
	DirectMigrateWarnings bool `metric:"direct_migrate_warnings"`
	// True when the post-convert plan check failed; the migration was not committed.
	DirectMigratePlanError bool `metric:"direct_migrate_plan_error"`
	// True when the migrated state's first plan would recreate (destroy + create) an
	// existing resource. The migration still commits; the recreate is a genuine pending
	// change gated by the deploy's --auto-approve, and this only records how often it happens.
	DirectMigrateRecreatePlanned bool `metric:"direct_migrate_recreate_planned"`

	// Recorded when an automatic post-deploy migration to the direct engine
	// actually ran (state was rewritten). Exactly one of the three keys is true;
	// all are absent when auto-migration did not run. If both config and env
	// set direct, bundle.engine wins per ValidateEngineSetting, so via_config
	// covers the "durable opt-in" population and via_env covers the
	// "env-var only" population.
	//   - via_config:  bundle.engine = "direct" was set in the bundle config.
	//   - via_env:     only DATABRICKS_BUNDLE_ENGINE=direct was set.
	//   - via_default: neither was set, so the migration came from the default.
	//                  This is the population that did not ask for anything.
	DirectAutoMigrateViaConfig  bool `metric:"direct_migrated_via_config"`
	DirectAutoMigrateViaEnv     bool `metric:"direct_migrated_via_env"`
	DirectAutoMigrateViaDefault bool `metric:"direct_migrated_via_default"`

	// Whether the (deprecated) terraform engine was explicitly opted into, and via
	// which source. Independent: both can be present when config and env both
	// request terraform. Emitted only when true; absence means "not opted in via
	// this source", so the population still on terraform can be sliced by source.
	//   - engine_terraform_config: bundle.engine = "terraform" in the bundle config.
	//   - engine_terraform_env:    DATABRICKS_BUNDLE_ENGINE=terraform in the environment.
	EngineTerraformConfig bool `metric:"engine_terraform_config"`
	EngineTerraformEnv    bool `metric:"engine_terraform_env"`

	// Whether workspace.state_path is under /Workspace/Shared.
	StatePathIsShared *bool `metric:"state_path_is_shared"`

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
	DMSCompatAuto               bool `metric:"dms_compat_auto"`
	DMSCompatOnlySelfUndeclared bool `metric:"dms_compat_only_self_undeclared"`
	DMSCompatNot                bool `metric:"dms_compat_not"`

	// Breakdown dimensions recorded on every deploy alongside the verdict above, so the
	// DMS auto-migration population can be sliced without inferring it from the verdict.
	// Each is an independent boolean.

	// Whether a top-level permissions section is set. The no-permissions case is always
	// auto-migration compatible (folder ACLs are mirrored), so this separates the two
	// populations that both land on dms_compat_auto.
	PermissionsSectionSet *bool `metric:"permissions_section_set"`

	// Where the deployment state folder lives. Exactly one of StatePathIsShared,
	// StatePathInDeployerHome, StatePathInOtherUserHome, and StatePathOther is true per
	// deploy. StatePathOther is any other /Workspace folder (not a user home or shared).
	StatePathInDeployerHome  *bool `metric:"state_path_in_deployer_home"`
	StatePathInOtherUserHome *bool `metric:"state_path_in_other_user_home"`
	StatePathOther           *bool `metric:"state_path_other"`

	// Which principal types have undeclared write access to the state folder — the
	// access an auto-migration governed by the permissions section would drop. These can
	// co-occur; all false when the deploy is auto-migration compatible.
	DMSUndeclaredDeployingUser    *bool `metric:"dms_undeclared_deploying_user"`
	DMSUndeclaredOtherUser        *bool `metric:"dms_undeclared_other_user"`
	DMSUndeclaredServicePrincipal *bool `metric:"dms_undeclared_service_principal"`
	DMSUndeclaredGroup            *bool `metric:"dms_undeclared_group"`

	// Task-type usage is recorded generically at deploy as one "has_<task_type>" key
	// per task type present in the bundle (e.g. has_notebook_task, has_ai_runtime_task);
	// see collectTaskTypes in bundle/phases. The two keys below are extra
	// ai_runtime_task-specific dimensions, emitted only when the bundle declares an
	// ai_runtime_task, so a false value means "has an ai_runtime_task, but it is not
	// scheduled / not multi-task". GPU type and count are intentionally not recorded here.
	AiRuntimeTaskScheduled *bool `metric:"ai_runtime_task_scheduled"`
	AiRuntimeTaskMultitask *bool `metric:"ai_runtime_task_multitask"`

	SkipArtifactCleanup              *bool `metric:"skip_artifact_cleanup"`
	SourceLinkedSetForNonDevelopment *bool `metric:"source_linked_set_for_non_development"`
	ConfigHasDoubleDollarBrace       bool  `metric:"config_has_double_dollar_brace"`
	ConfigHasDoubleDollar            bool  `metric:"config_has_double_dollar"`
	ConfigHasBackslashDollarBrace    bool  `metric:"config_has_backslash_dollar_brace"`
	ConfigHasBackslashDollar         bool  `metric:"config_has_backslash_dollar"`
	ArtifactsReferenceUsed           bool  `metric:"artifacts_reference_used"`
	UseLegacyRunAs                   *bool `metric:"experimental.use_legacy_run_as"`
	RunAsSet                         *bool `metric:"run_as_set"`
	HasTfOnlyReferences              bool  `metric:"has_tf_only_references"`
	HasServerlessCompute             *bool `metric:"has_serverless_compute"`
	HasClassicJobCompute             *bool `metric:"has_classic_job_compute"`
	HasClassicInteractiveCompute     *bool `metric:"has_classic_interactive_compute"`
}

// BoolValues returns the flags as deploy-event entries, in field order: a true
// bool, the value of a non-nil *bool, and every element of a []bool.
func (t *Telemetry) BoolValues() []protos.BoolMapEntry {
	var out []protos.BoolMapEntry
	v := reflect.ValueOf(t).Elem()
	for i := range v.NumField() {
		key := v.Type().Field(i).Tag.Get("metric")
		switch f := v.Field(i).Interface().(type) {
		case bool:
			if f {
				out = append(out, protos.BoolMapEntry{Key: key, Value: true})
			}
		case *bool:
			if f != nil {
				out = append(out, protos.BoolMapEntry{Key: key, Value: *f})
			}
		case []bool:
			for _, value := range f {
				out = append(out, protos.BoolMapEntry{Key: key, Value: value})
			}
		}
	}
	return out
}
