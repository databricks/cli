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
	// StateSerial and StateLineage are the state file's own monotonic counter
	// and its system-generated lineage UUID (uuid.New(), never user input).
	// StateSource is "local" or "remote". StatesAvailableCount is how many of
	// the four candidate state files (direct/terraform x local/remote) were
	// found; 0 means the run proceeded against a synthesized empty state.
	// No state paths and no target names are recorded.
	StateSerial          int64  `json:"state_serial,omitempty"`
	StateLineage         string `json:"state_lineage,omitempty"`
	StateSource          string `json:"state_source,omitempty"`
	StatesAvailableCount int64  `json:"states_available_count,omitempty"`

	// Number of --select-ids selectors passed in, and how many resolved to a
	// deployed resource in the state above. Zero matched with a non-zero count
	// is the hard-failure case; a shortfall between them means some selectors
	// were skipped as stale.
	SelectorCount        int64 `json:"selector_count,omitempty"`
	SelectorMatchedCount int64 `json:"selector_matched_count,omitempty"`

	// Scrubbed, truncated summary of the failure when the command exits with an
	// error. Privileged free-text (DATA_LABEL_USER_COMMANDS_RESPONSE, LPP-5543);
	// stays in-region and is stripped from centralized logfood. Unset on success.
	ErrorMessage string `json:"error_message,omitempty"`

	// Category of the failure when the command exits with an error.
	// Unset on success.
	ErrorCategory BundleConfigRemoteSyncErrorCategory `json:"error_category,omitempty"`
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
