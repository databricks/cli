package phases

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrubForTelemetry_BundleRootPath(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		bundleRoot string
		expected   string
	}{
		{
			// Bundle root is replaced with "." and then the resulting
			// ./databricks.yml is caught by the relative path scrubber.
			name:       "replaces bundle root in file path",
			msg:        "failed to load /home/user/project/databricks.yml: invalid config",
			bundleRoot: "/home/user/project",
			expected:   "failed to load [REDACTED_REL_PATH]: invalid config",
		},
		{
			// Bundle root without trailing separator is caught by the
			// absolute path regex.
			name:       "replaces bundle root without trailing content",
			msg:        "error at /home/user/project",
			bundleRoot: "/home/user/project",
			expected:   "error at [REDACTED_PATH]",
		},
		{
			name:       "replaces multiple occurrences",
			msg:        "path /home/user/project/a.yml and /home/user/project/b.yml",
			bundleRoot: "/home/user/project",
			expected:   "path [REDACTED_REL_PATH] and [REDACTED_REL_PATH]",
		},
		{
			name:       "empty bundle root is no-op",
			msg:        "some error",
			bundleRoot: "",
			expected:   "some error",
		},
		{
			name:       "empty message",
			msg:        "",
			bundleRoot: "/home/user/project",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, tt.bundleRoot, ""))
		})
	}
}

func TestScrubForTelemetry_HomeDir(t *testing.T) {
	tests := []struct {
		name       string
		msg        string
		bundleRoot string
		homeDir    string
		expected   string
	}{
		{
			// Home dir is replaced with ~ and then ~/.databricks/config.json
			// is caught by the relative path scrubber.
			name:     "replaces home dir and scrubs resulting path",
			msg:      "failed to read /Users/shreyas/.databricks/config.json",
			homeDir:  "/Users/shreyas",
			expected: "failed to read [REDACTED_REL_PATH]",
		},
		{
			name:     "home dir replacement for multiple paths",
			msg:      "error: /Users/shreyas/project/file.yml and /Users/shreyas/.cache/other",
			homeDir:  "/Users/shreyas",
			expected: "error: [REDACTED_REL_PATH] and [REDACTED_REL_PATH]",
		},
		{
			name:       "bundle root takes priority over home dir",
			msg:        "error at /Users/shreyas/project/databricks.yml",
			bundleRoot: "/Users/shreyas/project",
			homeDir:    "/Users/shreyas",
			expected:   "error at [REDACTED_REL_PATH]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, tt.bundleRoot, tt.homeDir))
		})
	}
}

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
			name:     "absolute path in middle of message",
			msg:      "error: /Users/jane/project/a.yml: not found, try again",
			expected: "error: [REDACTED_PATH]: not found, try again",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, "", ""))
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
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, "", ""))
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
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, "", ""))
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
			assert.Equal(t, tt.expected, scrubForTelemetry(tt.msg, "", ""))
		})
	}
}

func TestScrubForTelemetry_Combined(t *testing.T) {
	msg := "failed to load /Users/shreyas/myproject/databricks.yml: " +
		"workspace /Workspace/Users/shreyas@databricks.com/.bundle is invalid, " +
		"also tried /home/other/fallback/config.yml, " +
		"temp at /tmp/bundle-cache/state.json, " +
		"see .databricks/bundle/dev/variable-overrides.json"

	got := scrubForTelemetry(msg, "/Users/shreyas/myproject", "/Users/shreyas")

	assert.Equal(t,
		"failed to load [REDACTED_REL_PATH]: "+
			"workspace [REDACTED_WORKSPACE_PATH] is invalid, "+
			"also tried [REDACTED_PATH], "+
			"temp at [REDACTED_PATH], "+
			"see [REDACTED_REL_PATH]",
		got,
	)
}
