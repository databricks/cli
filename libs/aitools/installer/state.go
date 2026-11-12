package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
)

// clearFileRecords removes all file-provenance entries belonging to a skill
// (keys are "<skill>/<file>"), so refetching or removing a skill never leaves
// orphaned records behind.
func clearFileRecords(files map[string]FileRecord, skillName string) {
	prefix := skillName + "/"
	for path := range files {
		if strings.HasPrefix(path, prefix) {
			delete(files, path)
		}
	}
}

const stateFileName = ".state.json"

// CurrentSchemaVersion is the on-disk state schema version this CLI writes. v2
// added the Plugins and Files maps; v3 adds the optional MCP section. Older state
// loads forward without data changes.
const CurrentSchemaVersion = 3

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
	// MCP records the Databricks MCP servers the `aitools mcp` commands wrote.
	// Added in schema v3; omitted when none are recorded. Other writers keep it
	// by loading the file first, since they save the whole struct.
	MCP *MCPState `json:"mcp,omitempty"`
}

// MCPState is the MCP section of the install state: the servers this CLI added
// to agent configs, and how each one is bound to a Databricks profile.
type MCPState struct {
	Servers []MCPEntry `json:"servers"`
}

// MCPEntry is one installed MCP server: a service bound to one profile for one set of agents.
type MCPEntry struct {
	Name           string                    `json:"name"`
	Service        string                    `json:"service"`
	Profile        string                    `json:"profile"`
	Host           string                    `json:"host"`
	WorkspaceID    string                    `json:"workspace_id"`
	SignInRequired bool                      `json:"sign_in_required"`
	Command        string                    `json:"command"`
	ConfigFile     string                    `json:"config_file"`
	Agents         map[string]MCPAgentRecord `json:"agents"`
	InstalledAt    time.Time                 `json:"installed_at"`
	CLIVersion     string                    `json:"cli_version"`
}

// MCPAgentRecord is where an MCP entry was written for one agent.
type MCPAgentRecord struct {
	Path  string `json:"path"`
	Scope string `json:"scope"`
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

// HasSkillsOrPlugins reports whether the state records an installed skill or plugin. MCP records
// share the file but are not installs, so a state holding only MCP records reports false, and so
// does a nil state.
func (s *InstallState) HasSkillsOrPlugins() bool {
	return s != nil && (len(s.Skills) > 0 || len(s.Plugins) > 0)
}

// LoadState reads install state from the given directory.
// Returns (nil, nil) when the state file does not exist.
func LoadState(dir string) (*InstallState, error) {
	path := filepath.Join(dir, stateFileName)
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state InstallState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, &StateError{Path: path, Err: fmt.Errorf("failed to parse state file: %w", err)}
	}
	migrateState(&state)
	return &state, nil
}

// migrateState brings a loaded state forward to the current schema version. It
// is forward-only and idempotent. The v1 -> v2 and v2 -> v3 steps need no data
// transformation (the Plugins, Files and MCP fields are additive and optional),
// so migration only stamps the version; writers lazily initialize the maps,
// matching how RepoDirs is handled.
func migrateState(state *InstallState) {
	if state.SchemaVersion < CurrentSchemaVersion {
		state.SchemaVersion = CurrentSchemaVersion
	}
}

// SaveState writes the skills and plugins records of state to the given directory atomically,
// creating the directory if it does not exist. The MCP section is written by SaveMCPState, so
// SaveState keeps the one on disk, not the one in state. This keeps MCP records saved after state
// was loaded.
func SaveState(dir string, state *InstallState) error {
	onDisk, err := LoadState(dir)
	if err != nil {
		return err
	}
	out := *state
	out.MCP = nil
	if onDisk != nil {
		out.MCP = onDisk.MCP
	}
	return writeState(dir, &out)
}

// SaveMCPState replaces the MCP section of the state file in the given directory. The file is read
// again first, so the skills and plugins records are kept. A nil or empty state removes the MCP
// section, and the file is removed once no skills, plugins or MCP servers remain.
func SaveMCPState(dir string, mcp *MCPState) error {
	st, err := LoadState(dir)
	if err != nil {
		return err
	}
	if st == nil {
		if mcp == nil || len(mcp.Servers) == 0 {
			return nil
		}
		st = &InstallState{
			SchemaVersion: CurrentSchemaVersion,
			Skills:        map[string]string{},
			RepoDirs:      map[string]string{},
			Files:         map[string]FileRecord{},
		}
	}
	st.MCP = nil
	if mcp != nil && len(mcp.Servers) > 0 {
		st.MCP = mcp
	}
	if !st.HasSkillsOrPlugins() && st.MCP == nil {
		err := os.Remove(filepath.Join(dir, stateFileName))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove state file: %w", err)
		}
		return nil
	}
	return writeState(dir, st)
}

// mcpRecordedOnDisk reports whether the state file in the given directory holds MCP servers.
func mcpRecordedOnDisk(dir string) (bool, error) {
	st, err := LoadState(dir)
	if err != nil {
		return false, err
	}
	return st != nil && st.MCP != nil && len(st.MCP.Servers) > 0, nil
}

func writeState(dir string, state *InstallState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	data = append(data, '\n')

	return atomicfile.Write(filepath.Join(dir, stateFileName), data, 0o600, atomicfile.MkDir(0o755))
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
