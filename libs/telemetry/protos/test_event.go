package protos

// dummy event for testing the telemetry pipeline. Corresponds to `DatabricksCliTestEvent`
// proto in `databricks_cli_log.proto` as of 20 Dec 2024.
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
