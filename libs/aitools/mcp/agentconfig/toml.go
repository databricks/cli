package agentconfig

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
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

func (d *tomlDoc) set(root []string, name string, entry map[string]any) error {
	return d.edit(root, name, entry)
}

func (d *tomlDoc) remove(root []string, name string) error {
	return d.edit(root, name, nil)
}

// edit replaces the tables of server name under root with the rendering of
// entry, or deletes them when entry is nil. A server that exists without tables
// of its own (an inline table or dotted keys) is an error, not rewritten.
func (d *tomlDoc) edit(root []string, name string, entry map[string]any) error {
	nl := "\n"
	if bytes.Contains(d.text, []byte("\r\n")) {
		nl = "\r\n"
	}
	lines := splitLines(d.text)
	ranges := serverBlocks(lines, root, name)
	if _, found, err := d.entry(root, name); err != nil {
		return err
	} else if found && len(ranges) == 0 {
		return fmt.Errorf("server %q is not defined by its own [%s] tables (inline table or dotted keys), so it cannot be edited; edit it by hand",
			name, strings.Join(append(root, name), "."))
	}
	// BurntSushi accepts a table header added to an inline table, which TOML
	// does not, so the root table must have a header of its own before we add to it.
	if d.defined(root) && !hasHeaderUnder(lines, root) {
		return fmt.Errorf("the [%s] table is defined inline or with dotted keys, so this CLI cannot add server %q to it; edit it by hand",
			strings.Join(root, "."), name)
	}

	var block []byte
	if entry != nil {
		var err error
		if block, err = renderBlock(root, name, entry, nl); err != nil {
			return err
		}
	}
	var out []byte
	pos := 0
	placed := block == nil
	for _, r := range ranges {
		start := r.start
		// Removing a block also removes the blank line that separated it from the
		// lines before, so the file does not gain a double blank line.
		if block == nil && start > pos && strings.TrimSpace(string(lines[start-1])) == "" {
			start--
		}
		for _, line := range lines[pos:start] {
			out = append(out, line...)
		}
		if !placed {
			out = append(out, block...)
			placed = true
		}
		pos = r.end
	}
	for _, line := range lines[pos:] {
		out = append(out, line...)
	}
	if !placed {
		out = appendBlock(out, block, nl)
	}

	parsed := map[string]any{}
	if _, err := toml.Decode(string(out), &parsed); err != nil {
		return fmt.Errorf("server %q: the edited file does not decode: %w", name, err)
	}
	if err := checkEdit(d.root, parsed, root, name, entry); err != nil {
		return fmt.Errorf("server %q: %w", name, err)
	}
	d.text = out
	d.root = parsed
	return nil
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
func appendBlock(out, block []byte, nl string) []byte {
	if len(out) > 0 {
		if out[len(out)-1] != '\n' {
			out = append(out, nl...)
		}
		out = append(out, nl...)
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
// [mcp_servers.name] or [[mcp_servers.name]]. Other lines report false.
func tomlTableKey(line string) ([]string, bool) {
	s := strings.TrimSpace(line)
	if !strings.HasPrefix(s, "[") {
		return nil, false
	}
	s = strings.TrimLeft(s, "[")
	end := strings.IndexByte(s, ']')
	if end < 0 {
		return nil, false
	}
	var parts []string
	for p := range strings.SplitSeq(s[:end], ".") {
		parts = append(parts, strings.Trim(strings.TrimSpace(p), `"'`))
	}
	return parts, true
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
