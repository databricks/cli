package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"gopkg.in/ini.v1"
)

// Drift kinds reported by CheckEntries.
const (
	DriftProfileMissing DriftKind = "profile_missing"
	DriftHostChanged    DriftKind = "host_changed"
)

// Profile lookups that do not yield a usable profile.
var (
	// ErrProfileMissing means the config file has no profile of the name, or does not exist.
	ErrProfileMissing = errors.New("profile missing")
	// errProfileNoHost means the config file has a section of the name whose host is missing or empty.
	errProfileNoHost = errors.New("profile has no host")
)

// State is the MCP install records. They live in a file of their own, ~/.databricks/aitools/mcp.json, so that a CLI
// that does not know them never writes it. v1.28.0 kept them as a section of the skills state file, which the skills
// installer of an older CLI writes back without the section; LoadState still reads that section when the file of
// their own does not exist, and the first SaveState moves the records.
type State = installer.MCPState

// Entry is one installed server: a service bound to one profile for one set of agents.
type Entry = installer.MCPEntry

// AgentRecord is where an entry was written for one agent.
type AgentRecord = installer.MCPAgentRecord

// DriftKind names how a state entry no longer matches the profiles file.
type DriftKind string

// Drift is one state entry that no longer matches the Databricks config file.
type Drift struct {
	Kind  DriftKind
	Entry Entry
	// CurrentHost is the profile's host as the config file writes it, when Kind is DriftHostChanged.
	CurrentHost string
	// CurrentWorkspaceID is the workspace the profile names when Kind is DriftHostChanged: its
	// workspace_id, else the workspace in its host query. It is "" when the profile names none.
	CurrentWorkspaceID string
	// NoHost is set when Kind is DriftProfileMissing because the profile's section has no host, not
	// because the section is absent.
	NoHost bool
}

// recordsFileName is the file of the MCP install records, in ~/.databricks/aitools.
const recordsFileName = "mcp.json"

// recordsSchemaVersion is the schema version of the records file that this CLI writes. A file with a newer version is
// refused: writing it back would drop what this CLI does not know.
const recordsSchemaVersion = 1

// recordsFile is the records file as it is on disk.
type recordsFile struct {
	SchemaVersion int     `json:"schema_version"`
	Servers       []Entry `json:"servers"`
}

// recordsPaths returns the path of the records file, and the directory of the skills state file that held the records
// in v1.28.0.
func recordsPaths(ctx context.Context) (path, skillsDir string, err error) {
	skillsDir, err = installer.GlobalSkillsDir(ctx)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(filepath.Dir(skillsDir), recordsFileName), skillsDir, nil
}

// readRecords reads the records file at path. It returns nil when the file does not exist.
func readRecords(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP records file: %w", err)
	}
	var file recordsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, &installer.StateError{Path: path, Err: fmt.Errorf("failed to parse MCP records file %s: %w", filepath.ToSlash(path), err)}
	}
	if file.SchemaVersion > recordsSchemaVersion {
		return nil, &installer.StateError{Path: path, Err: fmt.Errorf("%s was written by a newer Databricks CLI (schema version %d, this CLI reads %d); upgrade the CLI to manage MCP servers", filepath.ToSlash(path), file.SchemaVersion, recordsSchemaVersion)}
	}
	return &State{Servers: file.Servers}, nil
}

// LoadState reads the MCP install records. When their file does not exist, it reads the section of the skills state
// file that v1.28.0 wrote. No file and no section yield an empty state.
func LoadState(ctx context.Context) (*State, error) {
	path, skillsDir, err := recordsPaths(ctx)
	if err != nil {
		return nil, err
	}
	state, err := readRecords(path)
	if err != nil || state != nil {
		return state, err
	}
	legacy, err := installer.LoadMCPState(skillsDir)
	if err != nil {
		return nil, err
	}
	if legacy == nil {
		return &State{}, nil
	}
	return legacy, nil
}

// SaveState writes the MCP install records to their file, and removes the file when no record is left. It then removes
// the section that v1.28.0 wrote to the skills state file, so that those records are not read again once the file of
// their own is gone. A records file that a newer CLI wrote is not written.
func SaveState(ctx context.Context, state *State) error {
	path, skillsDir, err := recordsPaths(ctx)
	if err != nil {
		return err
	}
	if _, err := readRecords(path); err != nil {
		return err
	}
	if len(state.Servers) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("failed to remove MCP records file: %w", err)
		}
	} else {
		data, err := json.MarshalIndent(recordsFile{SchemaVersion: recordsSchemaVersion, Servers: state.Servers}, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal MCP records: %w", err)
		}
		if err := atomicfile.Write(path, append(data, '\n'), 0o600, atomicfile.MkDir(0o755)); err != nil {
			return err
		}
	}
	legacy, err := installer.LoadMCPState(skillsDir)
	if err != nil || legacy == nil {
		return err
	}
	return installer.SaveMCPState(skillsDir, nil)
}

// EntryCheck is the result of checking one state entry. Drift is nil when the entry's profile still names the recorded
// host and workspace. Err is set when the entry cannot be checked. Path is the config file the entry is checked against,
// resolved; it is "" when that file cannot be resolved, and Err says why.
type EntryCheck struct {
	Drift *Drift
	Err   error
	Path  string
}

