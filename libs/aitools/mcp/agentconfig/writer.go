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
	// set stores entry as name under root, replacing an existing entry and
	// creating missing levels.
	set(root []string, name string, entry map[string]any) error
	// update changes the launch keys of the existing entry name under root to
	// those of entry. Other keys of the entry keep their bytes where the format
	// allows it.
	update(root []string, name string, entry map[string]any) error
	// remove deletes name under root. It is a no-op when name is absent.
	remove(root []string, name string) error
	// bytes serializes the document.
	bytes() ([]byte, error)
}

// launchKeys are the entry keys that describe how the server starts. Upsert sets
// them on an update, and an entry whose launch keys match is not rewritten. Any
// other key is the user's after the entry is first added: a default that the
// builder writes (Copilot's tools list) is not written again, and a key the user
// added (disabled, or a tools table) is kept.
var launchKeys = []string{"command", "args", "env", "environment", "type"}

// Upsert writes a stdio server name for agent a, which runs cmd with args and
// env. An entry this CLI wrote gets its launch keys updated, and the other keys
// are kept. A foreign entry is replaced only when force is set. The file is left
// alone when the entry already matches. cmd must be the databricks binary, or
// the entry could never be recognized as ours. It returns the config file path.
func Upsert(ctx context.Context, a *agents.Agent, name, cmd string, args []string, env map[string]string, force bool) (string, Status, error) {
	if err := checkName(name); err != nil {
		return "", "", err
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
	if !mcpconfig.IsDatabricksBinary(cmd) {
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

	var status Status
	switch {
	case !found:
		status = StatusAdded
		err = doc.set(root, name, want)
	case !mcpconfig.IsDatabricksEntry(current) && !force:
		return path, "", fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, ErrNotOwned)
	case !mcpconfig.IsDatabricksEntry(current):
		status = StatusUpdated
		err = doc.set(root, name, want)
	default:
		same, cmpErr := sameEntry(launchKeysOf(current), launchKeysOf(want))
		if cmpErr != nil {
			return path, "", cmpErr
		}
		if same {
			return path, StatusUnchanged, nil
		}
		status = StatusUpdated
		err = doc.update(root, name, keepUserKeys(current, want))
	}
	if err != nil {
		return path, "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if err := save(path, doc); err != nil {
		return path, "", err
	}
	return path, status, nil
}

// Remove deletes server name from agent a's config. It reports false, without
// writing, when there is no such entry, and refuses an entry this CLI did not
// write.
func Remove(ctx context.Context, a *agents.Agent, name string) (bool, error) {
	if err := checkName(name); err != nil {
		return false, err
	}
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

// checkName refuses a name that no agent would accept. Upsert and Remove both
// check it, so a name that cannot be written is never looked up either.
func checkName(name string) error {
	if !mcpconfig.ValidServerName(name) {
		return fmt.Errorf("server name %q must contain only letters, digits, '-' and '_'", name)
	}
	return nil
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
// 0600 and a new directory 0700, because agent configs can hold auth state.
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
	if err := atomicfile.Write(target, data, perm, atomicfile.MkDir(0o700)); err != nil {
		return fmt.Errorf("write %s: %w", filepath.ToSlash(path), err)
	}
	return nil
}

// stripBOM removes a leading UTF-8 byte order mark, reporting whether there was one.
func stripBOM(data []byte) ([]byte, bool) {
	rest, ok := bytes.CutPrefix(data, utf8BOM)
	return rest, ok
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

// launchKeysOf returns the launch keys of entry.
func launchKeysOf(entry map[string]any) map[string]any {
	out := map[string]any{}
	for _, key := range launchKeys {
		if v, ok := entry[key]; ok {
			out[key] = v
		}
	}
	return out
}

// keepUserKeys returns current with its launch keys set to those of want, and
// removes a launch key that want lacks. The other keys of current are kept, and
// a key that want has but current lacks is not added.
func keepUserKeys(current, want map[string]any) map[string]any {
	out := maps.Clone(current)
	for _, key := range launchKeys {
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
	if err := decodeJSON(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// decodeJSON decodes data into v. Numbers stay as their text, so a value read
// from a file is not rounded on the way.
func decodeJSON(data []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	return dec.Decode(v)
}
