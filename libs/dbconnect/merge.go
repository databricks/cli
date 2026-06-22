package dbconnect

import (
	"fmt"
	"regexp"
	"strings"
)

// managedMarkerStart and managedMarkerEnd bracket the region of pyproject.toml that
// "databricks dbconnect" owns. Everything between them is rewritten on each merge;
// everything outside is preserved byte-for-byte.
const (
	managedMarkerStart = "# managed by databricks dbconnect — do not edit"
	managedMarkerEnd   = "# end managed by databricks dbconnect"
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

	want := func(indent string) string {
		return fmt.Sprintf(`%srequires-python = "%s"`, indent, value)
	}

	for i := header + 1; i < end; i++ {
		m := requiresPythonRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		replacement := want(m[1])
		if lines[i] == replacement {
			return lines, false
		}
		lines[i] = replacement
		return lines, true
	}

	// Key absent: insert directly under the [project] header.
	inserted := make([]string, 0, len(lines)+1)
	inserted = append(inserted, lines[:header+1]...)
	inserted = append(inserted, want(""))
	inserted = append(inserted, lines[header+1:]...)
	return inserted, true
}

// dbconnectLineRe captures, for a line holding a databricks-connect dependency element:
// (1) the leading whitespace, and (3) any trailing comma (with optional trailing space),
// so that indentation and comma style are preserved when the quoted token is replaced.
var dbconnectLineRe = regexp.MustCompile(`^(\s*)"databricks-connect[^"]*"(\s*,?\s*)$`)

// mergeDatabricksConnect replaces the databricks-connect element inside
// [dependency-groups].dev. It handles both the multi-line array form (one element per
// line) and the single-line array form (dev = ["databricks-connect~=..."]).
func mergeDatabricksConnect(lines []string, value string) ([]string, bool) {
	header, end, found := tableBounds(lines, "[dependency-groups]")
	if !found {
		return lines, false
	}

	for i := header + 1; i < end; i++ {
		// Multi-line element form: a standalone line holding only the quoted token.
		if m := dbconnectLineRe.FindStringSubmatch(lines[i]); m != nil {
			replacement := fmt.Sprintf(`%s"%s"%s`, m[1], value, m[2])
			if lines[i] == replacement {
				return lines, false
			}
			lines[i] = replacement
			return lines, true
		}
		// Single-line array form: replace the quoted databricks-connect token in place.
		if strings.Contains(lines[i], `"databricks-connect`) {
			replaced := dbconnectTokenRe.ReplaceAllString(lines[i], `"`+value+`"`)
			if replaced == lines[i] {
				return lines, false
			}
			lines[i] = replaced
			return lines, true
		}
	}
	return lines, false
}

// dbconnectTokenRe matches a quoted databricks-connect element anywhere in a line, used
// for the single-line array form.
var dbconnectTokenRe = regexp.MustCompile(`"databricks-connect[^"]*"`)

// mergeToolUv rewrites the managed [tool.uv] constraint-dependencies block. If a
// marker-bracketed block already exists, its contents are replaced in place. Otherwise any
// plain [tool.uv] table is removed and a fresh marker-bracketed block is appended at EOF.
func mergeToolUv(lines, deps []string) ([]string, bool) {
	block := renderToolUvBlock(deps)

	start, stop, found := markerBounds(lines)
	if found {
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

	// No managed block: reconcile any plain [tool.uv] table, then append a fresh managed
	// block at EOF. If the table is effectively ours (its only meaningful key is
	// constraint-dependencies, from a pre-marker run), drop it whole. Otherwise the table
	// holds user-authored keys, so we preserve it and strip only our constraint-dependencies.
	if header, end, ok := tableBounds(lines, "[tool.uv]"); ok {
		if toolUvHasOnlyConstraintDeps(lines, header, end) {
			out := make([]string, 0, len(lines))
			out = append(out, lines[:header]...)
			out = append(out, lines[end:]...)
			lines = out
		} else {
			lines = removeConstraintDeps(lines, header, end)
		}
	}

	lines = appendManagedBlock(lines, block)
	return lines, true
}

// constraintDepsRe matches the start of a constraint-dependencies assignment within a
// [tool.uv] table, capturing its leading whitespace.
var constraintDepsRe = regexp.MustCompile(`^\s*constraint-dependencies\s*=`)

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
		// Multi-line array form: skip the continuation lines through the closing "]"
		// so the whole managed key counts as ignorable (mirrors removeConstraintDeps).
		// The single-line form already holds the "]" and does not advance i.
		if !strings.Contains(lines[i], "]") {
			for i++; i < end; i++ {
				if strings.TrimSpace(lines[i]) == "]" {
					break
				}
			}
		}
	}
	return true
}

// removeConstraintDeps strips a constraint-dependencies key from the [tool.uv] table body
// spanning (header, end), leaving the table header and all other user keys in place. It
// handles both the single-line array form and the multi-line array form (the value spans
// several lines until a line whose trimmed content is "]").
func removeConstraintDeps(lines []string, header, end int) []string {
	for i := header + 1; i < end; i++ {
		if !constraintDepsRe.MatchString(lines[i]) {
			continue
		}
		// Multi-line array form: extend through the closing "]" line. The single-line form
		// already contains the closing bracket, so this loop does not advance.
		end2 := i + 1
		if !strings.Contains(lines[i], "]") {
			for j := i + 1; j < end; j++ {
				end2 = j + 1
				if strings.TrimSpace(lines[j]) == "]" {
					break
				}
			}
		}
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

// renderToolUvBlock builds the marker-bracketed [tool.uv] block lines (no surrounding
// blank lines).
func renderToolUvBlock(deps []string) []string {
	block := []string{
		managedMarkerStart,
		"[tool.uv]",
		"constraint-dependencies = [",
	}
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

// RenderFreshPyproject produces a complete managed pyproject.toml for a project that has
// none, with [project], [dependency-groups].dev (carrying the databricks-connect pin), and
// the marker-bracketed [tool.uv] constraint block.
func RenderFreshPyproject(projectName string, c Constraints) []byte {
	var b strings.Builder
	b.WriteString("[project]\n")
	fmt.Fprintf(&b, "name = %q\n", projectName)
	fmt.Fprintf(&b, "requires-python = %q\n", c.RequiresPython)
	b.WriteString("\n")
	b.WriteString("[dependency-groups]\n")
	b.WriteString("dev = [\n")
	fmt.Fprintf(&b, "    %q,\n", c.DatabricksConnect)
	b.WriteString("]\n")
	b.WriteString("\n")
	for _, line := range renderToolUvBlock(c.ConstraintDeps) {
		b.WriteString(line)
		b.WriteString("\n")
	}
	return []byte(b.String())
}