// CheckEntries checks each entry of state against its profile and returns one result per entry, in state order. Each
// entry is checked against the config file it is pinned to; an entry with no pinned file is checked against the default
// file, ~/.databrickscfg, whatever the environment names. Each config file is read once for all the entries. An entry
// that cannot be checked, such as one whose "~" path has no home directory to resolve, does not stop the others.
func CheckEntries(ctx context.Context, state *State) []EntryCheck {
	files := configFiles{}
	checks := make([]EntryCheck, len(state.Servers))
	for i, entry := range state.Servers {
		checks[i] = checkEntry(ctx, files, entry)
	}
	return checks
}

// checkEntry checks one entry against the profile in the config file it is pinned to.
func checkEntry(ctx context.Context, files configFiles, entry Entry) EntryCheck {
	path, err := ResolveConfigFile(ctx, entry.ConfigFile)
	if err != nil {
		return EntryCheck{Err: err}
	}
	check := EntryCheck{Path: path}
	p, err := files.profile(path, entry.Profile)
	switch {
	case errors.Is(err, ErrProfileMissing) || errors.Is(err, errProfileNoHost):
		check.Drift = &Drift{Kind: DriftProfileMissing, Entry: entry, NoHost: errors.Is(err, errProfileNoHost)}
	case err != nil:
		check.Err = err
	default:
		check.Drift = driftOf(entry, p)
	}
	return check
}

// configFiles holds the config files that one lookup has read, by resolved path, so that each file is parsed once. A
// file that cannot be read keeps its error, which every profile of that file then reports.
type configFiles map[string]configFile

// configFile is one config file as a lookup read it: the parsed file, which is nil when the file does not exist, or the
// error that reading it gave.
type configFile struct {
	file *config.File
	err  error
}

// profile returns the profile named name from the config file at path. It returns ErrProfileMissing
// when the file does not exist or has no section of that name, and errProfileNoHost when the section
// has no host or an empty one.
func (c configFiles) profile(path, name string) (profile.Profile, error) {
	f, ok := c[path]
	if !ok {
		f = readConfigFile(path)
		c[path] = f
	}
	if f.err != nil {
		return profile.Profile{}, f.err
	}
	if f.file == nil {
		return profile.Profile{}, ErrProfileMissing
	}
	section, err := f.file.GetSection(name)
	if err != nil {
		return profile.Profile{}, ErrProfileMissing
	}
	keys := section.KeysHash()
	if len(keys) == 0 && name == ini.DefaultSection {
		// The INI parser always has a DEFAULT section, so an empty one means the file has no such profile.
		return profile.Profile{}, ErrProfileMissing
	}
	if keys["host"] == "" {
		return profile.Profile{}, errProfileNoHost
	}
	return profile.Profile{Name: name, Host: keys["host"], WorkspaceID: keys["workspace_id"]}, nil
}

// readConfigFile reads the config file at path. A file that does not exist is read as nil, with no error.
func readConfigFile(path string) configFile {
	file, err := config.LoadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return configFile{}
	case err != nil:
		return configFile{err: fmt.Errorf("failed to read Databricks profiles: %w", err)}
	}
	return configFile{file: file}
}

// driftOf returns the drift of an entry against the profile it names, or nil when the profile still
// names the recorded host and workspace.
func driftOf(entry Entry, p profile.Profile) *Drift {
	recordedHost, recordedWorkspaceID := recordedTarget(entry)
	host, workspaceID := profileTarget(p)
	if NormalizeHost(recordedHost) == NormalizeHost(host) && !workspaceChanged(recordedWorkspaceID, workspaceID) {
		return nil
	}
	return &Drift{Kind: DriftHostChanged, Entry: entry, CurrentHost: p.Host, CurrentWorkspaceID: workspaceID}
}

// recordedTarget returns the host and workspace ID that an entry recorded, resolved as the SDK resolves
// a profile's, so that a workspace named in the recorded host's query is compared too.
func recordedTarget(entry Entry) (host, workspaceID string) {
	return profileTarget(profile.Profile{Host: entry.Host, WorkspaceID: entry.WorkspaceID})
}

// profileTarget returns the host and workspace ID that the SDK resolves for a profile. It runs the
// SDK's host handling only, so it makes no network request: the SDK keeps a workspace_id the profile
// sets, and otherwise takes the workspace from the host query. The host has its query removed. The
// workspace is "" when the profile names none, or names "none".
func profileTarget(p profile.Profile) (host, workspaceID string) {
	c := &config.Config{Host: p.Host, WorkspaceID: p.WorkspaceID}
	host = c.CanonicalHostName()
	return host, namedWorkspaceID(c.WorkspaceID)
}

// workspaceChanged reports whether the recorded and current workspace IDs both name a workspace and
// the two differ. A workspace that is not named is not compared.
func workspaceChanged(recorded, current string) bool {
	recorded = namedWorkspaceID(recorded)
	return recorded != "" && current != "" && recorded != current
}

// namedWorkspaceID returns the workspace ID a value names, or "" when it names none. The "none"
// sentinel that the CLI writes to skip workspace selection names none.
func namedWorkspaceID(id string) string {
	if id == profile.WorkspaceIDNone {
		return ""
	}
	return id
}

// NormalizeHost returns the form in which two hosts compare equal for drift and binding
// checks. It applies the SDK's host canonicalization (scheme and trailing slash), then
// lowercases the result, because hostnames are case-insensitive.
func NormalizeHost(host string) string {
	return strings.ToLower((&config.Config{Host: host}).CanonicalHostName())
}
