package localenv

import (
	"fmt"
	"regexp"
	"strings"
)

// managedMarkerStart and managedMarkerEnd bracket the region of pyproject.toml that
// "databricks local-env python sync" owns. Everything between them is rewritten on each merge;
// everything outside is preserved byte-for-byte.
const (
	managedMarkerStart = "# managed by databricks local-env python sync — do not edit"
	managedMarkerEnd   = "# end managed by databricks local-env python sync"
)

// Region names reported back to the caller via MergeManaged's regions return value.
const (
	regionRequiresPython    = "requires-python"
	regionDatabricksConnect = "databricks-connect"
	regionToolUv            = "tool.uv.constraint-dependencies"
)

var (
	// tableHeaderRe matches a TOML table header line such as "[project]" or "[tool.uv]".
	tableHeaderRe = regexp.MustCompile(`^\s*\[[^\]]+\]\s*$`)
	// requiresPythonRe captures the leading whitespace of a requires-python assignment so it
	// can be preserved when the value is replaced.
	requiresPythonRe = regexp.MustCompile(`^(\s*)requires-python\s*=`)
)

// MergeManaged applies the three managed transforms to target, preserving every other
// byte (comments, ordering, whitespace). It returns the merged bytes and the list of
// regions that actually changed. The operation is idempotent: feeding its own output
// back in produces identical bytes.
func MergeManaged(target []byte, c Constraints) (merged []byte, regions []string, err error) {
	s := string(target)

	// Detect and normalize line endings. We process on "\n" and restore "\r\n" on exit.
	crlf := strings.Contains(s, "\r\n")
	if crlf {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}

	lines := strings.Split(s, "\n")

	lines, rpChanged := mergeRequiresPython(lines, c.RequiresPython)
	if rpChanged {
		regions = append(regions, regionRequiresPython)
	}

	lines, dbcChanged := mergeDatabricksConnect(lines, c.DatabricksConnect)
	if dbcChanged {
		regions = append(regions, regionDatabricksConnect)
	}

	lines, uvChanged := mergeToolUv(lines, c.ConstraintDeps)
	if uvChanged {
		regions = append(regions, regionToolUv)
	}

	out := strings.Join(lines, "\n")
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return []byte(out), regions, nil
}

// tableBounds returns the line index of the header matching name (e.g. "[project]") and
// the index of the first line after the table body (the next table header or EOF). If the
// table is absent, found is false.
func tableBounds(lines []string, name string) (header, end int, found bool) {
	header = -1
	for i, line := range lines {
		if strings.TrimSpace(line) == name {
			header = i
			break
		}
	}
	if header == -1 {
		return -1, -1, false
	}
	end = len(lines)
	for i := header + 1; i < len(lines); i++ {
		if tableHeaderRe.MatchString(lines[i]) {
			end = i
			break
		}
	}
	return header, end, true
}

// mergeRequiresPython replaces the value of requires-python within [project], preserving
// the line's leading whitespace. If the key is absent, it is inserted directly under the
// [project] header. Returns whether the line slice changed.
func mergeRequiresPython(lines []string, value string) ([]string, bool) {
	header, end, found := tableBounds(lines, "[project]")
	if !found {
		return lines, false
	}

	want := func(indent, comment string) string {
		return fmt.Sprintf(`%srequires-python = "%s"%s`, indent, value, comment)
	}

	for i := header + 1; i < end; i++ {
		m := requiresPythonRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		// Preserve a trailing inline comment; only the value is managed, so
		// "requires-python = \"...\" # note" keeps its note.
		replacement := want(m[1], trailingComment(lines[i]))
		if lines[i] == replacement {
			return lines, false
		}
		lines[i] = replacement
		return lines, true
	}

	// Key absent: insert directly under the [project] header.
	inserted := make([]string, 0, len(lines)+1)
	inserted = append(inserted, lines[:header+1]...)
	inserted = append(inserted, want("", ""))
	inserted = append(inserted, lines[header+1:]...)
	return inserted, true
}

// trailingComment returns the inline TOML comment suffix of a line (including the
// leading whitespace and "#"), or "" if there is none. It ignores "#" characters
// inside a quoted string so a value like requires-python = ">=3.10 # x" is not
// mistaken for a comment.
func trailingComment(line string) string {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '#':
			// Include any whitespace immediately preceding the "#".
			start := i
			for start > 0 && (line[start-1] == ' ' || line[start-1] == '\t') {
				start--
			}
			return line[start:]
		}
	}
	return ""
}

