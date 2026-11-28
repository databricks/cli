// Package agentconfig reads and writes the user-scope MCP server entries of
// coding agents. The direct writer changes only the bytes of the one entry it is
// asked to change, writes the file back atomically, and keeps comments and layout
// elsewhere in the file. An add that the agent's own CLI makes (see cli.go)
// leaves the file in the layout that the CLI writes.
package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"time"

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
	// this CLI. Upsert needs force to replace it, and RemoveAt never deletes it.
	ErrNotOwned = errors.New("entry was not written by databricks")
	// ErrConfigInvalid means an agent config file exists but cannot be parsed.
	// Such a file is never written to.
	ErrConfigInvalid = errors.New("agent config file cannot be parsed")
	// ErrReadOnly means an agent config file exists and the user cannot write it,
	// so the write is refused. The error is a *ReadOnlyError.
	ErrReadOnly = errors.New("agent config file is read-only")
)

var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// document is an agent config file in memory.
type document interface {
	// entry returns the server entry name under root, and whether it exists.
	entry(root []string, name string) (map[string]any, bool, error)
	// entries returns the object-valued members under root, keyed by name.
	entries(root []string) (map[string]map[string]any, error)
	// put writes entry as server name under root, creating missing levels and
	// replacing an existing server. A nil entry removes the server, and removing
	// an absent one is a no-op. With keys, an existing server changes only those
	// keys, and its other keys keep their bytes where the format allows it.
	put(root []string, name string, entry map[string]any, keys []string) error
	// bytes serializes the document.
	bytes() ([]byte, error)
}

// launchKeys are the entry keys that describe how a server starts. An agent's
// builder writes only some of them (see ownedKeys).
var launchKeys = []string{"command", "args", "env", "environment", "type"}

// ownedKeys returns the launch keys that the agent's builder writes. Upsert
// updates these keys and keeps every other key of the entry. The probe passes a
// variable, since a builder writes its env key only when it has variables.
func ownedKeys(spec *mcpconfig.Spec) []string {
	probe := spec.Entry("", nil, map[string]string{mcpconfig.ConfigFileEnv: ""})
	var keys []string
	for _, key := range launchKeys {
		if _, ok := probe[key]; ok {
			keys = append(keys, key)
		}
	}
	return keys
}

// Upsert writes a stdio server name for agent a, which runs cmd with args and
// env. The only variable env may set is mcpconfig.ConfigFileEnv, since other variables are the
// user's and no later write could remove them. An entry this CLI wrote gets its
// launch keys updated, and its other keys are kept. A foreign entry is replaced
// only when force is set. The file is left alone when the entry already matches.
// A new entry goes through the agent's own CLI when that CLI is on PATH (see
// addViaCLI), and the file then has the CLI's layout. cmd must be the databricks
// binary, or the entry could never be recognized as ours. It returns the config
// file path.
func Upsert(ctx context.Context, a *agents.Agent, name, cmd string, args []string, env map[string]string, force bool) (string, Status, error) {
	if err := checkName(name); err != nil {
		return "", "", err
	}
	for _, k := range slices.Sorted(maps.Keys(env)) {
		if k != mcpconfig.ConfigFileEnv {
			return "", "", fmt.Errorf("server %q: env may only set %s, got %q", name, mcpconfig.ConfigFileEnv, k)
		}
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
	path, err := configPath(ctx, spec)
	if err != nil {
		return "", "", err
	}
	// A new entry goes through the agent's own CLI, which takes the agent's own lock on the file, so this check runs, and
	// the CLI is started, without that lock. The direct writer below takes it, and reads the file again under it.
	doc, err := load(spec, path)
	if err != nil {
		return path, "", err
	}
	if err := checkLegacy(path, spec, doc); err != nil {
		return path, "", err
	}
	if _, found, err := doc.entry(rootPath(spec), name); err != nil {
		return path, "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	} else if !found {
		// The agent's CLI writes the file itself, so a read-only file is refused before it runs too.
		if err := checkWritable(path); err != nil {
			return path, "", err
		}
		done, cliErr := addViaCLI(ctx, a, spec, path, name, cmd, args, env, ours, force)
		if cliErr != nil {
			return path, "", cliErr
		}
		if done {
			return path, StatusAdded, nil
		}
	}
	status, err := upsertDirect(spec, path, name, want, ours, force)
	return path, status, err
}

// upsertDirect writes server name in the config file at path with the direct writer, under the agent's lock (see
// lockFor): it adds want when there is no entry, replaces an entry this CLI did not write only when force is set, and
// otherwise updates the launch keys of this CLI's entry to ours.
func upsertDirect(spec *mcpconfig.Spec, path, name string, want, ours map[string]any, force bool) (Status, error) {
	release, err := lockFor(spec, path)
	if err != nil {
		return "", err
	}
	defer release()
	doc, err := load(spec, path)
	if err != nil {
		return "", err
	}
	if err := checkLegacy(path, spec, doc); err != nil {
		return "", err
	}
	root := rootPath(spec)
	current, found, err := doc.entry(root, name)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}

	keys := ownedKeys(spec)
	var status Status
	switch {
	case !found:
		status = StatusAdded
		err = doc.put(root, name, want, nil)
	case !mcpconfig.IsDatabricksEntry(current) && !force:
		return "", fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, ErrNotOwned)
	case !mcpconfig.IsDatabricksEntry(current):
		status = StatusUpdated
		err = doc.put(root, name, want, nil)
	default:
		merged, changed, mergeErr := mergeLaunch(spec, current, ours)
		if mergeErr != nil {
			return "", fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, mergeErr)
		}
		if !changed {
			return StatusUnchanged, nil
		}
		status = StatusUpdated
		err = doc.put(root, name, merged, keys)
	}
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if err := checkWritable(path); err != nil {
		return "", err
	}
	if err := save(path, doc); err != nil {
		return "", err
	}
	return status, nil
}

