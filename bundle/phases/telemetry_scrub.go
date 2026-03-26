package phases

import (
	"path/filepath"
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
	// Matches home directory paths on macOS and Linux.
	// The leading delimiter check avoids matching workspace paths like
	// /Workspace/Users/... where /Users is not a top-level component.
	unixHomeDirRegexp = regexp.MustCompile(`(?:^|[\s:,"'])(/(?:Users|home)/[^\s:,"']+)`)

	// Matches home directory paths on Windows with either backslashes or
	// forward slashes (C:\Users\xxx\... or C:/Users/xxx/...).
	windowsHomeDirRegexp = regexp.MustCompile(`[A-Z]:[/\\]Users[/\\][^\s:,"']+`)

	// Matches absolute Unix paths with at least two components
	// (e.g., /tmp/foo, /Workspace/Users/..., /Volumes/catalog/schema/...).
	absPathRegexp = regexp.MustCompile(`(?:^|[\s:,"'])(/[^\s:,"'/]+/[^\s:,"']+)`)

	// Matches relative paths:
	// - Explicit: ./foo, ../foo
	// - Dot-prefixed directories: .databricks/bundle/..., .cache/foo
	// - Home shorthand: ~/.databricks/...
	explicitRelPathRegexp = regexp.MustCompile(`(?:^|[\s:,"'])((?:~|\.\.?|\.[a-zA-Z][^\s:,"'/]*)/[^\s:,"']+)`)

	// Matches implicit relative paths: at least two path components where
	// the last component has a file extension (e.g., "resources/job.yml",
	// "bundle/dev/state.json").
	implicitRelPathRegexp = regexp.MustCompile(`(?:^|[\s:,"'])([a-zA-Z0-9_][^\s:,"']*/[^\s:,"']*\.[a-zA-Z][^\s:,"']*)`)

	// Matches email addresses. Workspace paths in Databricks often contain
	// emails (e.g., /Workspace/Users/user@example.com/.bundle/dev).
	emailRegexp = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
)

// scrubForTelemetry is a best-effort scrubber that removes sensitive path and
// PII information from error messages before they are sent to telemetry.
// The error message is treated as PII by the logging infrastructure but we
// scrub to avoid collecting more information than necessary.
func scrubForTelemetry(msg, bundleRoot, homeDir string) string {
	// Replace the bundle root path first since it's the most specific match.
	// This turns "/Users/shreyas/project/databricks.yml" into "./databricks.yml".
	if bundleRoot != "" {
		msg = replacePath(msg, bundleRoot, ".")
	}

	// Replace the user's home directory. This catches paths outside the
	// bundle root like "/Users/shreyas/.databricks/..." → "~/.databricks/...".
	if homeDir != "" {
		msg = replacePath(msg, homeDir, "~")
	}

	// Regex fallback: redact remaining home directory paths not covered by the
	// direct home dir replacement above (e.g., paths from other users or
	// non-standard home directory locations).
	// Run Windows first to avoid partial matches from the Unix regex on
	// paths like C:/Users/...
	msg = windowsHomeDirRegexp.ReplaceAllString(msg, "[REDACTED_PATH]")
	msg = replaceDelimitedMatch(msg, unixHomeDirRegexp, "[REDACTED_PATH]")

	// Redact all remaining absolute paths.
	msg = replaceDelimitedMatch(msg, absPathRegexp, "[REDACTED_PATH]")

	// Redact relative paths.
	msg = replaceDelimitedMatch(msg, explicitRelPathRegexp, "[REDACTED_REL_PATH]")
	msg = replaceDelimitedMatch(msg, implicitRelPathRegexp, "[REDACTED_REL_PATH]")

	// Redact email addresses.
	msg = emailRegexp.ReplaceAllString(msg, "[REDACTED_EMAIL]")

	return msg
}

// replacePath replaces all occurrences of a directory path with the given
// replacement. It only replaces when the path appears as a complete prefix,
// i.e., followed by `/`, a delimiter, or end of string. This prevents partial
// matches like "/Users/shreyas" matching inside "/Workspace/Users/shreyas@...".
func replacePath(msg, path, replacement string) string {
	normalized := filepath.ToSlash(path)
	for _, p := range []string{normalized, path} {
		msg = strings.ReplaceAll(msg, p+"/", replacement+"/")

		// Replace occurrences not followed by '/' only when the path is at
		// a word boundary (followed by delimiter or end of string).
		result := strings.Builder{}
		for {
			idx := strings.Index(msg, p)
			if idx == -1 {
				result.WriteString(msg)
				break
			}
			after := idx + len(p)
			// Check the character after the match. Only replace if it's
			// a delimiter or end of string.
			if after == len(msg) || strings.ContainsRune(" \t\n:,\"'", rune(msg[after])) {
				result.WriteString(msg[:idx])
				result.WriteString(replacement)
				msg = msg[after:]
			} else {
				result.WriteString(msg[:after])
				msg = msg[after:]
			}
		}
		msg = result.String()
	}
	return msg
}

const delimiters = " \t\n:,\"'"

// replaceDelimitedMatch replaces paths matched by a regex that uses a leading
// delimiter group `(?:^|[\s:,"'])`. The optional delimiter character is
// preserved and only the path itself is replaced.
func replaceDelimitedMatch(msg string, re *regexp.Regexp, replacement string) string {
	return re.ReplaceAllStringFunc(msg, func(match string) string {
		if len(match) == 0 {
			return match
		}
		// If the first character is a delimiter, preserve it.
		if strings.ContainsRune(delimiters, rune(match[0])) {
			return match[:1] + replacement
		}
		// Otherwise the match starts at ^ and the whole match is the path.
		return replacement
	})
}
