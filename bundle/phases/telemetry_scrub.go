package phases

import (
	"path"
	"regexp"
	"strings"
)

// Scrub sensitive information from error messages before sending to telemetry.
// Inspired by VS Code's telemetry path scrubbing and Sentry's @userpath pattern.
//
// Path regexes use [\s:,"'] as boundary characters to delimit where a path
// ends. While these characters are technically valid in file paths, in error
// messages they act as delimiters (e.g. "error: /path/to/file: not found",
// or "failed to read '/some/path', skipping"). This is a practical tradeoff:
// paths containing colons, commas, or quotes are extremely rare, and without
// these boundaries the regexes would over-match into surrounding message text.
//
// References:
// - VS Code: https://github.com/microsoft/vscode/blob/main/src/vs/platform/telemetry/common/telemetryUtils.ts
// - Sentry: https://github.com/getsentry/relay (PII rule: @userpath)
var (
	// Matches Windows absolute paths with backslashes and at least two components
	// (e.g., C:\foo\bar, D:\Users\project).
	windowsBackslashPathRegexp = regexp.MustCompile(`[A-Za-z]:\\[^\s:,"'/\\]+\\[^\s:,"']+`)

	// Matches Windows absolute paths with forward slashes and at least two components
	// (e.g., C:/foo/bar, D:/Users/project).
	windowsFwdslashPathRegexp = regexp.MustCompile(`[A-Za-z]:/[^\s:,"'/\\]+/[^\s:,"']+`)

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
// These help understand usage patterns without capturing sensitive information.
var knownExtensions = map[string]bool{
	// Configuration and data formats
	".yml":        true,
	".yaml":       true,
	".json":       true,
	".toml":       true,
	".cfg":        true,
	".ini":        true,
	".env":        true,
	".xml":        true,
	".properties": true,
	".conf":       true,

	// Notebook and script languages
	".py":    true,
	".r":     true,
	".scala": true,
	".sql":   true,
	".ipynb": true,
	".sh":    true,

	// Web / Apps
	".js":   true,
	".ts":   true,
	".jsx":  true,
	".tsx":  true,
	".html": true,
	".css":  true,

	// Terraform
	".tf":      true,
	".hcl":     true,
	".tfstate": true,
	".tfvars":  true,

	// Build artifacts and archives
	".whl": true,
	".jar": true,
	".egg": true,
	".zip": true,
	".tar": true,
	".gz":  true,
	".tgz": true,
	".dbc": true,

	// Data formats
	".txt":     true,
	".csv":     true,
	".md":      true,
	".parquet": true,
	".avro":    true,

	// Logs and locks
	".log":  true,
	".lock": true,

	// Certificates and keys
	".pem": true,
	".crt": true,
}

// scrubForTelemetry is a best-effort scrubber that removes sensitive path and
// PII information from error messages before they are sent to telemetry.
// The error message is treated as PII by the logging infrastructure but we
// scrub to avoid collecting more information than necessary.
func scrubForTelemetry(msg string) string {
	// Redact absolute paths.
	msg = replacePathRegexp(msg, windowsBackslashPathRegexp, "[REDACTED_WIN_PATH]", false)
	msg = replacePathRegexp(msg, windowsFwdslashPathRegexp, "[REDACTED_WIN_FPATH]", false)
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
