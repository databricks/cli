package events

type ExecutionContext struct {
	// Lowercase string name for the operating system. Same value
	// as the one set in `runtime.GOOS` in Golang.
	OperatingSystem string `json:"operating_system,omitempty"`

	// Version of DBR from which CLI is being run.
	// Only set when the CLI is being run from a Databricks cluster.
	DbrVersion string `json:"dbr_version,omitempty"`

	// If true, the CLI is being run from a Databricks notebook / cluster web terminal.
	FromWebTerminal bool `json:"from_web_terminal,omitempty"`

	// UUID generated by the CLI for every CLI command run. This is also set in the HTTP user
	// agent under the key "cmd-exec-id" and can be used to correlate frontend_log table
	// with the http_access_log table.
	CommandExecutionId string `json:"command_exec_id,omitempty"`

	// Total execution time for the CLI command
	ExecutionTimeMs int64 `json:"execution_time_ms,omitempty"`

	// Exit code of the CLI command
	ExitCode int64 `json:"exit_code,omitempty"`
}
