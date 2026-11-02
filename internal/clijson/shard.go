package clijson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

// The cli.json document genkit produces is ~10MB, so it is committed sharded
// to keep every file well under 1MB:
//
//	index.json                  "commands" without "services", plus the ordered service IDs
//	schemas/<pkg>.json          the "schemas" entries whose key is "<pkg>.<Type>"
//	commands/<pkg>/<Name>.json  the service with ID "<pkg>.<Name>"
//
// Shards carry each entry's raw JSON, so splitting and merging is lossless,
// including fields the contract types don't decode yet.
const (
	indexFile   = "index.json"
	schemasDir  = "schemas"
	commandsDir = "commands"
	servicesKey = "services"
)

// Split writes the sharded form of the cli.json document doc to dir,
// replacing any shards already there.
func Split(doc []byte, dir string) error {
	var root struct {
		Schemas  map[string]json.RawMessage `json:"schemas"`
		Commands map[string]json.RawMessage `json:"commands"`
	}
	// Reject unknown top-level blocks: ReadDir only reassembles these two.
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&root); err != nil {
		return fmt.Errorf("decode cli.json: %w", err)
	}

	var services []json.RawMessage
	if err := json.Unmarshal(root.Commands[servicesKey], &services); err != nil {
		return fmt.Errorf("decode commands.services: %w", err)
	}

	for _, sub := range []string{schemasDir, commandsDir} {
		if err := os.RemoveAll(filepath.Join(dir, sub)); err != nil {
			return err
		}
	}

	ids := make([]string, 0, len(services))
	for _, svc := range services {
		var s struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(svc, &s); err != nil {
			return fmt.Errorf("decode service: %w", err)
		}
		path, err := servicePath(dir, s.ID)
		if err != nil {
			return err
		}
		if err := writeJSON(path, svc); err != nil {
			return err
		}
		ids = append(ids, s.ID)
	}

	index := root.Commands
	idsJSON, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	index[servicesKey] = idsJSON
	if err := writeJSON(filepath.Join(dir, indexFile), index); err != nil {
		return err
	}

	byPackage := map[string]map[string]json.RawMessage{}
	for key, schema := range root.Schemas {
		pkg, _, ok := strings.Cut(key, ".")
		if !ok {
			return fmt.Errorf("schema key %q has no package prefix", key)
		}
		if byPackage[pkg] == nil {
			byPackage[pkg] = map[string]json.RawMessage{}
		}
		byPackage[pkg][key] = schema
	}
	for pkg, schemas := range byPackage {
		if err := writeJSON(filepath.Join(dir, schemasDir, pkg+".json"), schemas); err != nil {
			return err
		}
	}
	return nil
}

// ReadDir reassembles the cli.json document from the shards in dir.
func ReadDir(dir string) ([]byte, error) {
	var commands map[string]json.RawMessage
	if err := readJSON(filepath.Join(dir, indexFile), &commands); err != nil {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal(commands[servicesKey], &ids); err != nil {
		return nil, fmt.Errorf("decode %s services: %w", indexFile, err)
	}

	services := make([]json.RawMessage, 0, len(ids))
	for _, id := range ids {
		path, err := servicePath(dir, id)
		if err != nil {
			return nil, err
		}
		var svc json.RawMessage
		if err := readJSON(path, &svc); err != nil {
			return nil, err
		}
		services = append(services, svc)
	}
	servicesJSON, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}
	commands[servicesKey] = servicesJSON

	paths, err := filepath.Glob(filepath.Join(dir, schemasDir, "*.json"))
	if err != nil {
		return nil, err
	}
	schemas := map[string]json.RawMessage{}
	for _, path := range paths {
		var shard map[string]json.RawMessage
		if err := readJSON(path, &shard); err != nil {
			return nil, err
		}
		maps.Copy(schemas, shard)
	}

	return json.Marshal(map[string]any{
		"schemas":  schemas,
		"commands": commands,
	})
}

// servicePath maps service ID "<pkg>.<Name>" to its shard under dir.
func servicePath(dir, id string) (string, error) {
	pkg, name, ok := strings.Cut(id, ".")
	if !ok || pkg == "" || name == "" {
		return "", fmt.Errorf("service id %q is not of the form <pkg>.<Name>", id)
	}
	return filepath.Join(dir, commandsDir, pkg, name+".json"), nil
}

// writeJSON writes v with the same 2-space indent genkit uses, plus a trailing
// newline.
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
