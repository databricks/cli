package agentconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
)

// RFC 6902 operations used to edit a document.
const (
	opAdd     = "add"
	opReplace = "replace"
	opRemove  = "remove"
)

// jsonDoc is an agent config file in JSON with comments (JSONC) and trailing
// commas also accepted. Edits are RFC 6902 patches applied to the parsed tree,
// so every byte outside the changed member is kept.
type jsonDoc struct {
	bom bool
	// nl is the file's line ending. Inserted text uses it, so the file keeps a
	// single line ending.
	nl   string
	root hujson.Value
}

func parseJSON(data []byte) (document, error) {
	data, bom := stripBOM(data)
	if len(bytes.TrimSpace(data)) == 0 {
		// A missing or blank file is an empty object, written with a trailing newline.
		data = []byte("{}\n")
	}
	root, err := hujson.Parse(data)
	if err != nil {
		return nil, err
	}
	if _, ok := root.Value.(*hujson.Object); !ok {
		return nil, errors.New("top-level value is not a JSON object")
	}
	return &jsonDoc{bom: bom, nl: lineEnding(data), root: root}, nil
}

// object returns the object at path, or nil when a level is missing. A level
// that exists but is not an object is an error. Every object on the way must have
// unique member names, since agents read the last of two equal names while an
// edit would change the first.
func (d *jsonDoc) object(path []string) (*hujson.Object, error) {
	obj := d.root.Value.(*hujson.Object)
	if err := checkUnique(obj); err != nil {
		return nil, err
	}
	for i, key := range path {
		member := findMember(obj, key)
		if member == nil {
			return nil, nil
		}
		next, ok := member.Value.Value.(*hujson.Object)
		if !ok {
			return nil, fmt.Errorf("%q is not a JSON object: %w", strings.Join(path[:i+1], "."), ErrConfigInvalid)
		}
		if err := checkUnique(next); err != nil {
			return nil, err
		}
		obj = next
	}
	return obj, nil
}

func checkUnique(obj *hujson.Object) error {
	seen := map[string]bool{}
	for _, m := range obj.Members {
		name := memberName(m)
		if seen[name] {
			return fmt.Errorf("duplicate key %q: %w", name, ErrConfigInvalid)
		}
		seen[name] = true
	}
	return nil
}

func findMember(obj *hujson.Object, name string) *hujson.ObjectMember {
	for i := range obj.Members {
		if memberName(obj.Members[i]) == name {
			return &obj.Members[i]
		}
	}
	return nil
}

func memberName(m hujson.ObjectMember) string {
	return m.Name.Value.(hujson.Literal).String()
}

// decodeObject returns the value as a JSON object, ignoring comments and
// trailing commas. The tree itself is not changed.
func decodeObject(v *hujson.Value) (map[string]any, bool) {
	std, err := hujson.Standardize(v.Pack())
	if err != nil {
		return nil, false
	}
	var out map[string]any
	if err := decodeJSON(std, &out); err != nil || out == nil {
		return nil, false
	}
	return out, true
}

func (d *jsonDoc) entry(root []string, name string) (map[string]any, bool, error) {
	table, err := d.object(root)
	if err != nil || table == nil {
		return nil, false, err
	}
	member := findMember(table, name)
	if member == nil {
		return nil, false, nil
	}
	if obj, ok := member.Value.Value.(*hujson.Object); ok {
		if err := checkUnique(obj); err != nil {
			return nil, false, err
		}
	}
	entry, ok := decodeObject(&member.Value)
	if !ok {
		return nil, false, fmt.Errorf("server %q is not a JSON object: %w", name, ErrConfigInvalid)
	}
	return entry, true, nil
}

// entries returns the object-valued members under root. Other members are not
// server entries and are skipped.
func (d *jsonDoc) entries(root []string) (map[string]map[string]any, error) {
	out := map[string]map[string]any{}
	table, err := d.object(root)
	if err != nil || table == nil {
		return out, err
	}
	for _, member := range table.Members {
		if entry, ok := decodeObject(&member.Value); ok {
			out[memberName(member)] = entry
		}
	}
	return out, nil
}

