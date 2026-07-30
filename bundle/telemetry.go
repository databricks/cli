package bundle

import (
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// Telemetry holds the boolean feature flags collected during a deploy. Its
// fields become part of the bundle bitmap (one bit per field; the json tag is
// the field path) and are also emitted on the deploy telemetry wire via
// BoolValues.
//
// Most flags are single bits: they are only ever set true when the feature is
// present, or they belong to a one-hot group where the set member already
// encodes the outcome.
//
// A "value-bearing" flag whose false is a real measurement (distinct from "not
// measured") is stored as a True/False pair so the bitmap can represent both
// outcomes. Exactly one of a pair is set per deploy; BoolValues collapses the
// pair back to the single wire key it replaces.
type Telemetry struct {
	// Single-bit flags. Set true only when the feature is present.
	SelectUsed             bool `json:"select_used,omitempty"`
	HasTfOnlyReferences    bool `json:"has_tf_only_references,omitempty"`
	ArtifactsReferenceUsed bool `json:"artifacts_reference_used,omitempty"`

	ConfigHasDoubleDollarBrace    bool `json:"config_has_double_dollar_brace,omitempty"`
	ConfigHasDoubleDollar         bool `json:"config_has_double_dollar,omitempty"`
	ConfigHasBackslashDollarBrace bool `json:"config_has_backslash_dollar_brace,omitempty"`
	ConfigHasBackslashDollar      bool `json:"config_has_backslash_dollar,omitempty"`

	ArtifactDynamicVersionIsSet bool `json:"artifact_dynamic_version_is_set,omitempty"`

	// State-path location: exactly one is set per deploy (one-hot), so each is a
	// single bit.
	StatePathIsShared        bool `json:"state_path_is_shared,omitempty"`
	StatePathInDeployerHome  bool `json:"state_path_in_deployer_home,omitempty"`
	StatePathInOtherUserHome bool `json:"state_path_in_other_user_home,omitempty"`
	StatePathOther           bool `json:"state_path_other,omitempty"`

	// DMS undeclared writer types: independent bits, all false when the deploy is
	// auto-migration compatible.
	DMSUndeclaredDeployingUser    bool `json:"dms_undeclared_deploying_user,omitempty"`
	DMSUndeclaredOtherUser        bool `json:"dms_undeclared_other_user,omitempty"`
	DMSUndeclaredServicePrincipal bool `json:"dms_undeclared_service_principal,omitempty"`
	DMSUndeclaredGroup            bool `json:"dms_undeclared_group,omitempty"`

	// DMS auto-migration verdict: exactly one is set per deploy (one-hot).
	DMSCompatAuto               bool `json:"dms_compat_auto,omitempty"`
	DMSCompatOnlySelfUndeclared bool `json:"dms_compat_only_self_undeclared,omitempty"`
	DMSCompatNot                bool `json:"dms_compat_not,omitempty"`

	// Opt-in auto-migration outcome. Only set on the opt-in path; absent
	// otherwise.
	DirectMigrateError       bool `json:"direct_migrate_error,omitempty"`
	DirectMigrateCommitError bool `json:"direct_migrate_commit_error,omitempty"`
	DirectMigrateWarnings    bool `json:"direct_migrate_warnings,omitempty"`

	// Which opt-in source triggered auto-migration: exactly one set when it ran.
	DirectMigratedViaConfig bool `json:"direct_migrated_via_config,omitempty"`
	DirectMigratedViaEnv    bool `json:"direct_migrated_via_env,omitempty"`

	// Whether the deprecated terraform engine was explicitly opted into, per
	// source. Independent: both can be set. Emitted only when true.
	EngineTerraformConfig bool `json:"engine_terraform_config,omitempty"`
	EngineTerraformEnv    bool `json:"engine_terraform_env,omitempty"`

	// True/False pairs. Exactly one of each pair is set when the flag is
	// measured; neither when it is not. BoolValues collapses each pair to its
	// single wire key.
	RunAsSetTrue  bool `json:"run_as_set_true,omitempty"`
	RunAsSetFalse bool `json:"run_as_set_false,omitempty"`

	UseLegacyRunAsTrue  bool `json:"experimental.use_legacy_run_as_true,omitempty"`
	UseLegacyRunAsFalse bool `json:"experimental.use_legacy_run_as_false,omitempty"`

	HasServerlessComputeTrue  bool `json:"has_serverless_compute_true,omitempty"`
	HasServerlessComputeFalse bool `json:"has_serverless_compute_false,omitempty"`

	HasClassicJobComputeTrue  bool `json:"has_classic_job_compute_true,omitempty"`
	HasClassicJobComputeFalse bool `json:"has_classic_job_compute_false,omitempty"`

	HasClassicInteractiveComputeTrue  bool `json:"has_classic_interactive_compute_true,omitempty"`
	HasClassicInteractiveComputeFalse bool `json:"has_classic_interactive_compute_false,omitempty"`

	PresetsNamePrefixIsSetTrue  bool `json:"presets_name_prefix_is_set_true,omitempty"`
	PresetsNamePrefixIsSetFalse bool `json:"presets_name_prefix_is_set_false,omitempty"`

	PythonWheelWrapperIsSetTrue  bool `json:"python_wheel_wrapper_is_set_true,omitempty"`
	PythonWheelWrapperIsSetFalse bool `json:"python_wheel_wrapper_is_set_false,omitempty"`

	SkipArtifactCleanupTrue  bool `json:"skip_artifact_cleanup_true,omitempty"`
	SkipArtifactCleanupFalse bool `json:"skip_artifact_cleanup_false,omitempty"`

	ArtifactBuildCommandIsSetTrue  bool `json:"artifact_build_command_is_set_true,omitempty"`
	ArtifactBuildCommandIsSetFalse bool `json:"artifact_build_command_is_set_false,omitempty"`

	ArtifactFilesIsSetTrue  bool `json:"artifact_files_is_set_true,omitempty"`
	ArtifactFilesIsSetFalse bool `json:"artifact_files_is_set_false,omitempty"`

	PermissionsSectionSetTrue  bool `json:"permissions_section_set_true,omitempty"`
	PermissionsSectionSetFalse bool `json:"permissions_section_set_false,omitempty"`

	SourceLinkedSetForNonDevelopmentTrue  bool `json:"source_linked_set_for_non_development_true,omitempty"`
	SourceLinkedSetForNonDevelopmentFalse bool `json:"source_linked_set_for_non_development_false,omitempty"`

	AppLifecycleStartedTrue  bool `json:"app_lifecycle_started_true,omitempty"`
	AppLifecycleStartedFalse bool `json:"app_lifecycle_started_false,omitempty"`

	ClusterLifecycleStartedTrue  bool `json:"cluster_lifecycle_started_true,omitempty"`
	ClusterLifecycleStartedFalse bool `json:"cluster_lifecycle_started_false,omitempty"`

	SqlWarehouseLifecycleStartedTrue  bool `json:"sql_warehouse_lifecycle_started_true,omitempty"`
	SqlWarehouseLifecycleStartedFalse bool `json:"sql_warehouse_lifecycle_started_false,omitempty"`

	DirectDryMigrateSuccessTrue  bool `json:"direct_drymigrate_success_true,omitempty"`
	DirectDryMigrateSuccessFalse bool `json:"direct_drymigrate_success_false,omitempty"`

	DirectDryMigrateWarningsTrue  bool `json:"direct_drymigrate_warnings_true,omitempty"`
	DirectDryMigrateWarningsFalse bool `json:"direct_drymigrate_warnings_false,omitempty"`

	// Local-cache flags. Embedded so cache.Metrics fields flatten into the
	// bitmap; libs/cache writes them directly (it cannot import this package).
	cache.Metrics
}

// SetPaired sets exactly one bool of a True/False pair from a measured value,
// e.g. b.Metrics.Telemetry.SetPaired(&t.RunAsSetTrue, &t.RunAsSetFalse, value).
func (t *Telemetry) SetPaired(vTrue, vFalse *bool, value bool) {
	if value {
		*vTrue = true
	} else {
		*vFalse = true
	}
}

// BoolValues returns the telemetry flags as wire entries. Single-bit flags emit
// one entry each only when set (matching the historical "absent unless set"
// behavior). Each True/False pair collapses to its single wire key: emitted as
// true/false when either half is set, omitted when neither is.
func (t *Telemetry) BoolValues() []protos.BoolMapEntry {
	var out []protos.BoolMapEntry

	single := func(key string, v bool) {
		if v {
			out = append(out, protos.BoolMapEntry{Key: key, Value: true})
		}
	}
	single("select_used", t.SelectUsed)
	single("has_tf_only_references", t.HasTfOnlyReferences)
	single("artifacts_reference_used", t.ArtifactsReferenceUsed)
	single("config_has_double_dollar_brace", t.ConfigHasDoubleDollarBrace)
	single("config_has_double_dollar", t.ConfigHasDoubleDollar)
	single("config_has_backslash_dollar_brace", t.ConfigHasBackslashDollarBrace)
	single("config_has_backslash_dollar", t.ConfigHasBackslashDollar)
	single("artifact_dynamic_version_is_set", t.ArtifactDynamicVersionIsSet)
	single("state_path_is_shared", t.StatePathIsShared)
	single("state_path_in_deployer_home", t.StatePathInDeployerHome)
	single("state_path_in_other_user_home", t.StatePathInOtherUserHome)
	single("state_path_other", t.StatePathOther)
	single("dms_undeclared_deploying_user", t.DMSUndeclaredDeployingUser)
	single("dms_undeclared_other_user", t.DMSUndeclaredOtherUser)
	single("dms_undeclared_service_principal", t.DMSUndeclaredServicePrincipal)
	single("dms_undeclared_group", t.DMSUndeclaredGroup)
	single("dms_compat_auto", t.DMSCompatAuto)
	single("dms_compat_only_self_undeclared", t.DMSCompatOnlySelfUndeclared)
	single("dms_compat_not", t.DMSCompatNot)
	single("direct_migrate_error", t.DirectMigrateError)
	single("direct_migrate_commit_error", t.DirectMigrateCommitError)
	single("direct_migrate_warnings", t.DirectMigrateWarnings)
	single("direct_migrated_via_config", t.DirectMigratedViaConfig)
	single("direct_migrated_via_env", t.DirectMigratedViaEnv)
	single("engine_terraform_config", t.EngineTerraformConfig)
	single("engine_terraform_env", t.EngineTerraformEnv)

	single("local.cache.attempt", t.Attempt)
	single("local.cache.hit", t.Hit)
	single("local.cache.miss", t.Miss)
	single("local.cache.error", t.Error)

	collapse := func(key string, vTrue, vFalse bool) {
		if vTrue {
			out = append(out, protos.BoolMapEntry{Key: key, Value: true})
		} else if vFalse {
			out = append(out, protos.BoolMapEntry{Key: key, Value: false})
		}
	}
	collapse("run_as_set", t.RunAsSetTrue, t.RunAsSetFalse)
	collapse("experimental.use_legacy_run_as", t.UseLegacyRunAsTrue, t.UseLegacyRunAsFalse)
	collapse("has_serverless_compute", t.HasServerlessComputeTrue, t.HasServerlessComputeFalse)
	collapse("has_classic_job_compute", t.HasClassicJobComputeTrue, t.HasClassicJobComputeFalse)
	collapse("has_classic_interactive_compute", t.HasClassicInteractiveComputeTrue, t.HasClassicInteractiveComputeFalse)
	collapse("presets_name_prefix_is_set", t.PresetsNamePrefixIsSetTrue, t.PresetsNamePrefixIsSetFalse)
	collapse("python_wheel_wrapper_is_set", t.PythonWheelWrapperIsSetTrue, t.PythonWheelWrapperIsSetFalse)
	collapse("skip_artifact_cleanup", t.SkipArtifactCleanupTrue, t.SkipArtifactCleanupFalse)
	collapse("artifact_build_command_is_set", t.ArtifactBuildCommandIsSetTrue, t.ArtifactBuildCommandIsSetFalse)
	collapse("artifact_files_is_set", t.ArtifactFilesIsSetTrue, t.ArtifactFilesIsSetFalse)
	collapse("permissions_section_set", t.PermissionsSectionSetTrue, t.PermissionsSectionSetFalse)
	collapse("source_linked_set_for_non_development", t.SourceLinkedSetForNonDevelopmentTrue, t.SourceLinkedSetForNonDevelopmentFalse)
	collapse("app_lifecycle_started", t.AppLifecycleStartedTrue, t.AppLifecycleStartedFalse)
	collapse("cluster_lifecycle_started", t.ClusterLifecycleStartedTrue, t.ClusterLifecycleStartedFalse)
	collapse("sql_warehouse_lifecycle_started", t.SqlWarehouseLifecycleStartedTrue, t.SqlWarehouseLifecycleStartedFalse)
	collapse("direct_drymigrate_success", t.DirectDryMigrateSuccessTrue, t.DirectDryMigrateSuccessFalse)
	collapse("direct_drymigrate_warnings", t.DirectDryMigrateWarningsTrue, t.DirectDryMigrateWarningsFalse)

	return out
}
