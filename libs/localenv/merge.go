package localenv

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// errMultilineString and errNoProjectTable are the conditions under which
// MergeManaged refuses to merge rather than risk corrupting the file. The caller
// surfaces both as E_MERGE.
var (
	errMultilineString = errors.New("pyproject.toml has an unterminated TOML multi-line string")
	errNoProjectTable  = errors.New("pyproject.toml has no [project] table to hold requires-python")
)

// managedMarkerStart and managedMarkerEnd bracket the region of pyproject.toml
// that this command owns. Everything between them is rewritten on each merge;
// everything outside is preserved byte-for-byte. They derive from CommandName so
// a command rename stays a single-place change (spec §0 / invariant 8).
var (
	managedMarkerStart = "# managed by databricks " + CommandName + " — do not edit"
	managedMarkerEnd   = "# end managed by databricks " + CommandName
)

// Region names reported back to the caller via MergeManaged's regions return value.
const (
	regionRequiresPython        = "requires-python"
	regionDatabricksConnect     = "databricks-connect"
	regionToolUv                = "tool.uv.constraint-dependencies"
	regionDatabricksEnvironment = "tool.databricks.environment"
)

// databricksEnvironmentTable is the TOML table header that carries the
// serverless environment version, and environmentVersionRe matches the start of
// its managed environment_version assignment (capturing leading whitespace so it
// is preserved when the value is replaced).
const databricksEnvironmentTable = "[tool.databricks.environment]"

var environmentVersionRe = regexp.MustCompile(`^(\s*)environment_version\s*=`)

var (
	// tableHeaderRe matches a TOML table header line: a standard table like
	// "[project]" / "[tool.uv]" or an array-of-tables like "[[tool.uv.index]]",
	// tolerating a trailing inline comment ("[project] # note"). Recognizing both
	// forms matters for bounds: an unrecognized "[[...]]" header would let a
	// table's end run past it (e.g. [tool.uv] swallowing its child [[tool.uv.index]]
	// items), and a commented header would similarly be missed.
	tableHeaderRe = regexp.MustCompile(`^\s*\[\[?[^\]]+\]\]?\s*(#.*)?$`)
	// requiresPythonRe captures the leading whitespace of a requires-python assignment so it
	// can be preserved when the value is replaced.
	requiresPythonRe = regexp.MustCompile(`^(\s*)requires-python\s*=`)
)

// removedDBConnect names a databricks-connect requirement the consolidation pass
// deletes and the array it came from. The location is a human-readable label
// ("[project].dependencies", "[dependency-groups].test") used in the advisory
// message and to subtract the pin from the warning detector's resolution view.
type removedDBConnect struct {
	location string
	pin      string
}

// dbconnectPlan is what the merge does to databricks-connect, computed by running
// the merge on a clone so the warning detector never re-derives the merge's rules.
// replacedDevPin is the dev-group pin mergeDatabricksConnect rewrites in place ("" if
// it inserts the managed pin instead); removed is every stray pin the consolidation
// pass deletes.
type dbconnectPlan struct {
	replacedDevPin string
	removed        []removedDBConnect
}

// planDBConnect returns the databricks-connect edits merging target would make, or
// the zero plan when databricks-connect is skipped (--no-dbconnect / --constraints-only)
// or the artifact carries no pin — the cases where MergeManaged leaves databricks-connect
// untouched. It mirrors MergeManaged's preprocessing (CRLF normalization and
// multi-line string protection) and runs both databricks-connect passes on a clone,
// in the same order as MergeManaged, so replacedDevPin and removed match what the
// real merge does.
func planDBConnect(target []byte, c Constraints, opts MergeOptions) dbconnectPlan {
	if opts.SkipDBConnect || c.DatabricksConnect == "" {
		return dbconnectPlan{}
	}
	// Mirror MergeManaged's own preprocessing so the same lines are inspected.
	protected, _, err := protectMultilineStrings(strings.ReplaceAll(string(target), "\r\n", "\n"))
	if err != nil {
		return dbconnectPlan{}
	}
	lines := strings.Split(protected, "\n")
	// mergeDatabricksConnect rewrites element lines in place, so hand it a copy: this
	// probe must not disturb the caller's view of the pre-merge file. The consolidation
	// pass then runs on its output, so removed reflects the post-dev-merge state where
	// the managed pin is already the dev group's first databricks-connect element.
	merged, replaced, _ := mergeDatabricksConnect(slices.Clone(lines), c.DatabricksConnect)
	_, removed, _ := removeStrayDatabricksConnect(merged, c.DatabricksConnect)
	return dbconnectPlan{replacedDevPin: replaced, removed: removed}
}

