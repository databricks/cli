package agentconfig

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

// jsonObject is a JSON object that keeps its keys in file order and each value
// as the original bytes, so a rewrite changes only the entry it was asked to.
type jsonObject struct {
	keys   []string
	values map[string]json.RawMessage
}

// parseJSONObject decodes a top-level JSON object. Empty input is an empty object.
func parseJSONObject(data []byte) (*jsonObject, error) {
	obj := &jsonObject{values: map[string]json.RawMessage{}}
	if len(bytes.TrimSpace(data)) == 0 {
		return obj, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('{') {
		return nil, errors.New("top-level value is not a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, errors.New("object key is not a string")
		}
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		obj.put(key, raw)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, errors.New("unexpected data after the JSON object")
	}
	return obj, nil
}

func (o *jsonObject) put(key string, raw json.RawMessage) {
	if _, ok := o.values[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.values[key] = raw
}

func (o *jsonObject) del(key string) {
	if _, ok := o.values[key]; !ok {
		return
	}
	delete(o.values, key)
	o.keys = slices.DeleteFunc(o.keys, func(k string) bool { return k == key })
}

// child returns the object under key, or an empty one when key is absent.
func (o *jsonObject) child(key string) (*jsonObject, error) {
	raw, ok := o.values[key]
	if !ok {
		return &jsonObject{values: map[string]json.RawMessage{}}, nil
	}
	child, err := parseJSONObject(raw)
	if err != nil {
		return nil, fmt.Errorf("%q is not a JSON object: %w", key, ErrConfigInvalid)
	}
	return child, nil
}

// edit applies fn to the object at path, creating missing levels and writing
// every changed level back into its parent.
func (o *jsonObject) edit(path []string, fn func(*jsonObject)) error {
	if len(path) == 0 {
		fn(o)
		return nil
	}
	child, err := o.child(path[0])
	if err != nil {
		return err
	}
	if err := child.edit(path[1:], fn); err != nil {
		return err
	}
	raw, err := child.compact()
	if err != nil {
		return err
	}
	o.put(path[0], raw)
	return nil
}

// compact serializes o without insignificant whitespace.
func (o *jsonObject) compact() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		name, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(o.values[key])
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// jsonDoc is an agent config file in JSON.
type jsonDoc struct {
	root *jsonObject
}

func parseJSON(data []byte) (document, error) {
	root, err := parseJSONObject(data)
	if err != nil {
		return nil, err
	}
	return &jsonDoc{root: root}, nil
}

// table returns the object at root. Missing levels read as empty objects.
func (d *jsonDoc) table(root []string) (*jsonObject, error) {
	obj := d.root
	for _, key := range root {
		next, err := obj.child(key)
		if err != nil {
			return nil, err
		}
		obj = next
	}
	return obj, nil
}

// asObject decodes raw as a JSON object. Other JSON values are reported as not ok.
func asObject(raw json.RawMessage) (map[string]any, bool) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, false
	}
	obj, ok := v.(map[string]any)
	return obj, ok
}

func (d *jsonDoc) entry(root []string, name string) (map[string]any, bool, error) {
	table, err := d.table(root)
	if err != nil {
		return nil, false, err
	}
	raw, ok := table.values[name]
	if !ok {
		return nil, false, nil
	}
	entry, ok := asObject(raw)
	if !ok {
		return nil, false, fmt.Errorf("server %q is not a JSON object: %w", name, ErrConfigInvalid)
	}
	return entry, true, nil
}

// entries returns the object-valued members under root. Other members are not
// server entries and are skipped.
func (d *jsonDoc) entries(root []string) (map[string]map[string]any, error) {
	table, err := d.table(root)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]any{}
	for _, key := range table.keys {
		if entry, ok := asObject(table.values[key]); ok {
			out[key] = entry
		}
	}
	return out, nil
}

func (d *jsonDoc) set(root []string, name string, entry map[string]any) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return d.root.edit(root, func(table *jsonObject) { table.put(name, raw) })
}

func (d *jsonDoc) remove(root []string, name string) error {
	return d.root.edit(root, func(table *jsonObject) { table.del(name) })
}

// bytes returns the document with two-space indentation and a trailing newline,
// which is what the agents write themselves.
func (d *jsonDoc) bytes() ([]byte, error) {
	compact, err := d.root.compact()
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, compact, "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}