// dbconnectLineRe captures, for a line holding a databricks-connect dependency element:
// (1) the leading whitespace, and (3) any trailing comma (with optional trailing space),
// so that indentation and comma style are preserved when the quoted token is replaced.
var dbconnectLineRe = regexp.MustCompile(`^(\s*)"databricks-connect[^"]*"(\s*,?\s*)$`)

// devKeyRe matches the start of the dev array assignment within [dependency-groups]
// (e.g. "dev = [" or "dev=["), capturing leading whitespace. Only this key is
// managed; sibling groups such as test/docs are user-owned and left untouched.
var devKeyRe = regexp.MustCompile(`^\s*dev\s*=`)

// mergeDatabricksConnect replaces the databricks-connect element inside
// [dependency-groups].dev only. It handles both the multi-line array form (one
// element per line) and the single-line array form (dev = ["databricks-connect~=..."]).
// An empty value (constraints-only mode) is a no-op: the user's dev group is left
// untouched rather than having its databricks-connect pin blanked out.
//
// The rewrite is scoped to the dev array's own span (found via bracket depth), so a
// databricks-connect pin sitting in a sibling group (e.g. docs/test) or inside a
// trailing comment on some other line is never clobbered.
func mergeDatabricksConnect(lines []string, value string) ([]string, bool) {
	if value == "" {
		return lines, false
	}
	header, end, found := tableBounds(lines, "[dependency-groups]")
	if !found {
		return lines, false
	}

	// Locate the dev assignment and the line span of its array value.
	devStart := -1
	for i := header + 1; i < end; i++ {
		if devKeyRe.MatchString(lines[i]) {
			devStart = i
			break
		}
	}
	if devStart == -1 {
		return lines, false
	}
	arrayLast, _ := arrayLineSpan(lines, devStart, end)

	// Single-line form: the whole array is on the dev line itself. Only rewrite
	// within the array (through its closing "]"); a trailing comment after it is
	// user content and must be left byte-for-byte intact.
	if devStart == arrayLast {
		line := lines[devStart]
		arrayPart, commentPart := splitAtArrayClose(line)
		if !strings.Contains(arrayPart, `"databricks-connect`) {
			return lines, false
		}
		replaced := dbconnectTokenRe.ReplaceAllString(arrayPart, `"`+value+`"`) + commentPart
		if replaced == line {
			return lines, false
		}
		lines[devStart] = replaced
		return lines, true
	}

	// Multi-line form: one element per line, between the dev line and the closing "]".
	for i := devStart + 1; i <= arrayLast; i++ {
		if m := dbconnectLineRe.FindStringSubmatch(lines[i]); m != nil {
			replacement := fmt.Sprintf(`%s"%s"%s`, m[1], value, m[2])
			if lines[i] == replacement {
				return lines, false
			}
			lines[i] = replacement
			return lines, true
		}
	}
	return lines, false
}

// splitAtArrayClose splits a single-line array assignment into the part up to and
// including the array's closing "]" and the remainder (a trailing comment, if any),
// tracking bracket depth outside quoted strings. This keeps token replacement inside
// the array from touching a trailing comment. If no balanced close is found the whole
// line is returned as the array part.
func splitAtArrayClose(line string) (arrayPart, rest string) {
	depth := 0
	var quote byte
	opened := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '[':
			depth++
			opened = true
		case ']':
			depth--
			if opened && depth == 0 {
				return line[:i+1], line[i+1:]
			}
		}
	}
	return line, ""
}

// arrayLineSpan returns the index of the line on which the array opened at
// lines[start] closes (brackets balance), scanning outside strings/comments. A
// single-line array returns start. It bounds in-place edits of an array value to
// the array's own lines.
func arrayLineSpan(lines []string, start, limit int) (last int, multiline bool) {
	depth := bracketDepthDelta(lines[start])
	if depth <= 0 {
		return start, false
	}
	for j := start + 1; j < limit; j++ {
		depth += bracketDepthDelta(lines[j])
		if depth <= 0 {
			return j, true
		}
	}
	return limit - 1, true
}

// dbconnectTokenRe matches a quoted databricks-connect element anywhere in a line, used
// for the single-line array form.
var dbconnectTokenRe = regexp.MustCompile(`"databricks-connect[^"]*"`)

