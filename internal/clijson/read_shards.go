package clijson

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strings"
)

func shardIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for position, character := range name {
		if character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character == '_' || position > 0 && character >= '0' && character <= '9' {
			continue
		}
		return false
	}
	return true
}

func readShardFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("shard %s is not a regular file", path)
	}
	if info.Size() >= MaxShardBytes {
		return nil, fmt.Errorf("shard %s must be smaller than %d bytes", path, MaxShardBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, MaxShardBytes))
	if err != nil {
		return nil, err
	}
	if len(content) >= MaxShardBytes {
		return nil, fmt.Errorf("shard %s must be smaller than %d bytes", path, MaxShardBytes)
	}
	return content, nil
}

func ReadDir(dir string) ([]byte, error) {
	content, err := readShardFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, err
	}
	var index ShardIndex
	if err := json.Unmarshal(content, &index); err != nil {
		return nil, fmt.Errorf("parse shard index: %w", err)
	}
	if index.Version != ShardFormatVersion {
		return nil, fmt.Errorf("unsupported shard format version %d", index.Version)
	}
	if len(index.Schemas) == 0 || len(index.Services) == 0 || index.Commands == nil || len(index.Commands.Services) != 0 {
		return nil, errors.New("shard index must list schemas and services, with command metadata only")
	}
	var metadata struct {
		Commands map[string]json.RawMessage `json:"commands"`
	}
	if err := json.Unmarshal(content, &metadata); err != nil {
		return nil, err
	}
	expected := map[string]bool{}
	readListed := func(path string, entry ShardJSON) ([]byte, error) {
		if expected[path] {
			return nil, fmt.Errorf("duplicate shard %s", path)
		}
		expected[path] = true
		content, err := readShardFile(filepath.Join(dir, path))
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(content)
		if hex.EncodeToString(digest[:]) != entry.SHA256 {
			return nil, fmt.Errorf("checksum mismatch for shard %s", path)
		}
		return content, nil
	}
	schemas := map[string]json.RawMessage{}
	for _, entry := range index.Schemas {
		if !shardIdentifier(entry.ID) {
			return nil, fmt.Errorf("invalid schema package %q", entry.ID)
		}
		content, err := readListed(filepath.Join("schemas", entry.ID+".json"), entry)
		if err != nil {
			return nil, err
		}
		var shard map[string]json.RawMessage
		if err := json.Unmarshal(content, &shard); err != nil {
			return nil, fmt.Errorf("parse schema package %s: %w", entry.ID, err)
		}
		if len(shard) == 0 {
			return nil, fmt.Errorf("empty schema package %s", entry.ID)
		}
		for name, schema := range shard {
			pkg, typeName, found := strings.Cut(name, ".")
			if !found || pkg != entry.ID || !shardIdentifier(typeName) || string(schema) == "null" {
				return nil, fmt.Errorf("invalid schema %q in package %s", name, entry.ID)
			}
			if _, exists := schemas[name]; exists {
				return nil, fmt.Errorf("duplicate schema %q", name)
			}
		}
		maps.Copy(schemas, shard)
	}
	services := make([]json.RawMessage, 0, len(index.Services))
	for _, entry := range index.Services {
		pkg, name, found := strings.Cut(entry.ID, ".")
		if !found || !shardIdentifier(pkg) || !shardIdentifier(name) {
			return nil, fmt.Errorf("invalid service ID %q", entry.ID)
		}
		content, err := readListed(filepath.Join("commands", pkg, name+".json"), entry)
		if err != nil {
			return nil, err
		}
		var service struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(content, &service); err != nil {
			return nil, fmt.Errorf("parse service %s: %w", entry.ID, err)
		}
		if service.ID != entry.ID {
			return nil, fmt.Errorf("service ID %q does not match indexed ID %q", service.ID, entry.ID)
		}
		services = append(services, json.RawMessage(content))
	}
	for _, sub := range []string{"schemas", "commands"} {
		err := filepath.WalkDir(filepath.Join(dir, sub), func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			if !expected[relative] {
				return fmt.Errorf("unexpected shard %s", relative)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	servicesJSON, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}
	metadata.Commands["services"] = servicesJSON
	return json.Marshal(map[string]any{"schemas": schemas, "commands": metadata.Commands})
}