// Changed reports whether current, an entry this CLI owns for agent a, differs from what Upsert writes now for a server
// that runs cmd with args and env. Only the launch keys are compared, as Upsert compares them.
func Changed(a *agents.Agent, current map[string]any, cmd string, args []string, env map[string]string) (bool, error) {
	spec, err := specOf(a)
	if err != nil {
		return false, err
	}
	ours, err := normalize(spec.Entry(cmd, args, env))
	if err != nil {
		return false, err
	}
	_, changed, err := mergeLaunch(spec, current, ours)
	return changed, err
}

// RemoveAt deletes server name from the config file at path, and refuses an entry this CLI did not write. It reports
// false, without writing, when there is no such entry. The file need not be the one the spec names now: a record names
// the file an entry was written to, and the agent's config may have moved since. A missing file has no entry. The direct
// writer removes the entry, so that the check of who wrote it and the removal are one edit, under the agent's lock
// where the agent has one: an agent's CLI removes whatever entry holds the name when it runs.
func RemoveAt(a *agents.Agent, path, name string) (bool, error) {
	if err := checkName(name); err != nil {
		return false, err
	}
	spec, err := specOf(a)
	if err != nil {
		return false, err
	}
	return removeDirect(a, spec, path, name)
}

// putDirect writes entry as server name in the config file at path with the direct writer, under the agent's lock (see
// lockFor). It reads the file again, since the agent's CLI may have changed it. It runs after the agent's CLI did, and
// replaces an entry of ours that the CLI wrote in another shape. An entry this CLI did not write was put there by
// someone else while the agent's CLI ran, and is replaced only when force is set, as upsertDirect replaces it.
func putDirect(spec *mcpconfig.Spec, path, name string, entry map[string]any, force bool) error {
	release, err := lockFor(spec, path)
	if err != nil {
		return err
	}
	defer release()
	doc, err := load(spec, path)
	if err != nil {
		return err
	}
	root := rootPath(spec)
	current, found, err := doc.entry(root, name)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if found && !mcpconfig.IsDatabricksEntry(current) && !force {
		return fmt.Errorf("%s: server %q: %w", filepath.ToSlash(path), name, ErrNotOwned)
	}
	if err := doc.put(root, name, entry, nil); err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return save(path, doc)
}

