package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
)

const (
	stateSchemaVersion = 1
	stateFileName      = "state.json"
)

// Drift kinds reported by Reconcile.
const (
	DriftProfileMissing DriftKind = "profile_missing"
	DriftHostChanged    DriftKind = "host_changed"
)

// State is the record of MCP servers this CLI wrote. Agent config files stay the
// source of truth for existence; state records ownership and the binding.
type State struct {
	SchemaVersion int     `json:"schema_version"`
	Servers       []Entry `json:"servers"`
}

// Entry is one installed server: a service bound to one profile for one set of agents.
type Entry struct {
	Name           string                 `json:"name"`
	Service        string                 `json:"service"`
	Profile        string                 `json:"profile"`
	Host           string                 `json:"host"`
	WorkspaceID    string                 `json:"workspace_id"`
	SignInRequired bool                   `json:"sign_in_required"`
	Command        string                 `json:"command"`
	ConfigFile     string                 `json:"config_file"`
	Agents         map[string]AgentRecord `json:"agents"`
	InstalledAt    time.Time              `json:"installed_at"`
	CLIVersion     string                 `json:"cli_version"`
}

// AgentRecord is where an entry was written for one agent.
type AgentRecord struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
}

// DriftKind names how a state entry no longer matches the profiles file.
type DriftKind string

// Drift is one state entry that no longer matches the Databricks config file.
type Drift struct {
	Kind  DriftKind
	Entry Entry
	// CurrentHost is the profile's host when Kind is DriftHostChanged.
	CurrentHost string
}

// LoadState reads the state file. A missing file yields an empty state.
func LoadState(ctx context.Context) (*State, error) {
	path, err := statePath(ctx)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &State{SchemaVersion: stateSchemaVersion}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse MCP state file %s: %w", path, err)
	}
	if err := migrateState(&state); err != nil {
		return nil, err
	}
	return &state, nil
}

// SaveState writes the state file atomically with mode 0600. It stamps the current schema version.
func SaveState(ctx context.Context, state *State) error {
	path, err := statePath(ctx)
	if err != nil {
		return err
	}
	state.SchemaVersion = stateSchemaVersion
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP state: %w", err)
	}
	data = append(data, '\n')
	return atomicfile.Write(path, data, 0o600, atomicfile.MkDir(0o755))
}

// migrateState brings a loaded state forward to stateSchemaVersion. It is forward-only:
// a file from a newer CLI is rejected instead of being rewritten.
func migrateState(state *State) error {
	if state.SchemaVersion > stateSchemaVersion {
		return fmt.Errorf("MCP state file has schema_version %d, but this CLI supports up to %d; upgrade the CLI", state.SchemaVersion, stateSchemaVersion)
	}
	state.SchemaVersion = stateSchemaVersion
	return nil
}

func statePath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "aitools", "mcp", stateFileName), nil
}

// Reconcile reports entries whose profile is gone or whose host differs from the
// recorded one. A missing Databricks config file means every entry's profile is missing.
func Reconcile(ctx context.Context, profiler profile.Profiler, state *State) ([]Drift, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if errors.Is(err, profile.ErrNoConfiguration) {
		profiles = nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to read Databricks profiles: %w", err)
	}

	var drifts []Drift
	for _, entry := range state.Servers {
		i := slices.IndexFunc(profiles, profile.WithName(entry.Profile))
		switch {
		case i < 0:
			drifts = append(drifts, Drift{Kind: DriftProfileMissing, Entry: entry})
		case NormalizeHost(profiles[i].Host) != NormalizeHost(entry.Host):
			drifts = append(drifts, Drift{Kind: DriftHostChanged, Entry: entry, CurrentHost: profiles[i].Host})
		}
	}
	return drifts, nil
}

// NormalizeHost returns the host form that profile.WithHost compares, so host
// equality in state checks agrees with profile lookup.
func NormalizeHost(host string) string {
	return (&config.Config{Host: host}).CanonicalHostName()
}
