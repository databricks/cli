package mcp

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

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

// SaveState replaces the MCP section of the aitools state file. The skills and plugins records
// are kept as they are on disk, including those written since the caller's LoadState.
func SaveState(ctx context.Context, state *State) error {
	dir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return err
	}
	return installer.SaveMCPState(dir, state)
}

// Reconcile reports entries whose profile is gone, or whose profile now names another host,
// or another workspace when both the entry and the profile name one. Each entry is checked
// against the config file it is pinned to; an entry with no pinned file is checked against
// the default file, ~/.databrickscfg, whatever the environment names.
func Reconcile(ctx context.Context, profiler profile.Profiler, state *State) ([]Drift, error) {
	var drifts []Drift
	for _, entry := range state.Servers {
		path, err := resolveConfigFile(ctx, entry.ConfigFile)
		if err != nil {
			return nil, err
		}
		entryCtx := env.Set(ctx, mcpconfig.ConfigFileEnv, path)
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
				CurrentWorkspaceID: profileWorkspaceID(p),
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
	current := profileWorkspaceID(p)
	return recorded != "" && current != "" && recorded != current
}

// profileWorkspaceID returns the workspace a profile names: its workspace_id when it has one,
// otherwise the workspace in its host's query, which the SDK resolves the same way. It returns
// "" when the profile names none.
func profileWorkspaceID(p profile.Profile) string {
	if p.WorkspaceID != "" {
		return namedWorkspaceID(p.WorkspaceID)
	}
	return namedWorkspaceID(auth.ExtractHostQueryParams(p.Host).WorkspaceID)
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

// resolveConfigFile returns the absolute path of the config file that name refers to. An empty
// name is the default file, ~/.databrickscfg. A leading "~" is the home directory, as the
// profile loader reads it. Only an absolute name needs no home directory.
func resolveConfigFile(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = "~/.databrickscfg"
	}
	if strings.HasPrefix(name, "~") {
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		name = filepath.Join(home, name[1:])
	}
	return filepath.Abs(name)
}
