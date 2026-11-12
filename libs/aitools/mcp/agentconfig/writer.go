// Package agentconfig reads and writes the user-scope MCP server entries of
// coding agents. Each write reads the whole file, changes one entry and writes
// the file back atomically; everything else in the file is kept.
package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/atomicfile"
)

// Status is the outcome of Upsert.
type Status string

const (
	StatusAdded     Status = "added"
	StatusUpdated   Status = "updated"
	StatusUnchanged Status = "unchanged"
)

var (
	// ErrNotOwned means an entry with the name exists and was not written by
	// this CLI. Upsert needs force to replace it, and Remove never deletes it.
	ErrNotOwned = errors.New("entry was not written by databricks")
	// ErrConfigInvalid means an agent config file exists but cannot be parsed.
	// Such a file is never written to.
	ErrConfigInvalid = errors.New("agent config file cannot be parsed")
)

// document is an agent config file in memory.
type document interface {
	// entry returns the server entry name under root, and whether it exists.
	entry(root []string, name string) (map[string]any, bool, error)
	// entries returns the object-valued members under root, keyed by name.
	entries(root []string) (map[string]map[string]any, error)
	// set stores entry as name under root, creating missing tables.
	set(root []string, name string, entry map[string]any) error
	// remove deletes name under root. It is a no-op when name is absent.
	remove(root []string, name string) error
	// bytes serializes the document.
	bytes() ([]byte, error)
}

// Upsert writes a stdio server name for agent a, which runs cmd with args and
// env. It replaces an entry this CLI wrote, or any entry when force is set, and
// leaves the file alone when the entry already matches. It returns the config
// file path.
func Upsert(ctx context.Context, a *agents.Agent, name, cmd string, args []string, env map[string]string, force bool) (string, Status, error) {
	spec, err := specOf(a)
	if err != nil {
		return "", "", err
	}
	path, err := spec.Path(ctx)
	if err != nil {
		return "", "", err
	}
	doc, err := load(spec, path)
	if err != nil {
		return path, "", err
	}
	root := rootPath(spec)
	want := spec.Entry(cmd, args, env)
	current, found, err := doc.entry(root, name)
	if err != nil {
		return path, "", fmt.Errorf("%s: %w", path, err)
	}
	if found {
		if !spec.IsOurs(current) && !force {
			return path, "", fmt.Errorf("%s: server %q: %w", path, name, ErrNotOwned)
		}
		same, err := sameEntry(current, want)
		if err != nil {
			return path, "", err
		}
		if same {
			return path, StatusUnchanged, nil
		}
	}
	if err := doc.set(root, name, want); err != nil {
		return path, "", fmt.Errorf("%s: %w", path, err)
	}
	if err := save(path, doc); err != nil {
		return path, "", err
	}
	if found {
		return path, StatusUpdated, nil
	}
	return path, StatusAdded, nil
}

// Remove deletes server name from agent a's config. It reports false, without
// writing, when there is no such entry, and refuses an entry this CLI did not
// write.
func Remove(ctx context.Context, a *agents.Agent, name string) (bool, error) {
	spec, err := specOf(a)
	if err != nil {
		return false, err
	}
	path, err := spec.Path(ctx)
	if err != nil {
		return false, err
	}
	doc, err := load(spec, path)
	if err != nil {
		return false, err
	}
	root := rootPath(spec)
	current, found, err := doc.entry(root, name)
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if !found {
		return false, nil
	}
	if !spec.IsOurs(current) {
		return false, fmt.Errorf("%s: server %q: %w", path, name, ErrNotOwned)
	}
	if err := doc.remove(root, name); err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	if err := save(path, doc); err != nil {
		return false, err
	}
	return true, nil
}

// Read returns every server entry in agent a's config, keyed by name. A missing
// file has no entries.
func Read(ctx context.Context, a *agents.Agent) (map[string]map[string]any, error) {
	spec, err := specOf(a)
	if err != nil {
		return nil, err
	}
	path, err := spec.Path(ctx)
	if err != nil {
		return nil, err
	}
	doc, err := load(spec, path)
	if err != nil {
		return nil, err
	}
	entries, err := doc.entries(rootPath(spec))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return entries, nil
}

func specOf(a *agents.Agent) (*mcpconfig.Spec, error) {
	if a.MCP == nil {
		return nil, fmt.Errorf("agent %s does not take MCP servers", a.DisplayName)
	}
	return a.MCP, nil
}

func rootPath(spec *mcpconfig.Spec) []string {
	return strings.Split(spec.Root, ".")
}

// load reads and parses the config at path. A missing file is an empty document.
func load(spec *mcpconfig.Spec, path string) (document, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var doc document
	switch spec.Format {
	case mcpconfig.FormatJSON:
		doc, err = parseJSON(data)
	case mcpconfig.FormatTOML:
		doc, err = parseTOML(data)
	default:
		return nil, fmt.Errorf("unknown config format %d", spec.Format)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %w", path, ErrConfigInvalid, err)
	}
	return doc, nil
}

// save writes doc to path atomically. The file keeps its mode; a new file gets
// 0600 because agent configs can hold auth state.
func save(path string, doc document) error {
	data, err := doc.bytes()
	if err != nil {
		return fmt.Errorf("serialize %s: %w", path, err)
	}
	perm := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	// Rename would replace a symlink (dotfile managers link configs) with a
	// regular file, so write to the link's target.
	target := path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		target = resolved
	}
	if err := atomicfile.Write(target, data, perm, atomicfile.MkDir(0o755)); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// sameEntry compares two entries as JSON values, so the in-memory Go types of a
// freshly built entry and one decoded from a file compare equal.
func sameEntry(a, b map[string]any) (bool, error) {
	na, err := normalize(a)
	if err != nil {
		return false, err
	}
	nb, err := normalize(b)
	if err != nil {
		return false, err
	}
	return reflect.DeepEqual(na, nb), nil
}

func normalize(entry map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(entry)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}
