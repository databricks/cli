package telemetry

import "github.com/databricks/cli/libs/telemetry/events"

// This corresponds to the FrontendLog lumberjack proto in universe.
// FrontendLog is the top-level struct for any client-side logs at Databricks
// regardless of whether they are generated from the CLI or the web UI.
type FrontendLog struct {
	// A unique identifier for the log event generated from the CLI.
	FrontendLogEventID string `json:"frontend_log_event_id,omitempty"`

	Entry FrontendLogEntry `json:"entry,omitempty"`
}

type FrontendLogEntry struct {
	DatabricksCliLog DatabricksCliLog `json:"databricks_cli_log,omitempty"`
}

type DatabricksCliLog struct {
	CliTestEvent    *events.CliTestEvent     `json:"cli_test_event,omitempty"`
	BundleInitEvent *events.BundleInitEvent `json:"bundle_init_event,omitempty"`
}
