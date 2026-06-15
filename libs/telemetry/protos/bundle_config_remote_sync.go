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

	// Number of detected remote string values that contain the literal
	// character sequence "${". The values themselves are not logged.
	RawValuesWithVarSyntax int64 `json:"raw_values_with_var_syntax,omitempty"`

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