// mergeToolUv rewrites the managed [tool.uv] constraint-dependencies block. If a
// marker-bracketed block already exists, its contents are replaced in place. Otherwise any
// plain [tool.uv] table is removed and a fresh marker-bracketed block is appended at EOF.
func mergeToolUv(lines, deps []string) ([]string, bool) {
	start, stop, found := markerBounds(lines)
	if found {
		// Replace the existing managed region in place. Whether it owns a [tool.uv]
		// header depends on whether it sits inside a user-authored [tool.uv] table:
		// a header-less region attached to the user table stays header-less on
		// re-merge, so idempotency holds.
		block := renderToolUvBlock(deps, !markerAttachedToToolUv(lines, start))
		existing := lines[start : stop+1]
		if equalLines(existing, block) {
			return lines, false
		}
		out := make([]string, 0, len(lines)-(stop-start+1)+len(block))
		out = append(out, lines[:start]...)
		out = append(out, block...)
		out = append(out, lines[stop+1:]...)
		return out, true
	}

	// No managed block yet: reconcile any plain [tool.uv] table.
	if header, end, ok := tableBounds(lines, "[tool.uv]"); ok {
		if toolUvHasOnlyConstraintDeps(lines, header, end) {
			// The table is effectively ours (only constraint-dependencies, from a
			// pre-marker run): drop it whole and append a fresh standalone block.
			out := make([]string, 0, len(lines))
			out = append(out, lines[:header]...)
			out = append(out, lines[end:]...)
			return appendManagedBlock(out, renderToolUvBlock(deps, true)), true
		}
		// The table holds user-authored keys: strip our stale constraint-dependencies,
		// then insert a header-less managed block INSIDE the existing table. Emitting a
		// second "[tool.uv]" header (as a standalone block would) is invalid TOML —
		// toml.Decode and uv both reject a table defined twice.
		lines = removeConstraintDeps(lines, header, end)
		header, end, _ = tableBounds(lines, "[tool.uv]")
		// Insert after the last non-blank line of the table body so the managed block
		// stays under [tool.uv] and any blank line separating the next table is kept.
		insertAt := end
		for insertAt > header+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
			insertAt--
		}
		block := renderToolUvBlock(deps, false)
		out := make([]string, 0, len(lines)+len(block))
		out = append(out, lines[:insertAt]...)
		out = append(out, block...)
		out = append(out, lines[insertAt:]...)
		return out, true
	}

	// No [tool.uv] at all: append a fresh standalone managed block at EOF.
	return appendManagedBlock(lines, renderToolUvBlock(deps, true)), true
}

// markerAttachedToToolUv reports whether the managed marker region beginning at
// index start sits inside an existing [tool.uv] table — i.e. the nearest table
// header above it is [tool.uv]. When true, the managed block must omit its own
// [tool.uv] header, because a second header for the same table is invalid TOML.
func markerAttachedToToolUv(lines []string, start int) bool {
	for i := start - 1; i >= 0; i-- {
		if tableHeaderRe.MatchString(lines[i]) {
			return strings.TrimSpace(lines[i]) == "[tool.uv]"
		}
	}
	return false
}

// constraintDepsRe matches the start of a constraint-dependencies assignment within a
// [tool.uv] table, capturing its leading whitespace.
var constraintDepsRe = regexp.MustCompile(`^\s*constraint-dependencies\s*=`)

// bracketDepthDelta returns the net change in "[" nesting contributed by line.
// It scans outside TOML strings and stops at an unquoted "#" comment, so a "]"
// inside a string element (e.g. "requests[security]==2.0") or a trailing comment
// does not affect the count. It underpins single- vs multi-line array detection;
// testing strings.Contains(line, "]") instead misreads such lines and corrupts
// the merge.
func bracketDepthDelta(line string) int {
	delta := 0
	var quote byte // 0 = outside a string; otherwise the opening quote rune
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote != 0 {
			if c == '\\' && quote == '"' {
				i++ // skip the escaped char inside a basic string
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '#':
			return delta // comment tail: ignore the rest of the line
		case '[':
			delta++
		case ']':
			delta--
		}
	}
	return delta
}

// constraintDepsArrayEndDelta reports whether the constraint-dependencies array
// opening at lines[start] spans multiple lines, and if so the index of its last
// line. A single-line array (brackets balanced on the opening line) returns
// (start, false).
func constraintDepsArrayEnd(lines []string, start, limit int) (last int, multiline bool) {
	depth := bracketDepthDelta(lines[start])
	if depth <= 0 {
		return start, false
	}
	for j := start + 1; j < limit; j++ {
		depth += bracketDepthDelta(lines[j])
		if depth <= 0 {
			return j, true
		}
	}
	return limit - 1, true
}

