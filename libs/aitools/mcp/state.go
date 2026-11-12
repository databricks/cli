package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

// configFileEnv names the variable that selects the Databricks config file.
const configFileEnv = "DATABRICKS_CONFIG_FILE"

// Drift kinds reported by Reconcile.
const (
	DriftProfileMissing DriftKind = "profile_missing"
	DriftHostChanged    DriftKind = "host_changed"
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
	// CurrentHost is the profile's host when Kind is DriftHostChanged.
	CurrentHost string
	// CurrentWorkspaceID is the profile's workspace_id when Kind is DriftHostChanged,
	// or "" when the profile names no workspace.
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

// SaveState replaces the MCP section of the aitools state file. The file is re-read first,
// so the skills and plugins records written since the caller's LoadState are kept.
func SaveState(ctx context.Context, state *State) error {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}
	st, err := installer.LoadState(dir)
	if err != nil {
		return err
	}
	if st == nil {
		st = &installer.InstallState{SchemaVersion: installer.CurrentSchemaVersion}
	}
	st.MCP = state
	return installer.SaveState(dir, st)
}

// Reconcile reports entries whose profile is gone, or whose profile now names another host,
// or another workspace when both the entry and the profile name one. Each entry is checked
// against the config file it is pinned to; an entry with no pinned file is checked against
// the current one.
func Reconcile(ctx context.Context, profiler profile.Profiler, state *State) ([]Drift, error) {
	var drifts []Drift
	for _, entry := range state.Servers {
		entryCtx := ctx
		if entry.ConfigFile != "" {
			entryCtx = env.Set(ctx, configFileEnv, entry.ConfigFile)
		}
		profiles, err := profiler.LoadProfiles(entryCtx, profile.WithName(entry.Profile))
		if errors.Is(err, profile.ErrNoConfiguration) {
			profiles = nil
		} else if err != nil {
			return nil, fmt.Errorf("failed to read Databricks profiles: %w", err)
		}
		if len(profiles) == 0 {
			drifts = append(drifts, Drift{Kind: DriftProfileMissing, Entry: entry})
			continue
		}
		p := profiles[0]
		if hostChanged(entry, p) || workspaceChanged(entry, p) {
			drifts = append(drifts, Drift{
				Kind:               DriftHostChanged,
				Entry:              entry,
				CurrentHost:        p.Host,
				CurrentWorkspaceID: namedWorkspaceID(p.WorkspaceID),
			})
		}
	}
	return drifts, nil
}

// hostChanged reports whether the profile names another host than the one recorded.
func hostChanged(entry Entry, p profile.Profile) bool {
	return NormalizeHost(entry.Host) != NormalizeHost(p.Host)
}

// workspaceChanged reports whether the entry and the profile each name a workspace and the
// two differ. A profile without a workspace has nothing to compare.
func workspaceChanged(entry Entry, p profile.Profile) bool {
	recorded := namedWorkspaceID(entry.WorkspaceID)
	current := namedWorkspaceID(p.WorkspaceID)
	return recorded != "" && current != "" && recorded != current
}

// namedWorkspaceID returns the workspace ID a value names, or "" when it names none. The
// "none" sentinel that the CLI writes to skip workspace selection names none.
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
