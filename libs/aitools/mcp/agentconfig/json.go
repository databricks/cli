package agentconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tailscale/hujson"
)

// jsonDoc is an agent config file in JSON with comments (JSONC) and trailing
// commas also accepted. Edits are RFC 6902 patches applied to the parsed tree,
// so every byte outside the changed member is kept.
type jsonDoc struct {
	bom bool
	// fresh is set when the file had no members, so its layout is ours to
	// choose: the output is re-indented with two spaces.
	fresh bool
	root  hujson.Value
}

func parseJSON(data []byte) (document, error) {
	data, bom := stripBOM(data)
	if len(bytes.TrimSpace(data)) == 0 {
		data = []byte("{}")
	}
	root, err := hujson.Parse(data)
	if err != nil {
		return nil, err
	}
	obj, ok := root.Value.(*hujson.Object)
	if !ok {
		return nil, fmt.Errorf("top-level value is not a JSON object: %w", ErrConfigInvalid)
	}
	return &jsonDoc{bom: bom, fresh: len(obj.Members) == 0, root: root}, nil
}

// object returns the object at path, or nil when a level is missing. A level
// that exists but is not an object is an error.
func (d *jsonDoc) object(path []string) (*hujson.Object, error) {
	obj := d.root.Value.(*hujson.Object)
	for i, key := range path {
		member := findMember(obj, key)
		if member == nil {
			return nil, nil
		}
		next, ok := member.Value.Value.(*hujson.Object)
		if !ok {
			return nil, fmt.Errorf("%q is not a JSON object: %w", strings.Join(path[:i+1], "."), ErrConfigInvalid)
		}
		obj = next
	}
	return obj, nil
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
	if err := json.Unmarshal(std, &out); err != nil || out == nil {
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
	parent, err := d.object(root)
	if err != nil {
		return err
	}
	entryNew := parent == nil || findMember(parent, name) == nil
	value, err := encodeIndented(entry, indent(n+1))
	if err != nil {
		return err
	}
	var ops []patch
	for i := first; i < n; i++ {
		ops = append(ops, patch{op: "add", path: jsonPointer(root[:i+1]...), value: []byte("{\n" + indent(i+1) + "}")})
	}
	ops = append(ops, patch{op: "add", path: jsonPointer(append(root, name)...), value: value})
	if err := d.apply(ops); err != nil {
		return err
	}
	for i := first; i < n; i++ {
		obj, err := d.object(root[:i])
		if err != nil {
			return err
		}
		place(obj, root[i], i+1, true)
	}
	obj, err := d.object(root)
	if err != nil {
		return err
	}
	place(obj, name, n+1, entryNew)
	return nil
}

// place lays out a member that set wrote: the value follows the colon after a
// space, and a new member starts its own line at depth.
func place(obj *hujson.Object, name string, depth int, newMember bool) {
	m := findMember(obj, name)
	m.Value.BeforeExtra = hujson.Extra(" ")
	if newMember {
		m.Name.BeforeExtra = hujson.Extra("\n" + indent(depth))
	}
}

// indent returns the two-space indentation of the given depth.
func indent(depth int) string {
	return strings.Repeat("  ", depth)
}

func (d *jsonDoc) remove(root []string, name string) error {
	return d.apply([]patch{{op: "remove", path: jsonPointer(append(root, name)...)}})
}

func (d *jsonDoc) apply(ops []patch) error {
	data, err := marshalPatch(ops)
	if err != nil {
		return err
	}
	return d.root.Patch(data)
}

// bytes returns the document. A file that had no members is written with
// two-space indentation and a trailing newline, as the agents write their own.
func (d *jsonDoc) bytes() ([]byte, error) {
	var out []byte
	if d.fresh {
		std, err := hujson.Standardize(d.root.Pack())
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		if err := json.Indent(&buf, std, "", "  "); err != nil {
			return nil, err
		}
		buf.WriteByte('\n')
		out = buf.Bytes()
	} else {
		out = d.root.Pack()
	}
	return withBOM(out, d.bom), nil
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