// removeDirect deletes server name from the config file at path, with the direct writer, under the agent's lock (see
// lockFor). It reads the file again, since the agent's CLI may have changed it, and it refuses an entry this CLI did not
// write.
func removeDirect(a *agents.Agent, spec *mcpconfig.Spec, path, name string) (bool, error) {
	release, err := lockFor(spec, path)
	if err != nil {
		return false, err
	}
	defer release()
	doc, err := load(spec, path)
	if err != nil {
		return false, err
	}
	if err := checkLegacy(path, spec, doc); err != nil {
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
	if err := doc.put(root, name, nil, nil); err != nil {
		return false, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	if err := checkWritable(path); err != nil {
		return false, err
	}
	if err := save(path, doc); err != nil {
		return false, err
	}
	return true, nil
}

// ReadAt returns every server entry in the config file at path, keyed by name. A missing file has no entries. The file
// need not be the one the spec names now: a record names the file an entry was written to, and the agent's config may
// have moved since.
func ReadAt(a *agents.Agent, path string) (map[string]map[string]any, error) {
	spec, err := specOf(a)
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

// Path returns the absolute path of agent a's MCP config file, where Upsert writes. It does not read the file.
func Path(ctx context.Context, a *agents.Agent) (string, error) {
	spec, err := specOf(a)
	if err != nil {
		return "", err
	}
	return configPath(ctx, spec)
}

// configPath returns the absolute path of the config file of spec. A config dir set to a relative path is made absolute
// here, not when the path is used, so that a record of the path works from any directory. The path is not cleaned, as
// mcpconfig.AbsoluteAsWritten keeps it, so that a ".." after a symlink still resolves through the link.
func configPath(ctx context.Context, spec *mcpconfig.Spec) (string, error) {
	path, err := spec.Path(ctx)
	if err != nil {
		return "", err
	}
	return mcpconfig.AbsoluteAsWritten(path)
}

// checkName refuses a name that no agent would accept. Upsert and RemoveAt both
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

// checkLegacy refuses a config that still holds servers in an older layout, next
// to the table that Root names. A write would add that table beside them, which
// the older layout reads as a server. An object with a type or command key is
// taken as a server. Other objects in the table, such as a timeout, are not.
func checkLegacy(path string, spec *mcpconfig.Spec, doc document) error {
	if !spec.Legacy {
		return nil
	}
	root := rootPath(spec)
	parent, own := root[:len(root)-1], root[len(root)-1]
	members, err := doc.entries(parent)
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	var legacy []string
	for name, member := range members {
		_, typed := member["type"]
		_, command := member["command"]
		if name != own && (typed || command) {
			legacy = append(legacy, name)
		}
	}
	if len(legacy) == 0 {
		return nil
	}
	slices.Sort(legacy)
	return fmt.Errorf("%s: %q holds servers in an older layout (%s); move them under %q and try again",
		filepath.ToSlash(path), strings.Join(parent, "."), strings.Join(legacy, ", "), spec.Root)
}

// Claude Code takes a lock while it saves its config, so a write that reads and
// then renames without the lock can lose Claude's update, or be lost to it. The
// lock is the directory <path>.lock: Claude Code makes it with mkdir and
// removes it with rmdir, and treats a lock whose mtime is over 10 s old as left
// behind by a process that died.
const (
	// lockStaleAfter is the age after which a lock is taken as abandoned.
	lockStaleAfter = 10 * time.Second
	// lockRetry is the pause between attempts to take a lock that is held.
	lockRetry = time.Millisecond
)

// lockTimeout bounds the wait for a lock held by a live process. It is longer
// than lockStaleAfter, so a lock left by a process that died is taken over on
// the first attempt rather than failing it. A live holder keeps the lock for a
// few milliseconds, since Claude Code only rereads and rewrites the file under it.
var lockTimeout = 15 * time.Second

// lockFor takes the lock on the config file at path when the agent of spec locks its config (see mcpconfig.Spec.Lock),
// and returns the function that releases it. It takes no lock otherwise. Only the direct writer takes it: the agent's own
// CLI takes the agent's lock itself, so the lock is never held while that CLI runs.
func lockFor(spec *mcpconfig.Spec, path string) (func(), error) {
	if !spec.Lock {
		return func() {}, nil
	}
	return lockConfig(path)
}

// lockConfig takes the lock on the config file at path and returns the function
// that releases it. The lock is held for a few milliseconds at most, so it is
// not refreshed while held; a holder that dies leaves it to go stale.
func lockConfig(path string) (func(), error) {
	lock, err := configLock(path)
	if err != nil {
		return nil, err
	}
	// With no config directory, Claude Code has never saved this file, so there is no lock to share. The lock is not
	// taken then, so that a remove does not create the directory.
	if _, err := os.Stat(filepath.Dir(lock)); errors.Is(err, fs.ErrNotExist) {
		return func() {}, nil
	}
	deadline := time.Now().Add(lockTimeout)
	for {
		err := os.Mkdir(lock, 0o700)
		if err == nil {
			return func() { _ = os.Remove(lock) }, nil
		}
		if lockBusy(err) {
			// Windows refuses to create a directory that another process is still removing, with access denied.
			// The lock is then being released, so it is tried again; a real permission problem fails at the deadline.
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("lock %s: %w", filepath.ToSlash(lock), err)
			}
			time.Sleep(lockRetry)
			continue
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("lock %s: %w", filepath.ToSlash(lock), err)
		}
		if info, statErr := os.Stat(lock); statErr == nil && time.Since(info.ModTime()) > lockStaleAfter {
			if err := os.Remove(lock); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, fmt.Errorf("remove stale lock %s: %w", filepath.ToSlash(lock), err)
			}
			continue
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("%s is locked by another process, most likely Claude Code saving its config; try again", filepath.ToSlash(lock))
		}
		time.Sleep(lockRetry)
	}
}

// lockBusy reports whether a failed mkdir of a lock means the lock is being released: on Windows, creating a directory
// that is pending deletion fails with access denied.
func lockBusy(err error) bool {
	return runtime.GOOS == "windows" && errors.Is(err, fs.ErrPermission)
}

// configLock returns the lock path for the config file at path: its absolute
// path with a .lock suffix. Claude Code names its lock after the config path as
// given, without resolving symlinks, so a symlinked config is locked next to the
// link, not next to its target. A symlink in a parent directory needs no care,
// since both names reach the same directory.
func configLock(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return abs + ".lock", nil
}

// openInUseTimeout bounds the retries of an open that fails because the file is in use (see openConfig).
const openInUseTimeout = 2 * time.Second

// openConfig opens the file at path with flag. An open without the agent's lock can run while the agent renames a new
// version over the file, as Claude Code does on every save; on Windows, the open then fails with a sharing violation
// for as long as the rename takes, so it is tried again.
func openConfig(path string, flag int) (*os.File, error) {
	deadline := time.Now().Add(openInUseTimeout)
	for {
		f, err := os.OpenFile(path, flag, 0)
		if err == nil || !fileInUse(err) || time.Now().After(deadline) {
			return f, err
		}
		time.Sleep(lockRetry)
	}
}

// readConfig reads the file at path, opening it with openConfig.
func readConfig(path string) ([]byte, error) {
	f, err := openConfig(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

// load reads and parses the config at path. A missing file is an empty document.
func load(spec *mcpconfig.Spec, path string) (document, error) {
	data, err := readConfig(path)
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

// ReadOnlyError is a write refused because the config file at Path is read-only.
// It matches ErrReadOnly, and wraps the error that opening the file returned.
type ReadOnlyError struct {
	Path string
	Err  error
}

func (e *ReadOnlyError) Error() string {
	return fmt.Sprintf("%s is read-only; make it writable: %v", e.Path, e.Err)
}

func (e *ReadOnlyError) Unwrap() error { return e.Err }

func (e *ReadOnlyError) Is(target error) bool { return target == ErrReadOnly }

// checkWritable refuses an existing config file that the user cannot write. The
// save replaces the file by rename, which needs write permission only on the
// directory, so a read-only file would be replaced anyway. A file the user made
// read-only is one they mean to keep as it is. A missing file is not refused.
func checkWritable(path string) error {
	f, err := openConfig(path, os.O_WRONLY)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil
	case errors.Is(err, fs.ErrPermission):
		return &ReadOnlyError{Path: filepath.ToSlash(path), Err: err}
	case err != nil:
		return fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
	}
	return f.Close()
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
		if isSymlink(path) {
			return fmt.Errorf("%s is a symlink to a file that does not exist: %w", filepath.ToSlash(path), err)
		}
	default:
		return fmt.Errorf("resolve %s: %w", filepath.ToSlash(path), err)
	}
	return atomicfile.Write(target, data, perm, atomicfile.MkDir(0o700))
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

// launchKeysOf returns the keys of entry that are listed in keys.
func launchKeysOf(entry map[string]any, keys []string) map[string]any {
	out := map[string]any{}
	for _, key := range keys {
		if v, ok := entry[key]; ok {
			out[key] = v
		}
	}
	return out
}

// mergeLaunch returns current with the launch keys of want written over it, and whether that changes the launch keys of
// current. The other keys of current are kept, so they are not compared.
func mergeLaunch(spec *mcpconfig.Spec, current, want map[string]any) (map[string]any, bool, error) {
	keys := ownedKeys(spec)
	merged, err := keepUserKeys(current, want, keys)
	if err != nil {
		return nil, false, err
	}
	same, err := sameEntry(launchOf(current, keys), launchOf(merged, keys))
	if err != nil {
		return nil, false, err
	}
	return merged, !same, nil
}

// keepUserKeys returns current with the keys listed in keys set to those of want,
// and every other key of current kept. A listed key that want lacks is removed.
// The env tables are merged by mergeEnv.
func keepUserKeys(current, want map[string]any, keys []string) (map[string]any, error) {
	out := maps.Clone(current)
	for _, key := range keys {
		value, ok := want[key]
		if slices.Contains(mcpconfig.EnvKeys, key) {
			var err error
			value, ok, err = mergeEnv(current[key], value, ok)
			if err != nil {
				return nil, err
			}
		}
		if ok {
			out[key] = value
		} else {
			delete(out, key)
		}
	}
	return out, nil
}

// mergeEnv returns the env table after a write. It holds the variables of want,
// and the variables of have other than the pinned config file variable. It
// reports false when the table is absent: want lacks it and no variable is left.
func mergeEnv(have, want any, inWant bool) (any, bool, error) {
	haveEnv, err := envTable(have)
	if err != nil {
		return nil, false, err
	}
	wantEnv, _ := want.(map[string]any)
	out := map[string]any{}
	for k, v := range haveEnv {
		if k != mcpconfig.ConfigFileEnv {
			out[k] = v
		}
	}
	maps.Copy(out, wantEnv)
	if len(out) == 0 && !inWant {
		return nil, false, nil
	}
	return out, true, nil
}

// envTable returns the variables of an env table, or nil when v is absent.
func envTable(v any) (map[string]any, error) {
	if v == nil {
		return nil, nil
	}
	table, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("environment variables are not a table: %w", ErrConfigInvalid)
	}
	return table, nil
}

// checkEdit verifies an edit to a document, in memory and before it is saved: the
// server is entry (or gone when entry is nil), and nothing else in the document
// changed. sameRest compares the document without the server, and sameEntry
// compares the server with entry; both compare in the terms of the format. A
// removal drops the containers it leaves empty, up to the root, as withoutServer
// does, and the check accepts that drop and nothing else.
func checkEdit(before, after map[string]any, root []string, name string, entry map[string]any, sameRest, sameEntry func(a, b map[string]any) (bool, error)) error {
	eq, err := sameRest(withoutServer(before, root, name), withoutServer(after, root, name))
	if err != nil {
		return err
	}
	if !eq {
		return errors.New("the edit changed other settings")
	}
	got, found := serverIn(after, root, name)
	if entry == nil {
		if found {
			return errors.New("the edit did not remove the server")
		}
		return nil
	}
	if !found {
		return errors.New("the edit did not add the server")
	}
	eq, err = sameEntry(got, entry)
	if err != nil {
		return err
	}
	if !eq {
		return errors.New("the edit wrote a different server")
	}
	return nil
}

// serverIn returns the table of server name under root, and whether name is there
// at all. A name that holds something other than a table is found, with a nil table.
func serverIn(doc map[string]any, root []string, name string) (map[string]any, bool) {
	for _, key := range root {
		child, ok := doc[key].(map[string]any)
		if !ok {
			return nil, false
		}
		doc = child
	}
	v, ok := doc[name]
	if !ok {
		return nil, false
	}
	entry, _ := v.(map[string]any)
	return entry, true
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