// MergeOptions selects which orthogonal managed axes are left unmanaged. Each flag
// is threaded explicitly rather than inferred from empty/nil values in the
// Constraints, so a caller's intent is unambiguous and the Constraints always carry
// the real artifact values. The zero value manages every axis. A new axis is a new
// field here — callers that manage everything keep passing MergeOptions{} unchanged.
type MergeOptions struct {
	// SkipConstraints (--no-constraints) leaves the requires-python and [tool.uv]
	// constraint regions unmanaged: existing values are preserved and none written.
	SkipConstraints bool
	// SkipDBConnect (--no-dbconnect / --constraints-only) leaves the
	// databricks-connect dependency unmanaged: an existing pin is preserved and none
	// is injected or asserted.
	SkipDBConnect bool
}

// MergeManaged applies the managed transforms to target, preserving every other
// byte (comments, ordering, whitespace). It returns the merged bytes and the list of
// regions that actually changed. The operation is idempotent: feeding its own output
// back in produces identical bytes. opts selects which axes are managed; the
// environment region is always reconciled.
func MergeManaged(target []byte, c Constraints, opts MergeOptions) (merged []byte, regions []string, err error) {
	s := string(target)

	// Detect and normalize line endings. We process on "\n" and restore "\r\n" on
	// exit. Line endings are treated as a whole-file property: a file that uses
	// CRLF anywhere is emitted entirely as CRLF. Real pyproject.toml files use one
	// consistent ending, so this is faithful in practice; a file that deliberately
	// mixes LF and CRLF would be normalized to CRLF (a benign whitespace-only
	// change), which we accept rather than track a terminator per line.
	crlf := strings.Contains(s, "\r\n")
	if crlf {
		s = strings.ReplaceAll(s, "\r\n", "\n")
	}

	protected, restore, err := protectMultilineStrings(s)
	if err != nil {
		return nil, nil, err
	}
	lines := strings.Split(protected, "\n")

	// requires-python needs a [project] table to hold it; without one the merge
	// would silently drop the pin, so fail loudly instead. Only enforced when
	// constraints are managed — under SkipConstraints there is no requires-python to
	// write, so a [project]-less file (e.g. one with only dependency groups) is
	// merged for its other axes rather than rejected.
	if !opts.SkipConstraints {
		if _, _, ok := tableBounds(lines, "[project]"); !ok {
			return nil, nil, errNoProjectTable
		}
	}

	if !opts.SkipConstraints {
		var rpChanged bool
		lines, rpChanged = mergeRequiresPython(lines, c.RequiresPython)
		if rpChanged {
			regions = append(regions, regionRequiresPython)
		}
	}

	if !opts.SkipDBConnect {
		var dbcChanged bool
		lines, _, dbcChanged = mergeDatabricksConnect(lines, c.DatabricksConnect)
		// In the install flow, after the managed pin lands in the dev group, remove any
		// databricks-connect pin elsewhere that is disjoint from it — the pins that would
		// otherwise make uv unsatisfiable. Compatible pins are left alone. An empty pin
		// (an artifact without databricks-connect) is a data no-op, distinct from the
		// SkipDBConnect intent handled by the gate above.
		strayChanged := false
		if c.DatabricksConnect != "" {
			lines, _, strayChanged = removeStrayDatabricksConnect(lines, c.DatabricksConnect)
		}
		if dbcChanged || strayChanged {
			regions = append(regions, regionDatabricksConnect)
		}
	}

	lines, envChanged := mergeDatabricksEnvironment(lines, c.EnvironmentVersion)
	if envChanged {
		regions = append(regions, regionDatabricksEnvironment)
	}

	if !opts.SkipConstraints {
		var uvChanged bool
		lines, uvChanged = mergeToolUv(lines, c.ConstraintDeps)
		if uvChanged {
			regions = append(regions, regionToolUv)
		}
	}

	out := restore(strings.Join(lines, "\n"))
	if crlf {
		out = strings.ReplaceAll(out, "\n", "\r\n")
	}
	return []byte(out), regions, nil
}

