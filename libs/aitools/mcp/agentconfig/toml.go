package agentconfig

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/databricks/cli/libs/localenv"
)

// tomlDoc is an agent config file in TOML. Reads decode the file. An edit
// replaces, deletes or appends the tables of one server as text, so comments
// and every other line stay as they are. The edited text is decoded again and
// must hold the wanted server and the same other settings, or it is not used.
type tomlDoc struct {
	bom  bool
	text []byte // the file without its BOM
	root map[string]any
}

func parseTOML(data []byte) (document, error) {
	data, bom := stripBOM(data)
	root := map[string]any{}
	if _, err := toml.Decode(string(data), &root); err != nil {
		return nil, err
	}
	return &tomlDoc{bom: bom, text: data, root: root}, nil
}

// table returns the table at root. A missing table reads as empty.
func (d *tomlDoc) table(root []string) (map[string]any, error) {
	t := d.root
	for _, key := range root {
		next, ok := t[key]
		if !ok {
			return map[string]any{}, nil
		}
		m, ok := next.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%q is not a TOML table: %w", key, ErrConfigInvalid)
		}
		t = m
	}
	return t, nil
}

func (d *tomlDoc) entry(root []string, name string) (map[string]any, bool, error) {
	table, err := d.table(root)
	if err != nil {
		return nil, false, err
	}
	raw, ok := table[name]
	if !ok {
		return nil, false, nil
	}
	entry, ok := raw.(map[string]any)
	if !ok {
		return nil, false, fmt.Errorf("server %q is not a TOML table: %w", name, ErrConfigInvalid)
	}
	return entry, true, nil
}

func (d *tomlDoc) entries(root []string) (map[string]map[string]any, error) {
	table, err := d.table(root)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]any{}
	for key, value := range table {
		if entry, ok := value.(map[string]any); ok {
			out[key] = entry
		}
	}
	return out, nil
}

// set and update rewrite the tables of the server from entry. The user's keys
// keep their values, but the text of those tables is rendered again, so their
// comments and layout inside the server's tables are not kept.
func (d *tomlDoc) set(root []string, name string, entry map[string]any) error {
	return d.edit(root, name, entry)
}

func (d *tomlDoc) update(root []string, name string, entry map[string]any) error {
	return d.edit(root, name, entry)
}

func (d *tomlDoc) remove(root []string, name string) error {
	return d.edit(root, name, nil)
}

// edit replaces the tables of server name under root with the rendering of
// entry, or deletes them when entry is nil. A server that exists without tables
// of its own (an inline table or dotted keys) is an error, not rewritten.
// Headers are found in the text with multi-line strings protected, so a line
// inside such a string is never taken for a table.
func (d *tomlDoc) edit(root []string, name string, entry map[string]any) error {
	protected, restore, err := localenv.ProtectMultilineStrings(string(d.text))
	if err != nil {
		return err
	}
	lines := splitLines([]byte(protected))
	ranges := serverBlocks(lines, root, name)
	_, found, err := d.entry(root, name)
	if err != nil {
		return err
	}
	if found && len(ranges) == 0 {
		return fmt.Errorf("server %q is not defined by its own [%s] tables (inline table or dotted keys), so it cannot be edited; edit it by hand",
			name, strings.Join(append(root, name), "."))
	}
	// BurntSushi accepts a table header added to an inline table, which TOML
	// does not, so the root table must have a header of its own before we add to it.
	if d.defined(root) && !hasHeaderUnder(lines, root) {
		return fmt.Errorf("the [%s] table is defined inline or with dotted keys, so this CLI cannot add server %q to it; edit it by hand",
			strings.Join(root, "."), name)
	}

	nl := lineEnding(d.text)
	var block [][]byte
	if entry != nil {
		rendered, err := renderBlock(root, name, entry, nl)
		if err != nil {
			return err
		}
		block = splitLines(rendered)
	}
	text := restore(string(bytes.Join(spliceBlock(lines, ranges, block, nl), nil)))

	parsed := map[string]any{}
	if _, err := toml.Decode(text, &parsed); err != nil {
		return fmt.Errorf("server %q: the edited file does not decode: %w", name, err)
	}
	if err := checkEdit(d.root, parsed, root, name, entry); err != nil {
		return fmt.Errorf("server %q: %w", name, err)
	}
	d.text = []byte(text)
	d.root = parsed
	return nil
}

