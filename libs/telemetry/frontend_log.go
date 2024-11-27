package telemetry

// This corresponds to the FrontendLog lumberjack proto in universe.
type FrontendLog struct {
	// A unique identifier for the log event generated from the CLI.
	FrontendLogEventID string `json:"frontend_log_event_id,omitempty"`

	Entry FrontendLogEntry `json:"entry,omitempty"`
}

type FrontendLogEntry struct {
	DatabricksCliLog DatabricksCliLog `json:"databricks_cli_log,omitempty"`
}

type DatabricksCliLog struct {
	CliTestEvent CliTestEvent `json:"cli_test_event,omitempty"`
}

// dummy event for testing the telemetry pipeline
type CliTestEvent struct {
	Name DummyCliEnum `json:"name,omitempty"`
}

type DummyCliEnum string

const (
	DummyCliEnumUnspecified DummyCliEnum = "DUMMY_CLI_ENUM_UNSPECIFIED"
	DummyCliEnumValue1      DummyCliEnum = "VALUE1"
	DummyCliEnumValue2      DummyCliEnum = "VALUE2"
	DummyCliEnumValue3      DummyCliEnum = "VALUE3"
)
