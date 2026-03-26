package phases

import (
	"path"
	"regexp"
	"strings"
)

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
	// (e.g., /home/user/..., /tmp/foo, ~/.config/databricks).
	absPathRegexp = regexp.MustCompile(`(^|[\s:,"'])(~?/[^\s:,"'/]+/[^\s:,"']+)`)

	// Matches relative paths:
	// - Explicit: ./foo, ../foo
	// - Dot-prefixed directories: .databricks/bundle/..., .cache/foo
	explicitRelPathRegexp = regexp.MustCompile(`(^|[\s:,"'])((?:\.\.?|\.[a-zA-Z][^\s:,"'/]*)/[^\s:,"']+)`)

	// Matches implicit relative paths: at least two path components where
	// the last component has a file extension (e.g., "resources/job.yml",
	// "bundle/dev/state.json").
	implicitRelPathRegexp = regexp.MustCompile(`(^|[\s:,"'])([a-zA-Z0-9_][^\s:,"']*/[^\s:,"']*\.[a-zA-Z][^\s:,"']*)`)

	// Matches email addresses. Workspace paths in Databricks often contain
	// emails (e.g., /Workspace/Users/user@example.com/.bundle/dev).
	emailRegexp = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

// Known file extensions that are safe to retain in redacted paths.
// These help with debugging without leaking sensitive information.
var knownExtensions = map[string]bool{
	// Configuration and data formats
	".yml":  true,
	".yaml": true,
	".json": true,
	".toml": true,
	".cfg":  true,
	".ini":  true,

	// Notebook and script languages
	".py":    true,
	".r":     true,
	".scala": true,
	".sql":   true,
	".ipynb": true,
	".sh":    true,

	// Terraform
	".tf":    true,
	".hcl":   true,

	// Build artifacts and archives
	".whl": true,
	".jar": true,
	".zip": true,
	".tar": true,

	// Other
	".txt": true,
	".csv": true,
}

// scrubForTelemetry is a best-effort scrubber that removes sensitive path and
// PII information from error messages before they are sent to telemetry.
// The error message is treated as PII by the logging infrastructure but we
// scrub to avoid collecting more information than necessary.
func scrubForTelemetry(msg string) string {
	// Redact absolute paths.
	msg = replacePathRegexp(msg, windowsAbsPathRegexp, "[REDACTED_PATH]", false)
	msg = replacePathRegexp(msg, workspacePathRegexp, "[REDACTED_WORKSPACE_PATH]", true)
	msg = replacePathRegexp(msg, absPathRegexp, "[REDACTED_PATH]", true)

	// Redact relative paths.
	msg = replacePathRegexp(msg, explicitRelPathRegexp, "[REDACTED_REL_PATH]", true)
	msg = replacePathRegexp(msg, implicitRelPathRegexp, "[REDACTED_REL_PATH]", true)

	// Redact email addresses.
	msg = emailRegexp.ReplaceAllString(msg, "[REDACTED_EMAIL]")

	return msg
}

// replacePathRegexp replaces path matches with the given label, retaining
// known file extensions. When hasDelimiterGroup is true, the first character
// of the match is preserved as a delimiter prefix.
func replacePathRegexp(msg string, re *regexp.Regexp, label string, hasDelimiterGroup bool) string {
	return re.ReplaceAllStringFunc(msg, func(match string) string {
		prefix := ""
		p := match
		if hasDelimiterGroup && len(match) > 0 {
			first := match[0]
			if strings.ContainsRune(" \t\n:,\"'", rune(first)) {
				prefix = match[:1]
				p = match[1:]
			}
		}

		ext := path.Ext(p)
		if knownExtensions[ext] {
			return prefix + label + "(" + ext[1:] + ")"
		}
		return prefix + label
	})
}
