package protos

// This corresponds to the FrontendLog lumberjack proto in universe.
// FrontendLog is the top-level struct for any client-side logs at Databricks.
type FrontendLog struct {
	// A UUID for the log event generated from the CLI.
	FrontendLogEventID string `json:"frontend_log_event_id,omitempty"`

	Entry FrontendLogEntry `json:"entry,omitempty"`
}

type FrontendLogEntry struct {
	DatabricksCliLog DatabricksCliLog `json:"databricks_cli_log,omitempty"`
}

type DatabricksCliLog struct {
	ExecutionContext *ExecutionContext `json:"execution_context,omitempty"`

	CliTestEvent      *CliTestEvent      `json:"cli_test_event,omitempty"`
	BundleInitEvent   *BundleInitEvent   `json:"bundle_init_event,omitempty"`
	BundleDeployEvent *BundleDeployEvent `json:"bundle_deploy_event,omitempty"`
}
