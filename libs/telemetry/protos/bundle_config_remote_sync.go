package protos

type BundleConfigRemoteSyncErrorCategory string

const (
	BundleConfigRemoteSyncErrorCategoryUnspecified         BundleConfigRemoteSyncErrorCategory = "TYPE_UNSPECIFIED"
	BundleConfigRemoteSyncErrorCategoryBundleLoadFailed    BundleConfigRemoteSyncErrorCategory = "BUNDLE_LOAD_FAILED"
	BundleConfigRemoteSyncErrorCategoryStateNotFound       BundleConfigRemoteSyncErrorCategory = "STATE_NOT_FOUND"
	BundleConfigRemoteSyncErrorCategoryDetectChangesFailed BundleConfigRemoteSyncErrorCategory = "DETECT_CHANGES_FAILED"
	BundleConfigRemoteSyncErrorCategoryResolveFailed       BundleConfigRemoteSyncErrorCategory = "RESOLVE_FAILED"
	BundleConfigRemoteSyncErrorCategoryYamlApplyFailed     BundleConfigRemoteSyncErrorCategory = "YAML_APPLY_FAILED"
	BundleConfigRemoteSyncErrorCategorySaveFailed          BundleConfigRemoteSyncErrorCategory = "SAVE_FAILED"
	BundleConfigRemoteSyncErrorCategoryOutputFailed        BundleConfigRemoteSyncErrorCategory = "OUTPUT_FAILED"
)

// BundleConfigRemoteSyncEvent is emitted on every execution of the
// `databricks bundle config-remote-sync` command.
//
// All fields are aggregate counts, booleans, or system-defined categories.
// No resource names, keys, field paths, file paths, or configuration values
// are logged.
type BundleConfigRemoteSyncEvent struct {
	// Whether the command was invoked with --save (config files written to
	// disk) as opposed to diff-only mode.
	Save bool `json:"save,omitempty"`

	// Deployment engine the state was read from: "direct" or "terraform".
	Engine string `json:"engine,omitempty"`

	// Total number of field-level changes detected between the deployed state
	// and the current remote state, across all resources.
	ChangesTotal int64 `json:"changes_total,omitempty"`

	// Number of detected changes by operation type.
	AddCount     int64 `json:"add_count,omitempty"`
	ReplaceCount int64 `json:"replace_count,omitempty"`
	RemoveCount  int64 `json:"remove_count,omitempty"`

	// Number of detected changes on fields the resource lifecycle metadata
	// marks recreate_on_changes (immutable): syncing these into config means the
	// next deploy will delete+recreate the resource (a data-loss risk).
	RecreateForcingChanges int64 `json:"recreate_forcing_changes,omitempty"`

	// Number of detected changes that overwrite a not-yet-deployed local config
	// edit (the local value differs from the last-deployed state). Direct engine
	// only; the terraform sync snapshot has no per-field base to compare against.
	OverwrittenLocalEdits int64 `json:"overwritten_local_edits,omitempty"`

	// One entry per resource type that has at least one detected change.
	ResourceChanges []BundleConfigRemoteSyncResourceChanges `json:"resource_changes,omitempty"`

	// Number of configuration files that would be modified by the detected
	// changes, and the number actually written to disk (--save only).
	FilesChangedCount int64 `json:"files_changed_count,omitempty"`
	FilesWrittenCount int64 `json:"files_written_count,omitempty"`

	// Variable-reference restoration counts for the two mechanisms that can
	// write a current-target-scoped reference into a shared file (the source of
	// the cross-target "reference does not exist" failures).
	RefsRetargeted   int64 `json:"refs_retargeted,omitempty"`
	RefsFromSiblings int64 `json:"refs_from_siblings,omitempty"`

	// Identity of the resource state this run read, so a selector that matched
	// nothing can be attributed: a stale local cache, a different target's or
	// bundle's state, and no state at all are otherwise indistinguishable.
	//
	// StateSerial is the state file's own monotonic counter.
	StateSerial int64 `json:"state_serial,omitempty"`

	// StateLineage is the state's lineage identifier. It cannot contain PII: it
	// is an opaque random UUID minted by dstate.GetOrInitLineage and is never
	// derived from a user, workspace, path, or resource name.
	StateLineage string `json:"state_lineage,omitempty"`

	// StateSource is "local" or "remote" — a closed, system-defined set of
	// exactly two values assigned in CollectStateStats, never user input, so it
	// cannot contain PII.
	StateSource string `json:"state_source,omitempty"`

	// No state paths and no target names are recorded: the target is
	// user-authored configuration, and BundleDeployEvent likewise records only
	// the target count.

	// How many of the four candidate state files (direct/terraform x
	// local/remote) were found. A pointer so that zero — the run proceeded
	// against a synthesized empty state — is emitted rather than dropped by
	// omitempty. Not derivable from StateResourceIDs: a state file holding no
	// resources and no state file at all both yield an empty id set.
	StatesAvailableCount *int64 `json:"states_available_count,omitempty"`

	// IDs of the resources found in the deployment state this run read, and the
	// IDs the run was asked to sync via --select-ids. Comparing the two is what
	// classifies a selector miss; the counts are intentionally not sent
	// separately since they are len() of these lists, and the matched set is
	// their intersection.
	StateResourceIDs    *BundleConfigRemoteSyncResourceIds `json:"state_resource_ids,omitempty"`
	SelectedResourceIDs *BundleConfigRemoteSyncResourceIds `json:"selected_resource_ids,omitempty"`

	// Scrubbed, truncated summary of the failure when the command exits with an
	// error. Privileged free-text (DATA_LABEL_USER_COMMANDS_RESPONSE, LPP-5543);
	// stays in-region and is stripped from centralized logfood. Unset on success.
	ErrorMessage string `json:"error_message,omitempty"`

	// Category of the failure when the command exits with an error.
	// Unset on success.
	ErrorCategory BundleConfigRemoteSyncErrorCategory `json:"error_category,omitempty"`
}

// BundleConfigRemoteSyncResourceIds holds resource IDs grouped by type. The
// same shape is used for the state's IDs and for the selected IDs so the two
// can be compared directly.
//
// IDs of resources managed by the bundle. Some resources like volumes or schemas
// do not expose a numerical or UUID identifier and are tracked by name. Those
// resources are not tracked here since the names are PII.
type BundleConfigRemoteSyncResourceIds struct {
	ResourceJobIDs       []string `json:"resource_job_ids,omitempty"`
	ResourcePipelineIDs  []string `json:"resource_pipeline_ids,omitempty"`
	ResourceClusterIDs   []string `json:"resource_cluster_ids,omitempty"`
	ResourceDashboardIDs []string `json:"resource_dashboard_ids,omitempty"`
}

// BundleConfigRemoteSyncResourceChanges holds field-level change counts for a
// single resource type within one config-remote-sync run.
type BundleConfigRemoteSyncResourceChanges struct {
	// Resource type name, e.g. "jobs", "pipelines", "dashboards".
	ResourceType string `json:"resource_type,omitempty"`

	ChangesCount int64 `json:"changes_count,omitempty"`
	AddCount     int64 `json:"add_count,omitempty"`
	ReplaceCount int64 `json:"replace_count,omitempty"`
	RemoveCount  int64 `json:"remove_count,omitempty"`
}
