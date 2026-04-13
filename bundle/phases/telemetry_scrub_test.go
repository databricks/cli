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
			name:     "tmp path with known extension",
			msg:      "failed to write /tmp/bundle-xyz/state.json",
			expected: "failed to write [REDACTED_PATH](json)",
		},
		{
			name:     "var folders path without extension",
			msg:      "error reading /var/folders/7t/n_tz6x9d4xj91h48pf8md5zh0000gp/T/test123/file",
			expected: "error reading [REDACTED_PATH]",
		},
		{
			name:     "etc path with known extension",
			msg:      "config at /etc/databricks/config.json not found",
			expected: "config at [REDACTED_PATH](json) not found",
		},
		{
			name:     "macOS home path with known extension",
			msg:      "failed to read /Users/otheruser/some-project/file.yml",
			expected: "failed to read [REDACTED_PATH](yml)",
		},
		{
			name:     "Linux home path with known extension",
			msg:      "failed to read /home/runner/work/project/file.yml",
			expected: "failed to read [REDACTED_PATH](yml)",
		},
		{
			name:     "absolute path after colon delimiter",
			msg:      "error: /Users/jane/project/a.yml: not found, try again",
			expected: "error: [REDACTED_PATH](yml): not found, try again",
		},
		{
			name:     "multiple absolute paths",
			msg:      "path /home/user/project/a.yml and /home/user/project/b.yml",
			expected: "path [REDACTED_PATH](yml) and [REDACTED_PATH](yml)",
		},
		{
			name:     "Windows path with backslashes",
			msg:      `error at C:\Users\shreyas\project\file.yml`,
			expected: "error at [REDACTED_WIN_PATH](yml)",
		},
		{
			name:     "Windows path with forward slashes",
			msg:      "error at C:/Users/shreyas/project/file.yml",
			expected: "error at [REDACTED_WIN_FPATH](yml)",
		},
		{
			name:     "Windows path with lowercase drive letter",
			msg:      `error at c:\Users\shreyas\project\file.yml`,
			expected: "error at [REDACTED_WIN_PATH](yml)",
		},
		{
			name:     "volume path with known extension",
			msg:      "artifact at /Volumes/catalog/schema/volume/artifact.whl",
			expected: "artifact at [REDACTED_PATH](whl)",
		},
		{
			name:     "dbfs path with known extension",
			msg:      "state at /dbfs/mnt/data/state.json",
			expected: "state at [REDACTED_PATH](json)",
		},
		{
			name:     "path with unknown extension",
			msg:      "error at /home/user/project/file.xyz",
			expected: "error at [REDACTED_PATH]",
		},
		{
			name:     "tilde home path with known extension",
			msg:      "reading ~/.databricks/config.json failed",
			expected: "reading [REDACTED_PATH](json) failed",
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
			name:     "workspace user path without extension",
			msg:      "uploading to /Workspace/Users/dev/.bundle/files",
			expected: "uploading to [REDACTED_WORKSPACE_PATH]",
		},
		{
			name:     "workspace path with email",
			msg:      "error at /Workspace/Users/user@example.com/.bundle/dev",
			expected: "error at [REDACTED_WORKSPACE_PATH]",
		},
		{
			name:     "workspace shared path without extension",
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
			expected: "failed to load [REDACTED_REL_PATH](yml)",
		},
		{
			name:     "explicit relative path with ../",
			msg:      "path ../parent/file.yml not allowed",
			expected: "path [REDACTED_REL_PATH](yml) not allowed",
		},
		{
			name:     "implicit relative path with extension",
			msg:      "failed to read resources/pipeline.yml: not found",
			expected: "failed to read [REDACTED_REL_PATH](yml): not found",
		},
		{
			name:     "dot-prefixed directory path with extension",
			msg:      "error reading .databricks/bundle/dev/variable-overrides.json",
			expected: "error reading [REDACTED_REL_PATH](json)",
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

	expected := "failed to load [REDACTED_PATH](yml): " +
		"workspace [REDACTED_WORKSPACE_PATH] is invalid, " +
		"also tried [REDACTED_PATH](yml), " +
		"temp at [REDACTED_PATH](json), " +
		"see [REDACTED_REL_PATH](json)"

	assert.Equal(t, expected, scrubForTelemetry(msg))
}
