package protos

// SkillsCommandEvent mirrors SkillsCommandEvent in the databricks_cli lumberjack proto: one event per
// `databricks skills` command run. The calling agent is ExecutionContext.CallingAgent, not a field here.
type SkillsCommandEvent struct {
	Command SkillsCommand `json:"command,omitempty"`

	// SkillName is set only for a successful read or read-file of a published skill.
	SkillName string `json:"skill_name,omitempty"`

	// CatalogReleaseVersion is <major>.<minor>.<patch> with no leading "v", or unset.
	CatalogReleaseVersion string `json:"catalog_release_version,omitempty"`

	Source SkillsSource `json:"source,omitempty"`

	Outcome SkillsCommandOutcome `json:"outcome,omitempty"`

	// ErrorCategory is set for a failed outcome only.
	ErrorCategory SkillsErrorCategory `json:"error_category,omitempty"`
}

// SkillsCommand mirrors SkillsCommand.Type.
type SkillsCommand string

const (
	SkillsCommandUnspecified SkillsCommand = "TYPE_UNSPECIFIED"
	SkillsCommandList        SkillsCommand = "LIST"
	SkillsCommandRead        SkillsCommand = "READ"
	SkillsCommandListFiles   SkillsCommand = "LIST_FILES"
	SkillsCommandReadFile    SkillsCommand = "READ_FILE"
	SkillsCommandSearch      SkillsCommand = "SEARCH"
)

// SkillsSource mirrors SkillsSource.Type.
type SkillsSource string

const (
	SkillsSourceUnspecified SkillsSource = "TYPE_UNSPECIFIED"
	SkillsSourceLive        SkillsSource = "LIVE"
	SkillsSourceCache       SkillsSource = "CACHE"
	SkillsSourceBundled     SkillsSource = "BUNDLED"
	SkillsSourceFolder      SkillsSource = "FOLDER"
)

// SkillsCommandOutcome mirrors SkillsCommandOutcome.Type.
type SkillsCommandOutcome string

const (
	SkillsCommandOutcomeUnspecified SkillsCommandOutcome = "TYPE_UNSPECIFIED"
	SkillsCommandOutcomeSucceeded   SkillsCommandOutcome = "SUCCEEDED"
	SkillsCommandOutcomeFailed      SkillsCommandOutcome = "FAILED"
)

// SkillsErrorCategory mirrors SkillsErrorCategory.Type. Only the categories the CLI sends are listed here.
type SkillsErrorCategory string

const (
	SkillsErrorCategoryUnspecified       SkillsErrorCategory = "TYPE_UNSPECIFIED"
	SkillsErrorCategoryUncategorized     SkillsErrorCategory = "UNCATEGORIZED"
	SkillsErrorCategorySkillNotFound     SkillsErrorCategory = "SKILL_NOT_FOUND"
	SkillsErrorCategoryFileNotFound      SkillsErrorCategory = "FILE_NOT_FOUND"
	SkillsErrorCategoryCatalogInvalid    SkillsErrorCategory = "CATALOG_INVALID"
	SkillsErrorCategorySourceUnavailable SkillsErrorCategory = "SOURCE_UNAVAILABLE"
	SkillsErrorCategoryPermissionDenied  SkillsErrorCategory = "PERMISSION_DENIED"
	SkillsErrorCategoryFilesystem        SkillsErrorCategory = "FILESYSTEM"
)
