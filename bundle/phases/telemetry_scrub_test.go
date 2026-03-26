package phases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubForTelemetry_AbsolutePaths(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "tmp path",
			msg:      "failed to write /tmp/bundle-xyz/state.json",
			expected: "failed to write [REDACTED_PATH]",
		},
		{
			name:     "var folders path",
			msg:      "error reading /var/folders/7t/n_tz6x9d4xj91h48pf8md5zh0000gp/T/test123/file",
			expected: "error reading [REDACTED_PATH]",
		},
		{
			name:     "etc path",
			msg:      "config at /etc/databricks/config.json not found",
			expected: "config at [REDACTED_PATH] not found",
		},
		{
			name:     "macOS home path",
			msg:      "failed to read /Users/otheruser/some-project/file.yml",
			expected: "failed to read [REDACTED_PATH]",
		},
		{
			name:     "Linux home path",
			msg:      "failed to read /home/runner/work/project/file.yml",
			expected: "failed to read [REDACTED_PATH]",
		},
		{
			name:     "absolute path after colon delimiter",
			msg:      "error: /Users/jane/project/a.yml: not found, try again",
			expected: "error: [REDACTED_PATH]: not found, try again",
		},
		{
			name:     "multiple absolute paths",
			msg:      "path /home/user/project/a.yml and /home/user/project/b.yml",
			expected: "path [REDACTED_PATH] and [REDACTED_PATH]",
		},
		{
			name:     "Windows path with backslashes",
			msg:      `error at C:\Users\shreyas\project\file.yml`,
			expected: "error at [REDACTED_PATH]",
		},
		{
			name:     "Windows path with forward slashes",
			msg:      "error at C:/Users/shreyas/project/file.yml",
			expected: "error at [REDACTED_PATH]",
		},
		{
			name:     "Windows path with lowercase drive letter",
			msg:      `error at c:\Users\shreyas\project\file.yml`,
			expected: "error at [REDACTED_PATH]",
		},
		{
			name:     "volume path is redacted",
			msg:      "artifact at /Volumes/catalog/schema/volume/artifact.whl",
			expected: "artifact at [REDACTED_PATH]",
		},
		{
			name:     "dbfs path is redacted",
			msg:      "state at /dbfs/mnt/data/state.json",
			expected: "state at [REDACTED_PATH]",
		},
		{
			name:     "single component path is not matched",
			msg:      "POST /telemetry-ext failed",
			expected: "POST /telemetry-ext failed",
		},
		{
			name:     "empty message",
			msg:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg))
		})
	}
}

func TestScrubForTelemetry_WorkspacePaths(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "workspace user path",
			msg:      "uploading to /Workspace/Users/dev/.bundle/files",
			expected: "uploading to [REDACTED_WORKSPACE_PATH]",
		},
		{
			name:     "workspace path with email",
			msg:      "error at /Workspace/Users/user@example.com/.bundle/dev",
			expected: "error at [REDACTED_WORKSPACE_PATH]",
		},
		{
			name:     "workspace shared path",
			msg:      "cannot access /Workspace/Shared/project/notebook",
			expected: "cannot access [REDACTED_WORKSPACE_PATH]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg))
		})
	}
}

func TestScrubForTelemetry_RelativePaths(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "explicit relative path with ./",
			msg:      "failed to load ./resources/job.yml",
			expected: "failed to load [REDACTED_REL_PATH]",
		},
		{
			name:     "explicit relative path with ../",
			msg:      "path ../parent/file.yml not allowed",
			expected: "path [REDACTED_REL_PATH] not allowed",
		},
		{
			name:     "implicit relative path with extension",
			msg:      "failed to read resources/pipeline.yml: not found",
			expected: "failed to read [REDACTED_REL_PATH]: not found",
		},
		{
			name:     "dot-prefixed directory path",
			msg:      "error reading .databricks/bundle/dev/variable-overrides.json",
			expected: "error reading [REDACTED_REL_PATH]",
		},
		{
			name:     "tilde home path",
			msg:      "reading ~/.databricks/config.json failed",
			expected: "reading [REDACTED_REL_PATH] failed",
		},
		{
			name:     "single filename without path separator is preserved",
			msg:      "failed to load databricks.yml",
			expected: "failed to load databricks.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg))
		})
	}
}

func TestScrubForTelemetry_Emails(t *testing.T) {
	tests := []struct {
		name     string
		msg      string
		expected string
	}{
		{
			name:     "email in message",
			msg:      "access denied for user@example.com in workspace",
			expected: "access denied for [REDACTED_EMAIL] in workspace",
		},
		{
			name:     "no email present",
			msg:      "some error without emails",
			expected: "some error without emails",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg))
		})
	}
}

func TestScrubForTelemetry_Combined(t *testing.T) {
	msg := "failed to load /Users/shreyas/myproject/databricks.yml: " +
		"workspace /Workspace/Users/shreyas@databricks.com/.bundle is invalid, " +
		"also tried /home/other/fallback/config.yml, " +
		"temp at /tmp/bundle-cache/state.json, " +
		"see .databricks/bundle/dev/variable-overrides.json"

	expected := "failed to load [REDACTED_PATH]: " +
		"workspace [REDACTED_WORKSPACE_PATH] is invalid, " +
		"also tried [REDACTED_PATH], " +
		"temp at [REDACTED_PATH], " +
		"see [REDACTED_REL_PATH]"

	assert.Equal(t, expected, scrubForTelemetry(msg))
}