// headerName returns the bracketed table name of a TOML table-header line (e.g.
// "[project]" from "  [project] # note", or "[[tool.uv.index]]" from an
// array-of-tables header), or "" if the line is not a table header. It strips a
// trailing inline comment so a commented header still matches by name, and keeps
// the full "[[...]]" form so an array-of-tables header is never treated as the
// same table as its "[...]" parent.
func headerName(line string) string {
	if !tableHeaderRe.MatchString(line) {
		return ""
	}
	s := strings.TrimSpace(line)
	// Array-of-tables: "[[name]]" — return through the closing "]]".
	if strings.HasPrefix(s, "[[") {
		if i := strings.Index(s, "]]"); i >= 0 {
			return s[:i+2]
		}
	}
	if i := strings.Index(s, "]"); i >= 0 {
		return s[:i+1]
	}
	return s
}

// tableBounds returns the line index of the header matching name (e.g. "[project]") and
// the index of the first line after the table body (the next table header or EOF). If the
// table is absent, found is false.
func tableBounds(lines []string, name string) (header, end int, found bool) {
	header = -1
	for i, line := range lines {
		if headerName(line) == name {
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
	// Only reached when constraints are managed (MergeManaged gates on
	// skipConstraints), where a fetched artifact always carries a requires-python.
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

// mergeDatabricksEnvironment pins environment_version to version within the
// env-owned [tool.databricks.environment] table: it replaces an existing value
// (preserving the line's indentation and any inline comment), inserts the key
// when the table exists without it, and appends the table when it is absent.
// An empty version (a cluster target) is a no-op — the section is only written
// for serverless targets, so an existing one is left untouched rather than
// removed. Returns whether the line slice changed.
func mergeDatabricksEnvironment(lines []string, version string) ([]string, bool) {
	if version == "" {
		return lines, false
	}

	header, end, found := tableBounds(lines, databricksEnvironmentTable)
	if !found {
		return appendManagedBlock(lines, []string{databricksEnvironmentTable, fmt.Sprintf(`environment_version = "%s"`, version)}), true
	}

	for i := header + 1; i < end; i++ {
		m := environmentVersionRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		// Only the value is managed; a trailing inline comment is user content.
		replacement := fmt.Sprintf(`%senvironment_version = "%s"%s`, m[1], version, trailingComment(lines[i]))
		if lines[i] == replacement {
			return lines, false
		}
		lines[i] = replacement
		return lines, true
	}

	// Table exists but has no environment_version: insert directly under the header.
	inserted := make([]string, 0, len(lines)+1)
	inserted = append(inserted, lines[:header+1]...)
	inserted = append(inserted, fmt.Sprintf(`environment_version = "%s"`, version))
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

// devKeyRe matches the start of the dev array assignment within [dependency-groups]
// (e.g. "dev = [" or "dev=["), capturing leading whitespace. Only this key is
// managed; sibling groups such as test/docs are user-owned and left untouched.
var devKeyRe = regexp.MustCompile(`^\s*dev\s*=`)

// mergeDatabricksConnect ensures [dependency-groups].dev pins databricks-connect
// to value. It replaces an existing databricks-connect element (multi-line or
// single-line array form) and, when none exists, inserts one — creating the dev
// key and the [dependency-groups] table when those are absent too, mirroring
// RenderFreshPyproject so an existing project without a pin provisions the same as
// a greenfield one. An empty value (constraints-only mode) is a no-op: the user's
// dev group is left untouched rather than having its pin blanked out or one added.
//
// The edit is scoped to the dev array's own span (found via bracket depth), so a
// databricks-connect pin sitting in a sibling group (e.g. docs/test) or inside a
// trailing comment on some other line is never clobbered. The insert path is
// idempotent: a subsequent merge finds the element and rewrites it in place.
//
// replacedPin is the requirement it rewrote in place, empty when it inserted the
// managed pin instead. That distinction is what detectMergeWarnings needs: only a
// rewrite means the user's pin is gone, and only the merge itself can say which
// spellings it recognizes.
func mergeDatabricksConnect(lines []string, value string) (out []string, replacedPin string, changed bool) {
	if value == "" {
		return lines, "", false
	}
	elem := `"` + value + `"`

	header, end, found := tableBounds(lines, "[dependency-groups]")
	if !found {
		// No [dependency-groups] table: append a fresh managed dev group.
		return appendManagedBlock(lines, []string{"[dependency-groups]", "dev = [", "    " + elem + ",", "]"}), "", true
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
		// The table exists but has no dev key: insert one right after the header.
		insert := []string{"dev = [", "    " + elem + ",", "]"}
		out := make([]string, 0, len(lines)+len(insert))
		out = append(out, lines[:header+1]...)
		out = append(out, insert...)
		out = append(out, lines[header+1:]...)
		return out, "", true
	}
	arrayLast, _ := arrayLineSpan(lines, devStart, end)

	// Single-line form: the whole array is on the dev line itself. Only edit
	// within the array (through its closing "]"); a trailing comment after it is
	// user content and must be left byte-for-byte intact.
	if devStart == arrayLast {
		line := lines[devStart]
		arrayPart, commentPart := splitAtArrayClose(line)
		if rewritten, replaced, ok := replaceDbconnectElement(arrayPart, elem); ok {
			newLine := rewritten + commentPart
			if newLine == line {
				return lines, replaced, false
			}
			lines[devStart] = newLine
			return lines, replaced, true
		}
		// No databricks-connect element: insert one as the first array element.
		open := strings.Index(arrayPart, "[")
		closeIdx := strings.LastIndex(arrayPart, "]")
		if open < 0 || closeIdx < open {
			return lines, "", false
		}
		inner := strings.TrimSpace(arrayPart[open+1 : closeIdx])
		newInner := elem
		if inner != "" {
			newInner = elem + ", " + inner
		}
		lines[devStart] = arrayPart[:open+1] + newInner + arrayPart[closeIdx:] + commentPart
		return lines, "", true
	}

	// Multi-line form: the array spans devStart..arrayLast. An existing
	// databricks-connect element may sit on a dedicated element line, share the
	// opening "dev = [" line, or carry a trailing comment, and may be spelled with
	// any PEP 503-equivalent separators/case — so detect it (outside comments,
	// name-normalized) anywhere in the span and rewrite in place. Only when no
	// element exists anywhere do we insert, which avoids duplicating a pin a raw
	// substring match would miss.
	lastElem := -1
	for i := devStart; i <= arrayLast; i++ {
		code, comment := lines[i], ""
		if c := commentStart(code); c >= 0 {
			code, comment = code[:c], code[c:]
		}
		if rewritten, replaced, ok := replaceDbconnectElement(code, elem); ok {
			// Rewrite only the code portion; a trailing comment is user content and
			// must be preserved byte-for-byte, even if it contains a quoted token.
			newLine := rewritten + comment
			if newLine == lines[i] {
				return lines, replaced, false
			}
			lines[i] = newLine
			return lines, replaced, true
		}
		if dbconnectQuotedRe.MatchString(code) {
			lastElem = i
		}
	}
	// No databricks-connect element anywhere in the array: insert one before the
	// closing "]", matching the indentation of the existing elements (default four
	// spaces when empty).
	indent := "    "
	for i := devStart + 1; i < arrayLast; i++ {
		if strings.TrimSpace(lines[i]) != "" {
			indent = leadingWhitespace(lines[i])
			break
		}
	}
	// TOML lets the final element omit its trailing comma; appending after it would
	// then leave two adjacent elements with no separator. Ensure the current last
	// element (if it is on its own line, not the "]" line) carries a comma first.
	if lastElem >= 0 && lastElem < arrayLast {
		lines[lastElem] = ensureTrailingComma(lines[lastElem])
	}
	inserted := make([]string, 0, len(lines)+1)
	inserted = append(inserted, lines[:arrayLast]...)
	inserted = append(inserted, indent+elem+",")
	inserted = append(inserted, lines[arrayLast:]...)
	return inserted, "", true
}

// dbconnectQuotedRe matches any double-quoted array element token.
var dbconnectQuotedRe = regexp.MustCompile(`"[^"]*"`)

// replaceDbconnectElement replaces the first quoted element in code whose package
// name is databricks-connect (compared under PEP 503 normalization, so
// "databricks_connect" / "Databricks-Connect" / "databricks.connect" all match)
// with elem. It returns the rewritten code, the requirement it replaced, and
// whether a replacement was made. Matching mirrors the artifact side
// (isDatabricksConnectDep) so a differently spelled existing pin is rewritten in
// place rather than left for the insert path to duplicate.
func replaceDbconnectElement(code, elem string) (out, replaced string, ok bool) {
	for _, m := range dbconnectQuotedRe.FindAllStringIndex(code, -1) {
		inner := code[m[0]+1 : m[1]-1]
		if isDatabricksConnectDep(inner) {
			return code[:m[0]] + elem + code[m[1]:], inner, true
		}
	}
	return code, "", false
}

// arrayKeyRe matches a "key = [" array assignment, capturing the key. The "[" must
// be on the assignment line (TOML array syntax), so an inline table value ("= {")
// or a scalar is not matched. "." is allowed in the key so a dotted assignment
// (optional-dependencies.extra = [...] inside [project]) is captured whole — as the
// key "optional-dependencies.extra", which is intentionally not equal to "dependencies"
// and so is skipped, rather than mistaken for the [project].dependencies array.
var arrayKeyRe = regexp.MustCompile(`^\s*([A-Za-z0-9._-]+)\s*=\s*\[`)

// arraySpan locates one "key = [ ... ]" array assignment: the key and the line
// indices its value opens and closes on.
type arraySpan struct {
	key   string
	start int
	last  int
}

// arrayAssignmentsIn returns each top-level "key = [ ... ]" array assignment in the
// table body spanning (header, end), in file order. Continuation lines of a
// multi-line array are skipped so a requirement string that happens to look like an
// assignment is not mistaken for one.
func arrayAssignmentsIn(lines []string, header, end int) []arraySpan {
	var out []arraySpan
	for i := header + 1; i < end; i++ {
		m := arrayKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		last, _ := arrayLineSpan(lines, i, end)
		out = append(out, arraySpan{key: m[1], start: i, last: last})
		i = last
	}
	return out
}

// removeStrayDatabricksConnect deletes the databricks-connect requirements that
// provably cannot co-resolve with the managed env pin (envPin) — from
// [project].dependencies, every [project.optional-dependencies] extra, and every
// [dependency-groups] group. It reports each deleted pin with its location.
//
// Only a *disjoint* pin is removed (see dbconnectPinConflicts). That is the pin that
// actually makes uv unsatisfiable — the bug this exists to fix — and it is the whole
// justification for reaching into locations the merge otherwise leaves alone,
// including [project].dependencies, which the default template compiles into the
// built wheel's metadata: deleting a pin there is only warranted when the project
// would not resolve or build as-is. A pin that co-resolves (databricks-connect>=15
// against ~=17.2.0), carries no version, is marker-gated, or already equals envPin is
// left untouched — there is nothing to fix, so the user's declaration stands. This
// deliberately overrides, for disjoint pins only, the "sibling groups are user-owned
// and left untouched" contract that scopes mergeDatabricksConnect (see devKeyRe).
//
// The removal is line-based and matches the same shapes the rewrite does: a
// double-quoted element under a bare-key array in the [project], [project.optional-
// dependencies], or [dependency-groups] tables. Rarer-but-valid spellings are out of
// its reach and left in place — a single-quoted pin, a pin under a quoted TOML key
// ("qa group" = [...]), and a pin in an inline-table or dotted sub-table form. These
// are not silent: detectMergeWarnings decodes the file fully, so a disjoint survivor
// is still surfaced as W_DBCONNECT_PIN_DUPLICATED.
func removeStrayDatabricksConnect(lines []string, envPin string) (out []string, removed []removedDBConnect, changed bool) {
	type target struct {
		span     arraySpan
		location string
	}
	var targets []target

	if h, e, ok := tableBounds(lines, "[project]"); ok {
		for _, a := range arrayAssignmentsIn(lines, h, e) {
			// Only [project].dependencies is a resolution requirement; other [project]
			// arrays (keywords, classifiers, ...) never hold databricks-connect.
			if a.key == "dependencies" {
				targets = append(targets, target{a, "[project].dependencies"})
			}
		}
	}
	if h, e, ok := tableBounds(lines, "[project.optional-dependencies]"); ok {
		for _, a := range arrayAssignmentsIn(lines, h, e) {
			targets = append(targets, target{a, "[project.optional-dependencies]." + a.key})
		}
	}
	if h, e, ok := tableBounds(lines, "[dependency-groups]"); ok {
		for _, a := range arrayAssignmentsIn(lines, h, e) {
			targets = append(targets, target{a, "[dependency-groups]." + a.key})
		}
	}
	if len(targets) == 0 {
		return lines, nil, false
	}

	// Rewrite spans in a single ascending pass so a line-count change from one edit
	// does not shift the indices of the others. Spans in different tables never
	// overlap, and arrayAssignmentsIn already returns each table's spans in order.
	slices.SortFunc(targets, func(a, b target) int { return cmp.Compare(a.span.start, b.span.start) })

	out = make([]string, 0, len(lines))
	prev := 0
	for _, t := range targets {
		out = append(out, lines[prev:t.span.start]...)
		newSpan, pins := removeDbconnectFromArraySpan(lines[t.span.start:t.span.last+1], envPin)
		out = append(out, newSpan...)
		for _, pin := range pins {
			removed = append(removed, removedDBConnect{location: t.location, pin: pin})
		}
		prev = t.span.last + 1
	}
	out = append(out, lines[prev:]...)

	// out is fully built above; it is only returned when something was removed, so a
	// no-op run returns the original slice unchanged.
	if len(removed) == 0 {
		return lines, nil, false
	}
	return out, removed, true
}

// dbconnectPinConflicts reports whether a databricks-connect pin provably cannot
// co-resolve with the managed env pin — their version ranges are disjoint. A pin the
// range model cannot compare (no version, a marker-gated requirement, an unparseable
// or wildcard-only spelling) is not provably conflicting, so it is treated as
// compatible and left in place. envPin, written by the merge, always parses.
func dbconnectPinConflicts(pin, envPin string) bool {
	_, pinSpec, pinOK := splitDepSpec(pin)
	_, envSpec, envOK := splitDepSpec(envPin)
	return pinOK && envOK && rangesDisjoint(pinSpec, envSpec)
}

// removeDbconnectFromArraySpan removes the databricks-connect elements that conflict
// with envPin (see dbconnectPinConflicts) from a single array value spanning spanLines
// (single- or multi-line), returning the rewritten lines and the removed pins. A
// compatible databricks-connect element — including the managed dev pin, which equals
// envPin and so never conflicts with itself — is left in place. It operates on
// top-level array elements so a version range comma ("databricks-connect>=15,<16") or
// an inline table is not split, and rejoining survivors preserves each element's own
// leading whitespace, newline, and trailing comma.
func removeDbconnectFromArraySpan(spanLines []string, envPin string) (out, removed []string) {
	block := strings.Join(spanLines, "\n")
	prefix, body, suffix, ok := arrayParts(block)
	if !ok {
		return spanLines, nil
	}
	var kept []string
	// carry holds the leading comment/blank lines of a removed element. splitTopLevelElements
	// breaks on commas, so a trailing comment left on the *previous* element's line lands as
	// a leading comment line on this element's token; dropping the whole token would delete
	// that previous element's comment (MergeManaged preserves comments). Carry those lines to
	// the next retained token so the comment stays on the line it belonged to. A comment that
	// instead described the removed element is indistinguishable from that case, so it is kept
	// too and may end up beside the following element — content is never lost, but a comment
	// can be relocated; that is the accepted trade-off of the split-on-commas approach.
	carry := ""
	for _, elem := range splitTopLevelElements(body) {
		elem = carry + elem
		carry = ""
		if pin, isDBC := dbconnectElementPin(elem); isDBC && dbconnectPinConflicts(pin, envPin) {
			removed = append(removed, pin)
			carry = leadingLinesBeforeElement(elem)
			continue
		}
		kept = append(kept, elem)
	}
	if len(removed) == 0 {
		return spanLines, nil
	}
	// A carry left over after the last token (the removed element was last) has no following
	// token; emit it on its own line before the closing "]" so the comment survives and does
	// not comment out the bracket.
	if strings.TrimSpace(carry) != "" {
		kept = append(kept, carry+"\n")
	}
	// Removing the last element of a multi-line array would pull the closing "]" up onto
	// the previous element's line (that element's token carried no trailing newline) and
	// drop the array's trailing comma. When the array was multi-line (body ends in a
	// newline) and a real element still remains, restore the trailing comma and the
	// newline before "]" so the bracket keeps its own line and the magic trailing comma
	// survives.
	joined := strings.Join(kept, ",")
	if strings.HasSuffix(body, "\n") && len(kept) > 0 && !strings.HasSuffix(joined, "\n") {
		if !strings.HasSuffix(joined, ",") {
			joined += ","
		}
		joined += "\n"
	}
	return strings.Split(prefix+joined+suffix, "\n"), removed
}

// leadingLinesBeforeElement returns the comment/blank lines that precede the value line
// in an array-element token — the trailing comment splitTopLevelElements moved here from
// the previous element's line. The value line and anything after it (the element's own
// trailing comment) is excluded, since that belongs to the element being removed.
func leadingLinesBeforeElement(elem string) string {
	lines := strings.Split(elem, "\n")
	for i, line := range lines {
		if c := commentStart(line); c >= 0 {
			line = line[:c]
		}
		if strings.TrimSpace(line) != "" {
			return strings.Join(lines[:i], "\n")
		}
	}
	return ""
}

// arrayParts splits a joined "key = [ ... ]" block into the text up to and including
// the opening "[", the body between the brackets, and the text from the matching "]"
// onward. It scans across newlines, tracking quoted strings and per-line "#" comments
// so a bracket inside either is ignored. ok is false when no balanced array is found.
func arrayParts(block string) (prefix, body, suffix string, ok bool) {
	depth := 0
	var quote byte
	inComment := false
	openIdx := -1
	for i := 0; i < len(block); i++ {
		c := block[i]
		switch {
		case c == '\n':
			inComment = false
		case inComment:
			// skip until newline
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			inComment = true
		case c == '[':
			depth++
			if depth == 1 {
				openIdx = i
			}
		case c == ']':
			depth--
			if depth == 0 && openIdx != -1 {
				return block[:openIdx+1], block[openIdx+1 : i], block[i:], true
			}
		}
	}
	return "", "", "", false
}

// splitTopLevelElements splits an array body into its element tokens, breaking only
// on commas outside quotes, outside nested []/{} , and outside "#" comments. Each
// token keeps its surrounding whitespace, newline, and any comment, so survivors
// reassemble byte-for-byte. The final segment (the whitespace before "]") is kept so
// a trailing comma is preserved on rejoin.
func splitTopLevelElements(body string) []string {
	var elems []string
	depth := 0
	var quote byte
	inComment := false
	start := 0
	for i := 0; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '\n':
			inComment = false
		case inComment:
			// skip until newline
		case quote != 0:
			if c == '\\' && quote == '"' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			inComment = true
		case c == '[' || c == '{':
			depth++
		case c == ']' || c == '}':
			depth--
		case c == ',' && depth == 0:
			elems = append(elems, body[start:i])
			start = i + 1
		}
	}
	return append(elems, body[start:])
}

// dbconnectElementPin returns the databricks-connect requirement an array element
// token pins, if the token is a double-quoted string naming databricks-connect.
// Single-quoted and non-string tokens (e.g. a PEP 735 include-group table) are not
// matched, mirroring replaceDbconnectElement's double-quoted-only rule.
//
// The token is scanned line by line rather than as one string: splitTopLevelElements
// breaks only on commas, so the token for an element carries any inline comment left
// on the *previous* element's line as a leading comment line. Stripping the comment
// with commentStart over the whole token would treat that leading "#" as running to
// end-of-token and hide the quoted requirement on the next line, silently skipping a
// stray pin. commentStart is therefore applied per line, where it correctly stops at
// the line's own end.
func dbconnectElementPin(elem string) (pin string, ok bool) {
	for line := range strings.SplitSeq(elem, "\n") {
		if c := commentStart(line); c >= 0 {
			line = line[:c]
		}
		line = strings.TrimSpace(line)
		if len(line) >= 2 && line[0] == '"' && line[len(line)-1] == '"' {
			inner := line[1 : len(line)-1]
			if isDatabricksConnectDep(inner) {
				return inner, true
			}
		}
	}
	return "", false
}

// ensureTrailingComma appends a "," after the last non-space code character of
// line when it lacks one, preserving any trailing whitespace and inline comment.
// A blank or comment-only line is returned unchanged.
func ensureTrailingComma(line string) string {
	code, comment := line, ""
	if c := commentStart(code); c >= 0 {
		code, comment = code[:c], code[c:]
	}
	trimmed := strings.TrimRight(code, " \t")
	if trimmed == "" || strings.HasSuffix(trimmed, ",") {
		return line
	}
	return trimmed + "," + code[len(trimmed):] + comment
}

// leadingWhitespace returns the run of spaces and tabs at the start of s.
func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
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

// mergeToolUv rewrites the managed [tool.uv] constraint-dependencies block. If a
// marker-bracketed block already exists, its contents are replaced in place. Otherwise any
// plain [tool.uv] table is removed and a fresh marker-bracketed block is appended at EOF.
func mergeToolUv(lines, deps []string) ([]string, bool) {
	// Only reached when constraints are managed (MergeManaged gates on
	// skipConstraints). A nil deps slice is treated identically to an empty one:
	// both render an empty managed block.
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
		if name := headerName(lines[i]); name != "" {
			return name == "[tool.uv]"
		}
	}
	return false
}

// constraintDepsRe matches the start of a constraint-dependencies assignment within a
// [tool.uv] table, capturing its leading whitespace.
var constraintDepsRe = regexp.MustCompile(`^\s*constraint-dependencies\s*=`)

// protectMultilineStrings replaces TOML multi-line strings with unique ordinary
// string values while the line-based merge runs. This prevents string contents
// that resemble tables, assignments, brackets, comments, or managed markers
// from affecting the merge. The returned restore function puts each original
// string back byte-for-byte after the managed edits are complete.
func protectMultilineStrings(s string) (protected string, restore func(string) string, err error) {
	type replacement struct {
		placeholder string
		original    string
	}

	prefix := "__databricks_setup_local_multiline_"
	for strings.Contains(s, prefix) {
		prefix += "_"
	}

	var replacements []replacement
	var out strings.Builder
	for i := 0; i < len(s); {
		switch {
		case s[i] == '#':
			end := strings.IndexByte(s[i:], '\n')
			if end < 0 {
				out.WriteString(s[i:])
				i = len(s)
				continue
			}
			end += i
			out.WriteString(s[i:end])
			i = end
		case strings.HasPrefix(s[i:], `"""`) || strings.HasPrefix(s[i:], "'''"):
			delimiter := s[i : i+3]
			end, ok := multilineStringEnd(s, i+3, delimiter)
			if !ok {
				return "", nil, errMultilineString
			}
			placeholder := fmt.Sprintf(`"%s%d__"`, prefix, len(replacements))
			replacements = append(replacements, replacement{placeholder: placeholder, original: s[i:end]})
			out.WriteString(placeholder)
			i = end
		case s[i] == '"' || s[i] == '\'':
			quote := s[i]
			start := i
			i++
			for i < len(s) {
				if quote == '"' && s[i] == '\\' {
					i += min(2, len(s)-i)
					continue
				}
				i++
				if s[i-1] == quote {
					break
				}
			}
			out.WriteString(s[start:i])
		default:
			out.WriteByte(s[i])
			i++
		}
	}

	restore = func(merged string) string {
		for _, r := range replacements {
			merged = strings.ReplaceAll(merged, r.placeholder, r.original)
		}
		return merged
	}
	return out.String(), restore, nil
}

// multilineStringEnd returns the byte immediately after a multi-line string's
// closing delimiter. Backslash escapes apply only to basic (double-quoted)
// strings. Runs of four or five quotes include one or two quotes in the value,
// followed by the closing three-quote delimiter.
func multilineStringEnd(s string, start int, delimiter string) (int, bool) {
	for i := start; i < len(s); {
		if delimiter == `"""` && s[i] == '\\' {
			i += min(2, len(s)-i)
			continue
		}
		if strings.HasPrefix(s[i:], delimiter) {
			run := 3
			for run < 5 && i+run < len(s) && s[i+run] == delimiter[0] {
				run++
			}
			return i + run, true
		}
		i++
	}
	return 0, false
}

// commentStart returns the index of the "#" that begins an inline comment on
// line, or -1 if there is none. A "#" inside a quoted string is not a comment.
func commentStart(line string) int {
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
			return i
		}
	}
	return -1
}

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
// none, with [project], [dependency-groups].dev (carrying the databricks-connect pin), the
// [tool.databricks.environment] section (serverless targets only), and the marker-bracketed
// [tool.uv] constraint block. When databricks-connect is skipped (--no-dbconnect /
// --constraints-only) or the artifact carries no pin, the dev group is emitted empty.
func RenderFreshPyproject(projectName string, c Constraints, opts MergeOptions) []byte {
	var b strings.Builder
	b.WriteString("[project]\n")
	fmt.Fprintf(&b, "name = %q\n", projectName)
	// uv requires project.version when a [project] table is present.
	fmt.Fprintf(&b, "version = %q\n", freshProjectVersion)
	// Omitted under SkipConstraints, letting uv pick the interpreter for the project.
	if !opts.SkipConstraints {
		fmt.Fprintf(&b, "requires-python = %q\n", c.RequiresPython)
	}
	b.WriteString("\n")
	b.WriteString("[dependency-groups]\n")
	if !opts.SkipDBConnect && c.DatabricksConnect != "" {
		b.WriteString("dev = [\n")
		fmt.Fprintf(&b, "    %q,\n", c.DatabricksConnect)
		b.WriteString("]\n")
	} else {
		b.WriteString("dev = []\n")
	}
	b.WriteString("\n")
	// The serverless environment version is written only for serverless targets;
	// a cluster target leaves EnvironmentVersion empty and omits the section.
	if c.EnvironmentVersion != "" {
		b.WriteString(databricksEnvironmentTable + "\n")
		fmt.Fprintf(&b, "environment_version = %q\n", c.EnvironmentVersion)
		b.WriteString("\n")
	}
	// Written whenever constraints are managed — an empty block when the artifact
	// carries no constraint-dependencies (nil and empty are treated alike).
	if !opts.SkipConstraints {
		for _, line := range renderToolUvBlock(c.ConstraintDeps, true) {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	return []byte(b.String())
}