// spliceBlock returns lines with the tables at ranges replaced by block, or
// removed when block is nil. The first table takes block and the others are
// dropped, each with its separating blank lines.
func spliceBlock(lines [][]byte, ranges []lineRange, block [][]byte, nl string) [][]byte {
	var out [][]byte
	pos := 0
	placed := false
	for _, r := range ranges {
		out = append(out, lines[pos:r.start]...)
		pos = r.end
		if block != nil && !placed {
			out = append(out, block...)
			placed = true
			continue
		}
		out, pos = dropTable(out, lines, pos, nl)
	}
	out = append(out, lines[pos:]...)
	if block != nil && !placed {
		out = appendBlock(out, block, nl)
	}
	return out
}

// dropTable is called after a table is dropped, with the lines emitted so far
// and the index of the first line after the table. When nothing is emitted yet,
// the blank lines after the table go, so the file does not start with a blank
// line. Otherwise the blank lines around the table become one blank line, when
// there was one on either side of it and something follows.
func dropTable(out, lines [][]byte, pos int, nl string) ([][]byte, int) {
	if !hasContent(out) {
		for pos < len(lines) && isBlankLine(lines[pos]) {
			pos++
		}
		return out, pos
	}
	blank := false
	for len(out) > 0 && isBlankLine(out[len(out)-1]) {
		out = out[:len(out)-1]
		blank = true
	}
	for pos < len(lines) && isBlankLine(lines[pos]) {
		pos++
		blank = true
	}
	if blank && pos < len(lines) {
		out = append(out, []byte(nl))
	}
	return out, pos
}

func hasContent(lines [][]byte) bool {
	return slices.ContainsFunc(lines, func(line []byte) bool { return !isBlankLine(line) })
}

func isBlankLine(line []byte) bool {
	return strings.TrimSpace(string(line)) == ""
}

// checkEdit verifies an edit: the server is entry (or gone when entry is nil),
// and nothing else in the document changed.
func checkEdit(before, after map[string]any, root []string, name string, entry map[string]any) error {
	same, err := sameEntry(withoutServer(before, root, name), withoutServer(after, root, name))
	if err != nil {
		return err
	}
	if !same {
		return errors.New("the edit changed other settings")
	}
	got, found, err := (&tomlDoc{root: after}).entry(root, name)
	if err != nil {
		return err
	}
	if entry == nil {
		if found {
			return errors.New("the edit did not remove the server")
		}
		return nil
	}
	if !found {
		return errors.New("the edit did not add the server")
	}
	same, err = sameEntry(got, entry)
	if err != nil {
		return err
	}
	if !same {
		return errors.New("the edit wrote a different server")
	}
	return nil
}

// withoutServer returns doc without the server name under root. The tables on
// the way are copied, so doc is not changed. A table left empty is dropped, so
// a file without the table and one whose table only held the server compare equal.
func withoutServer(doc map[string]any, root []string, name string) map[string]any {
	out := maps.Clone(doc)
	if len(root) == 0 {
		delete(out, name)
		return out
	}
	child, ok := out[root[0]].(map[string]any)
	if !ok {
		return out
	}
	rest := withoutServer(child, root[1:], name)
	if len(rest) == 0 {
		delete(out, root[0])
	} else {
		out[root[0]] = rest
	}
	return out
}

// defined reports whether the key path root exists in the document.
func (d *tomlDoc) defined(root []string) bool {
	t := d.root
	for _, key := range root {
		next, ok := t[key]
		if !ok {
			return false
		}
		m, ok := next.(map[string]any)
		if !ok {
			return true
		}
		t = m
	}
	return true
}

// hasHeaderUnder reports whether a table header in lines is root or below it.
func hasHeaderUnder(lines [][]byte, root []string) bool {
	for _, line := range lines {
		parts, ok := tomlTableKey(string(line))
		if !ok || len(parts) < len(root) {
			continue
		}
		if slices.Equal(parts[:len(root)], root) {
			return true
		}
	}
	return false
}

