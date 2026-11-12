// Package agentconfig reads and writes the user-scope MCP server entries of
// coding agents. A write changes only the bytes of the one entry it is asked to
// change, writes the file back atomically, and keeps comments and layout
// elsewhere in the file.
package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
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

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// document is an agent config file in memory.
type document interface {
	// entry returns the server entry name under root, and whether it exists.
	entry(root []string, name string) (map[string]any, bool, error)
	// entries returns the object-valued members under root, keyed by name.
	entries(root []string) (map[string]map[string]any, error)
	// set stores entry as name under root, creating missing levels.
	set(root []string, name string, entry map[string]any) error
	// remove deletes name under root. It is a no-op when name is absent.
	remove(root []string, name string) error
	// bytes serializes the document.
	bytes() ([]byte, error)
}

// builderKeys are the entry keys the agent builders write. Upsert overwrites
// them. Any other key was added by the user (for example disabled or enabled)
// and is kept when an entry we wrote is updated.
var builderKeys = []string{"command", "args", "env", "environment", "type", "tools"}

// Upsert writes a stdio server name for agent a, which runs cmd with args and
// env. An entry this CLI wrote is updated in its builder keys, and keys the user
// added to it are kept. A foreign entry is replaced only when force is set. The
// file is left alone when the entry already matches. cmd must be the databricks
// binary, or the entry could never be recognized as ours. It returns the config
// file path.
func Upsert(ctx context.Context, a *agents.Agent, name, cmd string, args []string, env map[string]string, force bool) (string, Status, error) {
	if !mcpconfig.ValidServerName(name) {
		return "", "", fmt.Errorf("server name %q must contain only letters, digits, '-' and '_'", name)
	}
	spec, err := specOf(a)
	if err != nil {
		return "", "", err
	}
	want := spec.Entry(cmd, args, env)
	ours, err := normalize(want)
	if err != nil {
		return "", "", err
	}
	if !mcpconfig.IsBinary(cmd, "databricks", "databricks.exe") {
		return "", "", fmt.Errorf("server %q: command %q must be the databricks binary, named databricks or databricks.exe", name, cmd)
	}
	if !mcpconfig.IsDatabricksEntry(ours) {
		return "", "", fmt.Errorf("server %q: args must start with aitools mcp serve", name)
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
	current, found, err := doc.entry(root, name)
	if err != nil {
		return path, "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if found {
		if mcpconfig.IsDatabricksEntry(current) {
			same, err := sameEntry(builderKeysOf(current), builderKeysOf(want))
			if err != nil {
				return path, "", err
			}
			if same {
				return path, StatusUnchanged, nil
			}
			want = keepUserKeys(current, want)
		} else {
			if !force {
				return path, "", fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, ErrNotOwned)
			}
			same, err := sameEntry(current, want)
			if err != nil {
				return path, "", err
			}
			if same {
				return path, StatusUnchanged, nil
			}
		}
	}
	if err := doc.set(root, name, want); err != nil {
		return path, "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
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
		return false, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if !found {
		return false, nil
	}
	if !mcpconfig.IsDatabricksEntry(current) {
		return false, fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, ErrNotOwned)
	}
	if err := doc.remove(root, name); err != nil {
		return false, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
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
		return nil, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
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
		return nil, fmt.Errorf("read %s: %w", filepath.ToSlash(path), err)
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
		return nil, fmt.Errorf("%s: %w: %w", filepath.ToSlash(path), ErrConfigInvalid, err)
	}
	return doc, nil
}

// save writes doc to path atomically. The file keeps its mode; a new file gets
// 0600 because agent configs can hold auth state.
func save(path string, doc document) error {
	data, err := doc.bytes()
	if err != nil {
		return fmt.Errorf("serialize %s: %w", filepath.ToSlash(path), err)
	}
	perm := os.FileMode(0o600)
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}
	// Rename would replace a symlink (dotfile managers link configs) with a
	// regular file, so write to the link's target. A link whose target does not
	// exist is refused: there is no target to write.
	target := path
	resolved, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		target = resolved
	case errors.Is(err, fs.ErrNotExist):
		if info, lerr := os.Lstat(path); lerr == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink to a file that does not exist", filepath.ToSlash(path))
		}
	default:
		return fmt.Errorf("resolve %s: %w", filepath.ToSlash(path), err)
	}
	if err := atomicfile.Write(target, data, perm, atomicfile.MkDir(0o755)); err != nil {
		return fmt.Errorf("write %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

// stripBOM removes a leading UTF-8 byte order mark, reporting whether there was one.
func stripBOM(data []byte) ([]byte, bool) {
	if bytes.HasPrefix(data, utf8BOM) {
		return data[len(utf8BOM):], true
	}
	return data, false
}

// lineEnding returns CRLF when data uses it, and LF otherwise.
func lineEnding(data []byte) string {
	if bytes.Contains(data, []byte("\r\n")) {
		return "\r\n"
	}
	return "\n"
}

// withBOM puts the byte order mark back in front of data when the file had one.
func withBOM(data []byte, bom bool) []byte {
	if !bom {
		return data
	}
	return append(append([]byte{}, utf8BOM...), data...)
}

// builderKeysOf returns the keys of entry that the agent builders write.
func builderKeysOf(entry map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range builderKeys {
		if v, ok := entry[key]; ok {
			out[key] = v
		}
	}
	return out
}

// keepUserKeys returns current with its builder keys set to those of want. The
// other keys of current are kept.
func keepUserKeys(current, want map[string]any) map[string]any {
	out := maps.Clone(current)
	for _, key := range builderKeys {
		if v, ok := want[key]; ok {
			out[key] = v
		} else {
			delete(out, key)
		}
	}
	return out
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
