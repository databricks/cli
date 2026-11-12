package mcp

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
)

// Drift kinds reported by CheckEntries.
const (
	DriftProfileMissing DriftKind = "profile_missing"
	DriftHostChanged    DriftKind = "host_changed"
)

// Profile lookups that do not yield a usable profile.
var (
	// errProfileMissing means the config file has no profile of the name, or does not exist.
	errProfileMissing = errors.New("profile missing")
	// errProfileNoHost means the config file has a section of the name whose host is missing or empty.
	errProfileNoHost = errors.New("profile has no host")
)

// State is the MCP section of the aitools install state. It lives in the same file as
// the skills state, so writers that save the whole file keep it.
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

// LoadState reads the MCP section of the aitools state file. A missing file or section
// yields an empty state.
func LoadState(ctx context.Context) (*State, error) {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return nil, err
	}
	mcp, err := installer.LoadMCPState(dir)
	if err != nil {
		return nil, err
	}
	if mcp == nil {
		return &State{}, nil
	}
	return mcp, nil
}

// SaveState replaces the MCP section of the aitools state file. The skills and plugins records
// are kept as they are on disk, including those written since the caller's LoadState.
func SaveState(ctx context.Context, state *State) error {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}
	return installer.SaveMCPState(dir, state)
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
	case errors.Is(err, errProfileMissing) || errors.Is(err, errProfileNoHost):
		check.Drift = &Drift{Kind: DriftProfileMissing, Entry: entry, NoHost: errors.Is(err, errProfileNoHost)}
	case err != nil:
		check.Err = err
	default:
		check.Drift = driftOf(entry, p)
	}
	return check
}

// configFiles holds the config files that one lookup has read, by resolved path, so that each file is
// parsed once. A nil file is one that does not exist.
type configFiles map[string]*config.File

// profile returns the profile named name from the config file at path. It returns errProfileMissing
// when the file does not exist or has no section of that name, and errProfileNoHost when the section
// has no host or an empty one.
func (c configFiles) profile(path, name string) (profile.Profile, error) {
	file, ok := c[path]
	if !ok {
		loaded, err := config.LoadFile(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			loaded = nil
		case err != nil:
			return profile.Profile{}, fmt.Errorf("failed to read Databricks profiles: %w", err)
		}
		file = loaded
		c[path] = file
	}
	if file == nil {
		return profile.Profile{}, errProfileMissing
	}
	section, err := file.GetSection(name)
	if err != nil {
		return profile.Profile{}, errProfileMissing
	}
	keys := section.KeysHash()
	if keys["host"] == "" {
		return profile.Profile{}, errProfileNoHost
	}
	return profile.Profile{Name: name, Host: keys["host"], WorkspaceID: keys["workspace_id"]}, nil
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