// renderBlock renders the tables of one server: [root.name] and any sub-tables,
// such as [root.name.env]. The table header of root itself is dropped, and each
// line is written without the indentation the encoder adds.
func renderBlock(root []string, name string, entry map[string]any, nl string) ([]byte, error) {
	var tree any = map[string]any{name: entry}
	for _, key := range slices.Backward(root) {
		tree = map[string]any{key: tree}
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(tree); err != nil {
		return nil, err
	}
	header := "[" + strings.Join(root, ".") + "]"
	var out strings.Builder
	for line := range strings.SplitSeq(strings.TrimSuffix(buf.String(), "\n"), "\n") {
		line = strings.TrimLeft(line, " \t")
		if line == header {
			continue
		}
		out.WriteString(line)
		out.WriteString(nl)
	}
	return []byte(out.String()), nil
}

// appendBlock appends block after out, separated by one blank line.
func appendBlock(out, block [][]byte, nl string) [][]byte {
	if len(out) > 0 {
		if last := out[len(out)-1]; !bytes.HasSuffix(last, []byte("\n")) {
			out[len(out)-1] = append(slices.Clip(last), nl...)
		}
		out = append(out, []byte(nl))
	}
	return append(out, block...)
}

// lineRange is the half-open range of lines [start, end).
type lineRange struct {
	start, end int
}

// serverBlocks returns the line ranges of the tables that define server name
// under root: its [root.name] table and its sub-tables, each ending before the
// next table header. Blank and comment lines right before that header are not
// part of the range, since they describe the header that follows.
func serverBlocks(lines [][]byte, root []string, name string) []lineRange {
	var headers []int
	var keys [][]string
	for i, line := range lines {
		if parts, ok := tomlTableKey(string(line)); ok {
			headers = append(headers, i)
			keys = append(keys, parts)
		}
	}
	var ranges []lineRange
	for h, parts := range keys {
		if !isServerTable(parts, root, name) {
			continue
		}
		end := len(lines)
		if h+1 < len(headers) {
			end = headers[h+1]
		}
		for end > headers[h]+1 && isBlankOrComment(string(lines[end-1])) {
			end--
		}
		ranges = append(ranges, lineRange{start: headers[h], end: end})
	}
	return ranges
}

func isServerTable(parts, root []string, name string) bool {
	if len(parts) <= len(root) || parts[len(root)] != name {
		return false
	}
	for i := range root {
		if parts[i] != root[i] {
			return false
		}
	}
	return true
}

// tomlTableKey returns the key path of a table header line, such as
// [mcp_servers.name] or [[mcp_servers."a.b"]], with the quoted keys unquoted.
// Other lines, including an array element that starts with a bracket, report
// false.
func tomlTableKey(line string) ([]string, bool) {
	s := strings.TrimSpace(line)
	closing := "]"
	switch {
	case strings.HasPrefix(s, "[["):
		s, closing = s[2:], "]]"
	case strings.HasPrefix(s, "["):
		s = s[1:]
	default:
		return nil, false
	}
	var parts []string
	for {
		s = strings.TrimLeft(s, " \t")
		key, rest, ok := tomlKey(s)
		if !ok {
			return nil, false
		}
		parts = append(parts, key)
		s = strings.TrimLeft(rest, " \t")
		if !strings.HasPrefix(s, ".") {
			break
		}
		s = s[1:]
	}
	if !strings.HasPrefix(s, closing) {
		return nil, false
	}
	if tail := strings.TrimSpace(s[len(closing):]); tail != "" && !strings.HasPrefix(tail, "#") {
		return nil, false
	}
	return parts, true
}

// tomlKey reads one key at the start of s, a bare key or a basic ("...") or
// literal ('...') quoted key, and returns it with the rest of s.
func tomlKey(s string) (key, rest string, ok bool) {
	if s == "" {
		return "", "", false
	}
	switch s[0] {
	case '"':
		end := 1
		for end < len(s) && s[end] != '"' {
			if s[end] == '\\' {
				end++
			}
			end++
		}
		if end >= len(s) {
			return "", "", false
		}
		name, err := strconv.Unquote(s[:end+1])
		if err != nil {
			return "", "", false
		}
		return name, s[end+1:], true
	case '\'':
		end := strings.IndexByte(s[1:], '\'')
		if end < 0 {
			return "", "", false
		}
		return s[1 : end+1], s[end+2:], true
	}
	end := 0
	for end < len(s) && isBareKeyChar(s[end]) {
		end++
	}
	if end == 0 {
		return "", "", false
	}
	return s[:end], s[end:], true
}

func isBareKeyChar(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || '0' <= c && c <= '9' || c == '-' || c == '_'
}

func isBlankOrComment(line string) bool {
	s := strings.TrimSpace(line)
	return s == "" || strings.HasPrefix(s, "#")
}

// splitLines splits data after each newline, keeping the newlines.
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	for len(data) > 0 {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			lines = append(lines, data)
			break
		}
		lines = append(lines, data[:i+1])
		data = data[i+1:]
	}
	return lines
}

func (d *tomlDoc) bytes() ([]byte, error) {
	return withBOM(d.text, d.bom), nil
}