// toolUvHasOnlyConstraintDeps reports whether the [tool.uv] table body spanning
// (header, end) contains no meaningful key other than constraint-dependencies. Blank lines
// and comment-only lines are ignored when deciding "only".
func toolUvHasOnlyConstraintDeps(lines []string, header, end int) bool {
	for i := header + 1; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !constraintDepsRe.MatchString(lines[i]) {
			return false
		}
		// Skip the continuation lines of a multi-line array so the whole managed
		// key counts as ignorable (mirrors removeConstraintDeps).
		last, multiline := constraintDepsArrayEnd(lines, i, end)
		if multiline {
			i = last
		}
	}
	return true
}

// removeConstraintDeps strips a constraint-dependencies key from the [tool.uv] table body
// spanning (header, end), leaving the table header and all other user keys in place. It
// handles both the single-line array form and the multi-line array form (the value spans
// several lines until the array's brackets balance).
func removeConstraintDeps(lines []string, header, end int) []string {
	for i := header + 1; i < end; i++ {
		if !constraintDepsRe.MatchString(lines[i]) {
			continue
		}
		last, _ := constraintDepsArrayEnd(lines, i, end)
		end2 := last + 1
		out := make([]string, 0, len(lines)-(end2-i))
		out = append(out, lines[:i]...)
		out = append(out, lines[end2:]...)
		return out
	}
	return lines
}

// markerBounds returns the indices of the managed marker start and end lines, if present.
func markerBounds(lines []string) (start, stop int, found bool) {
	start, stop = -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == managedMarkerStart {
			start = i
			break
		}
	}
	if start == -1 {
		return -1, -1, false
	}
	for i := start + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == managedMarkerEnd {
			stop = i
			break
		}
	}
	if stop == -1 {
		return -1, -1, false
	}
	return start, stop, true
}

// renderToolUvBlock builds the marker-bracketed managed block lines (no surrounding
// blank lines). When withHeader is true it emits its own "[tool.uv]" table header
// (standalone block appended at EOF); when false it omits the header so the block can
// be nested inside a user-authored [tool.uv] table without defining the table twice.
func renderToolUvBlock(deps []string, withHeader bool) []string {
	block := []string{managedMarkerStart}
	if withHeader {
		block = append(block, "[tool.uv]")
	}
	block = append(block, "constraint-dependencies = [")
	for _, d := range deps {
		block = append(block, fmt.Sprintf("    %q,", d))
	}
	block = append(block, "]", managedMarkerEnd)
	return block
}

// appendManagedBlock appends block to lines, ensuring exactly one blank line separates it
// from prior content and the file ends with a single trailing newline.
func appendManagedBlock(lines, block []string) []string {
	// strings.Split on a trailing "\n" leaves a final empty element; drop trailing empty
	// lines so we control the spacing precisely.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	out := make([]string, 0, len(lines)+len(block)+2)
	out = append(out, lines...)
	if len(out) > 0 {
		out = append(out, "") // exactly one blank line before the managed block
	}
	out = append(out, block...)
	out = append(out, "") // trailing newline after final join
	return out
}

// equalLines reports whether two line slices are identical.
func equalLines(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// freshProjectVersion is the placeholder version written into a greenfield
// [project] table. uv rejects a [project] table that has neither project.version
// nor project.dynamic containing "version", even for a non-distributed local
// environment, so a concrete value is required for `uv sync` to succeed.
const freshProjectVersion = "0.0.0"

// RenderFreshPyproject produces a complete managed pyproject.toml for a project that has
// none, with [project], [dependency-groups].dev (carrying the databricks-connect pin), and
// the marker-bracketed [tool.uv] constraint block. When c.DatabricksConnect is empty
// (constraints-only mode) the dev group is emitted empty rather than with a blank entry.
func RenderFreshPyproject(projectName string, c Constraints) []byte {
	var b strings.Builder
	b.WriteString("[project]\n")
	fmt.Fprintf(&b, "name = %q\n", projectName)
	// uv requires project.version when a [project] table is present.
	fmt.Fprintf(&b, "version = %q\n", freshProjectVersion)
	fmt.Fprintf(&b, "requires-python = %q\n", c.RequiresPython)
	b.WriteString("\n")
	b.WriteString("[dependency-groups]\n")
	if c.DatabricksConnect != "" {
		b.WriteString("dev = [\n")
		fmt.Fprintf(&b, "    %q,\n", c.DatabricksConnect)
		b.WriteString("]\n")
	} else {
		b.WriteString("dev = []\n")
	}
	b.WriteString("\n")
	for _, line := range renderToolUvBlock(c.ConstraintDeps, true) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return []byte(b.String())
}
