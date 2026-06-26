package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/libs/env"
)

const stateFileName = ".state.json"

// schemaVersionV2 is the current on-disk state schema version. v2 adds the
// additive Plugins and Files maps; v1 state loads forward without data changes.
const schemaVersionV2 = 2

// Scope constants for skill installation.
const (
	ScopeGlobal  = "global"
	ScopeProject = "project"
)

// InstallState records the state of all installed skills in a scope directory.
type InstallState struct {
	SchemaVersion       int               `json:"schema_version"`
	IncludeExperimental bool              `json:"include_experimental,omitempty"`
	Release             string            `json:"release"`
	LastUpdated         time.Time         `json:"last_updated"`
	Skills              map[string]string `json:"skills"`
	RepoDirs            map[string]string `json:"repo_dirs,omitempty"`
	Scope               string            `json:"scope,omitempty"`

	// Plugins records databricks plugins installed through an agent's own CLI,
	// keyed by registry agent name (e.g. "claude-code"). Added in schema v2;
	// omitted for files-only installs.
	Plugins map[string]PluginRecord `json:"plugins,omitempty"`
	// Files records provenance for skill files the CLI wrote, keyed by the
	// file path relative to the scope's canonical skills dir (forward slashes,
	// e.g. "databricks/SKILL.md"). Added in schema v2; used to prune only the
	// skills we wrote that the user hasn't modified.
	Files map[string]FileRecord `json:"files,omitempty"`
}

// PluginRecord records a databricks plugin installed for an agent through the
// agent's own plugin CLI, so update/uninstall act on exactly where we
// installed and list/version can report real plugin state.
type PluginRecord struct {
	// Marketplace is the marketplace registry name the plugin was installed from.
	Marketplace string `json:"marketplace"`
	// Plugin is the installed plugin id (e.g. "databricks").
	Plugin string `json:"plugin"`
	// Scope is the agent-native install scope (e.g. "user" or "project").
	Scope string `json:"scope,omitempty"`
	// Version is the last seen plugin/release version, when known.
	Version string `json:"version,omitempty"`
	// InstalledMarketplace is true when this CLI registered the marketplace,
	// so uninstall may de-register it. False when it was already present and we
	// must leave it for whatever else shares it.
	InstalledMarketplace bool `json:"installed_marketplace,omitempty"`
}

// FileRecord records provenance for a single skill file the CLI wrote, so
// update can prune a vanished skill only when the on-disk file still matches
// what we wrote (i.e. the user hasn't modified it).
type FileRecord struct {
	// SHA256 is the hex-encoded checksum of the file content the CLI wrote.
	SHA256 string `json:"sha256"`
	// Origin is the skills ref the file was fetched from, when known.
	Origin string `json:"origin,omitempty"`
}

// LoadState reads install state from the given directory.
// Returns (nil, nil) when the state file does not exist.
func LoadState(dir string) (*InstallState, error) {
	data, err := os.ReadFile(filepath.Join(dir, stateFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state InstallState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	migrateState(&state)
	return &state, nil
}

// migrateState brings a loaded state forward to the current schema version. It
// is forward-only and idempotent. v1 -> v2 needs no data transformation (the
// Plugins/Files maps are additive and optional), so it only stamps the version;
// writers lazily initialize the maps, matching how RepoDirs is handled.
func migrateState(state *InstallState) {
	if state.SchemaVersion < schemaVersionV2 {
		state.SchemaVersion = schemaVersionV2
	}
}

// SaveState writes install state to the given directory atomically.
// Creates the directory if it does not exist.
func SaveState(dir string, state *InstallState) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	data = append(data, '\n')

	// Atomic write: write to temp file in the same directory, then rename.
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, filepath.Join(dir, stateFileName)); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename state file: %w", err)
	}
	return nil
}

// GlobalSkillsDir returns the path to the global skills directory (~/.databricks/aitools/skills/).
func GlobalSkillsDir(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "aitools", "skills"), nil
}

// ProjectSkillsDir returns the path to the project-scoped skills directory.
// The project root is the current working directory.
func ProjectSkillsDir(_ context.Context) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	return filepath.Join(cwd, ".databricks", "aitools", "skills"), nil
}
