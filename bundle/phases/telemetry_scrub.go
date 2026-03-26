package phases

import "regexp"

// Scrub sensitive information from error messages before sending to telemetry.
// Inspired by VS Code's telemetry path scrubbing and Sentry's @userpath pattern.
//
// References:
// - VS Code: https://github.com/microsoft/vscode/blob/main/src/vs/platform/telemetry/common/telemetryUtils.ts
// - Sentry: https://github.com/getsentry/relay (PII rule: @userpath)
var (
	// Matches Windows absolute paths with at least two components
	// (e.g., C:\foo\bar, D:/projects/myapp).
	windowsAbsPathRegexp = regexp.MustCompile(`[A-Za-z]:[/\\][^\s:,"'/\\]+[/\\][^\s:,"']+`)

	// Matches Databricks workspace paths (/Workspace/...).
	workspacePathRegexp = regexp.MustCompile(`(^|[\s:,"'])(/Workspace/[^\s:,"']+)`)

	// Matches absolute Unix paths with at least two components
	// (e.g., /home/user/..., /tmp/foo).
	absPathRegexp = regexp.MustCompile(`(^|[\s:,"'])(/[^\s:,"'/]+/[^\s:,"']+)`)

	// Matches relative paths:
	// - Explicit: ./foo, ../foo
	// - Dot-prefixed directories: .databricks/bundle/..., .cache/foo
	// - Home shorthand: ~/.databricks/...
	explicitRelPathRegexp = regexp.MustCompile(`(^|[\s:,"'])((?:~|\.\.?|\.[a-zA-Z][^\s:,"'/]*)/[^\s:,"']+)`)

	// Matches implicit relative paths: at least two path components where
	// the last component has a file extension (e.g., "resources/job.yml",
	// "bundle/dev/state.json").
	implicitRelPathRegexp = regexp.MustCompile(`(^|[\s:,"'])([a-zA-Z0-9_][^\s:,"']*/[^\s:,"']*\.[a-zA-Z][^\s:,"']*)`)

	// Matches email addresses. Workspace paths in Databricks often contain
	// emails (e.g., /Workspace/Users/user@example.com/.bundle/dev).
	emailRegexp = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

// scrubForTelemetry is a best-effort scrubber that removes sensitive path and
// PII information from error messages before they are sent to telemetry.
// The error message is treated as PII by the logging infrastructure but we
// scrub to avoid collecting more information than necessary.
func scrubForTelemetry(msg string) string {
	// Redact absolute paths.
	msg = windowsAbsPathRegexp.ReplaceAllString(msg, "[REDACTED_PATH]")
	msg = workspacePathRegexp.ReplaceAllString(msg, "${1}[REDACTED_WORKSPACE_PATH]")
	msg = absPathRegexp.ReplaceAllString(msg, "${1}[REDACTED_PATH]")

	// Redact relative paths.
	msg = explicitRelPathRegexp.ReplaceAllString(msg, "${1}[REDACTED_REL_PATH]")
	msg = implicitRelPathRegexp.ReplaceAllString(msg, "${1}[REDACTED_REL_PATH]")

	// Redact email addresses.
	msg = emailRegexp.ReplaceAllString(msg, "[REDACTED_EMAIL]")

	return msg
}
