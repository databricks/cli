package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
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
	// Added in schema v3; omitted when none are recorded. SaveState ignores this
	// field and keeps the section on disk; SaveMCPState is the only writer of it.
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

// hasInstallerRecords reports whether the state holds anything the installer writes: a release,
// skills, plugins, files or repo dirs.
func hasInstallerRecords(s *InstallState) bool {
	return s.Release != "" || len(s.Skills) > 0 || len(s.Plugins) > 0 || len(s.Files) > 0 || len(s.RepoDirs) > 0
}

// isMCPOnly reports whether the state holds MCP records and nothing the installer wrote. The installer
// records a release in every state it saves, so such a file was written by the MCP commands alone. A file
// without an MCP section is never MCP-only, so a state the installer left behind, even an empty one, keeps
// the behavior it always had.
func (s *InstallState) isMCPOnly() bool {
	return s != nil && s.MCP != nil && !hasInstallerRecords(s)
}

// LoadState reads install state from the given directory.
// Returns (nil, nil) when the state file does not exist or holds only MCP records, since those are not
// an install of skills or plugins. LoadMCPState reads the MCP records.
func LoadState(dir string) (*InstallState, error) {
	state, err := readState(dir)
	if err != nil || state.isMCPOnly() {
		return nil, err
	}
	return state, nil
}

// LoadMCPState reads the MCP section of the state file in the given directory. It returns nil when the
// file does not exist or has no MCP section.
func LoadMCPState(dir string) (*MCPState, error) {
	state, err := readState(dir)
	if err != nil || state == nil {
		return nil, err
	}
	return state.MCP, nil
}

// readState reads the state file in the given directory, whatever it holds. It returns (nil, nil) when
// the file does not exist.
func readState(dir string) (*InstallState, error) {
	data, err := os.ReadFile(filepath.Join(dir, stateFileName))
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

// SaveState writes state to the given directory atomically, creating the directory if it does not
// exist. The MCP section is the one on disk: SaveState does not write the MCP field of state, so
// MCP records saved after state was loaded are kept.
func SaveState(dir string, state *InstallState) error {
	onDisk, err := readState(dir)
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

// SaveMCPState replaces the MCP section of the state file in the given directory with mcp. The skills
// and plugins records are kept as they are on disk. A nil or empty mcp removes the section, and the file
// is removed once it holds nothing else.
func SaveMCPState(dir string, mcp *MCPState) error {
	st, err := readState(dir)
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
	if st.MCP == nil && !hasInstallerRecords(st) {
		return removeStateFile(dir)
	}
	return writeState(dir, st)
}

// removeState deletes the state file once uninstall has removed every skill and plugin. Uninstall does
// not own MCP records, so when servers remain the file is rewritten to hold only them.
func removeState(dir string) error {
	st, err := readState(dir)
	if err != nil {
		return err
	}
	if st != nil && st.MCP != nil && len(st.MCP.Servers) > 0 {
		return writeState(dir, &InstallState{SchemaVersion: CurrentSchemaVersion, MCP: st.MCP})
	}
	return removeStateFile(dir)
}

// removeStateFile deletes the state file in the given directory. A missing file is not an error.
func removeStateFile(dir string) error {
	err := os.Remove(filepath.Join(dir, stateFileName))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove state file: %w", err)
	}
	return nil
}

// writeState writes state to the state file in the given directory atomically, as written.
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