// set stores entry as name under root, creating the missing levels of root. The
// layout follows the two-space indentation the agents write: a member sits at
// the depth of its key, and a new member starts its own line.
func (d *jsonDoc) set(root []string, name string, entry map[string]any) error {
	n := len(root)
	first := n
	for i := range root {
		obj, err := d.object(root[:i+1])
		if err != nil {
			return err
		}
		if obj == nil {
			first = i
			break
		}
	}
	// Only the deepest existing object can be empty: the ones above it hold the
	// next level of root.
	holder, err := d.object(root[:first])
	if err != nil {
		return err
	}
	emptied := len(holder.Members) == 0
	parent, err := d.object(root)
	if err != nil {
		return err
	}
	entryNew := parent == nil || findMember(parent, name) == nil
	value, err := d.encodeValue(entry, n+1)
	if err != nil {
		return err
	}
	var ops []patch
	for i := first; i < n; i++ {
		ops = append(ops, patch{op: opAdd, path: jsonPointer(root[:i+1]...), value: []byte("{" + d.nl + indent(i+1) + "}")})
	}
	ops = append(ops, patch{op: opAdd, path: jsonPointer(append(root, name)...), value: value})
	if err := d.apply(ops); err != nil {
		return err
	}
	if emptied {
		holder, err := d.object(root[:first])
		if err != nil {
			return err
		}
		holder.AfterExtra = relayout(holder.AfterExtra, d.nl+indent(first))
	}
	for i := first; i < n; i++ {
		obj, err := d.object(root[:i])
		if err != nil {
			return err
		}
		place(obj, root[i], i+1, true, d.nl)
	}
	obj, err := d.object(root)
	if err != nil {
		return err
	}
	place(obj, name, n+1, entryNew, d.nl)
	return nil
}

// update changes the launch keys of the server name under root to those of
// entry. A launch key that entry lacks is removed, and one that is missing from
// the file is added. A launch key whose value is unchanged is not touched, and
// the other keys of the server keep their bytes. name must exist.
func (d *jsonDoc) update(root []string, name string, entry map[string]any) error {
	current, _, err := d.entry(root, name)
	if err != nil {
		return err
	}
	depth := len(root) + 2
	var ops []patch
	var added, removed []string
	for _, key := range launchKeys {
		want, inEntry := entry[key]
		have, inFile := current[key]
		switch {
		case inEntry && !inFile:
			value, err := d.encodeValue(want, depth)
			if err != nil {
				return err
			}
			ops = append(ops, patch{op: opAdd, path: jsonPointer(append(root, name, key)...), value: value})
			added = append(added, key)
		case inFile && !inEntry:
			removed = append(removed, key)
		case inEntry:
			same, err := sameEntry(map[string]any{key: have}, map[string]any{key: want})
			if err != nil {
				return err
			}
			if same {
				continue
			}
			value, err := d.encodeValue(want, depth)
			if err != nil {
				return err
			}
			ops = append(ops, patch{op: opReplace, path: jsonPointer(append(root, name, key)...), value: value})
		}
	}
	if len(ops) > 0 {
		if err := d.apply(ops); err != nil {
			return err
		}
	}
	for _, key := range removed {
		if err := d.remove(append(root, name), key); err != nil {
			return err
		}
	}
	server, err := d.server(root, name)
	if err != nil {
		return err
	}
	for _, key := range added {
		place(server, key, depth, true, d.nl)
	}
	return nil
}

// server returns the object of the server name under root. The entry must exist
// and be an object.
func (d *jsonDoc) server(root []string, name string) (*hujson.Object, error) {
	table, err := d.object(root)
	if err != nil {
		return nil, err
	}
	// entry decoded the member as an object, so its value is an object.
	return findMember(table, name).Value.Value.(*hujson.Object), nil
}

// encodeValue encodes v for a member at depth, with the file's line endings.
func (d *jsonDoc) encodeValue(v any, depth int) ([]byte, error) {
	value, err := encodeIndented(v, indent(depth))
	if err != nil {
		return nil, err
	}
	return bytes.ReplaceAll(value, []byte("\n"), []byte(d.nl)), nil
}

