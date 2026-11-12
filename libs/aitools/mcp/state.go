package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

// Drift kinds reported by Reconcile.
const (
	DriftProfileMissing DriftKind = "profile_missing"
	DriftHostChanged    DriftKind = "host_changed"
)

// Profile lookups that do not yield a usable profile.
var (
	// errProfileMissing means the config file has no profile of the name.
	errProfileMissing = errors.New("profile missing")
	// errProfileNoHost means the config file has a section of the name without a host.
	// LoadProfiles skips such sections, so the lookup reports them separately.
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
}

// LoadState reads the MCP section of the aitools state file. A missing file or section
// yields an empty state.
func LoadState(ctx context.Context) (*State, error) {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return nil, err
	}
	st, err := installer.LoadState(dir)
	if err != nil {
		return nil, err
	}
	if st == nil || st.MCP == nil {
		return &State{}, nil
	}
	return st.MCP, nil
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

// Reconcile reports entries whose profile is gone or has no host, and entries whose profile now
// names another host, or another workspace when both the entry and the profile name one. Each
// entry is checked against the config file it is pinned to; an entry with no pinned file is checked
// against the default file, ~/.databrickscfg, whatever the environment names.
//
// An entry that cannot be checked, such as one whose "~" path has no home directory to resolve, does
// not stop the others. Its error is joined into the returned error, and the drifts of the other
// entries are still returned.
func Reconcile(ctx context.Context, profiler profile.Profiler, state *State) ([]Drift, error) {
	var drifts []Drift
	var errs []error
	for _, entry := range state.Servers {
		drift, err := checkEntry(ctx, profiler, entry)
		if err != nil {
			errs = append(errs, fmt.Errorf("MCP server %s: %w", entry.Name, err))
			continue
		}
		if drift != nil {
			drifts = append(drifts, *drift)
		}
	}
	return drifts, errors.Join(errs...)
}

// checkEntry returns the drift of one entry, or nil when its profile still names the recorded host
// and workspace. The entry's profile is read from the config file the entry is pinned to.
func checkEntry(ctx context.Context, profiler profile.Profiler, entry Entry) (*Drift, error) {
	path, err := resolveConfigFile(ctx, entry.ConfigFile)
	if err != nil {
		return nil, err
	}
	p, err := loadProfile(env.Set(ctx, mcpconfig.ConfigFileEnv, path), profiler, entry.Profile)
	if errors.Is(err, errProfileMissing) || errors.Is(err, errProfileNoHost) {
		return &Drift{Kind: DriftProfileMissing, Entry: entry}, nil
	}
	if err != nil {
		return nil, err
	}
	return driftOf(entry, p), nil
}

// loadProfile returns the profile named name from the config file in ctx. It returns errProfileMissing
// when the file has no such profile, and errProfileNoHost when it has a section of that name without a
// host.
func loadProfile(ctx context.Context, profiler profile.Profiler, name string) (profile.Profile, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(name))
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return profile.Profile{}, fmt.Errorf("failed to read Databricks profiles: %w", err)
	}
	if len(profiles) > 0 {
		return profiles[0], nil
	}
	file, err := profile.FileProfilerImpl{}.Get(ctx)
	if errors.Is(err, profile.ErrNoConfiguration) {
		return profile.Profile{}, errProfileMissing
	}
	if err != nil {
		return profile.Profile{}, fmt.Errorf("failed to read Databricks profiles: %w", err)
	}
	if file.HasSection(name) {
		return profile.Profile{}, errProfileNoHost
	}
	return profile.Profile{}, errProfileMissing
}

// driftOf returns the drift of an entry against the profile it names, or nil when the profile still
// names the recorded host and workspace.
func driftOf(entry Entry, p profile.Profile) *Drift {
	host, workspaceID := profileTarget(p)
	if NormalizeHost(entry.Host) == NormalizeHost(host) && !workspaceChanged(entry.WorkspaceID, workspaceID) {
		return nil
	}
	return &Drift{Kind: DriftHostChanged, Entry: entry, CurrentHost: p.Host, CurrentWorkspaceID: workspaceID}
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
