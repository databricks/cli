package agentconfig

import (
	"bytes"
	"fmt"

	"github.com/BurntSushi/toml"
)

// tomlDoc is an agent config file in TOML, decoded into plain maps. Encoding it
// again drops comments and sorts keys; Upsert and Remove write only when an
// entry changes, so an untouched file keeps its comments.
type tomlDoc struct {
	root map[string]any
}

func parseTOML(data []byte) (document, error) {
	root := map[string]any{}
	if _, err := toml.Decode(string(data), &root); err != nil {
		return nil, err
	}
	return &tomlDoc{root: root}, nil
}

// table returns the table at root. A missing table reads as empty, and is
// created when create is set.
func (d *tomlDoc) table(root []string, create bool) (map[string]any, error) {
	t := d.root
	for _, key := range root {
		next, ok := t[key]
		if !ok {
			if !create {
				return map[string]any{}, nil
			}
			m := map[string]any{}
			t[key] = m
			t = m
			continue
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
	table, err := d.table(root, false)
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
	table, err := d.table(root, false)
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
	table, err := d.table(root, true)
	if err != nil {
		return err
	}
	table[name] = entry
	return nil
}

func (d *tomlDoc) remove(root []string, name string) error {
	table, err := d.table(root, false)
	if err != nil {
		return err
	}
	delete(table, name)
	return nil
}

func (d *tomlDoc) bytes() ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(d.root); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