// place lays out a member that set wrote: the value follows the colon after a
// space, and a new member starts its own line at depth. Comments before the
// member stay where they are.
func place(obj *hujson.Object, name string, depth int, newMember bool, nl string) {
	m := findMember(obj, name)
	if m.Value.BeforeExtra == nil {
		m.Value.BeforeExtra = hujson.Extra(" ")
	}
	if newMember {
		m.Name.BeforeExtra = relayout(m.Name.BeforeExtra, nl+indent(depth))
	}
}

// relayout keeps the comments in extra and replaces the whitespace after the
// last one with layout. A line comment ends at its newline, which layout provides.
func relayout(extra hujson.Extra, layout string) hujson.Extra {
	kept := bytes.TrimRight(extra, " \t\r\n")
	return append(append(hujson.Extra{}, kept...), layout...)
}

// indent returns the two-space indentation of the given depth.
func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

// remove deletes name under root. A missing name is a no-op. Removing the last
// member keeps the file's trailing-comma state: a strict file gets no comma after
// the member that becomes last.
func (d *jsonDoc) remove(root []string, name string) error {
	obj, err := d.object(root)
	if err != nil || obj == nil || findMember(obj, name) == nil {
		return err
	}
	trailing := hasTrailingComma(obj)
	if err := d.apply([]patch{{op: opRemove, path: jsonPointer(append(root, name)...)}}); err != nil {
		return err
	}
	obj, err = d.object(root)
	if err != nil {
		return err
	}
	setTrailingComma(obj, trailing)
	return nil
}

// hasTrailingComma reports whether the last member of obj is followed by a
// comma. hujson keeps that state in the last member's AfterExtra.
func hasTrailingComma(obj *hujson.Object) bool {
	return len(obj.Members) > 0 && obj.Members[len(obj.Members)-1].Value.AfterExtra != nil
}

// setTrailingComma gives the last member of obj a trailing comma when on is set,
// and removes it otherwise. The whitespace before a removed comma moves to the
// closing brace, so it stays where it was.
func setTrailingComma(obj *hujson.Object, on bool) {
	if len(obj.Members) == 0 {
		return
	}
	last := &obj.Members[len(obj.Members)-1].Value
	switch {
	case on && last.AfterExtra == nil:
		last.AfterExtra = hujson.Extra{}
	case !on && last.AfterExtra != nil:
		obj.AfterExtra = append(append(hujson.Extra{}, last.AfterExtra...), obj.AfterExtra...)
		last.AfterExtra = nil
	}
}

func (d *jsonDoc) apply(ops []patch) error {
	data, err := marshalPatch(ops)
	if err != nil {
		return err
	}
	return d.root.Patch(data)
}

func (d *jsonDoc) bytes() ([]byte, error) {
	return withBOM(d.root.Pack(), d.bom), nil
}

// patch is one RFC 6902 operation. value holds encoded JSON, spliced in as
// given: json.Marshal would escape <, > and & inside it.
type patch struct {
	op    string
	path  string
	value []byte
}

func marshalPatch(ops []patch) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, op := range ops {
		if i > 0 {
			buf.WriteByte(',')
		}
		path, err := encodeJSON(op.path)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&buf, `{"op":%q,"path":%s`, op.op, path)
		if op.value != nil {
			buf.WriteString(`,"value":`)
			buf.Write(op.value)
		}
		buf.WriteByte('}')
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// jsonPointer builds an RFC 6901 JSON Pointer from path segments.
func jsonPointer(segments ...string) string {
	var b strings.Builder
	r := strings.NewReplacer("~", "~0", "/", "~1")
	for _, s := range segments {
		b.WriteByte('/')
		b.WriteString(r.Replace(s))
	}
	return b.String()
}

// encodeJSON encodes v as compact JSON without escaping <, > and &.
func encodeJSON(v any) ([]byte, error) {
	return encodeWith(v, "", "")
}

// encodeIndented encodes v with two-space indentation. Lines after the first
// start with prefix, the indentation of the member the value belongs to.
func encodeIndented(v any, prefix string) ([]byte, error) {
	return encodeWith(v, prefix, "  ")
}

func encodeWith(v any, prefix, step string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent(prefix, step)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
