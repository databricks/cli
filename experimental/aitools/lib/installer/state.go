package installer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
)

const stateFileName = ".state.json"

// ErrNotImplemented indicates that a feature is not yet implemented.
var ErrNotImplemented = errors.New("project scope not yet implemented")

// InstalledSkill records the installed version and timestamp for a single skill.
type InstalledSkill struct {
	Version     string `json:"version"`
	InstalledAt string `json:"installed_at"`
}

// InstallState records the state of all installed skills in a scope directory.
type InstallState struct {
	SchemaVersion int                       `json:"schema_version"`
	SkillsRef     string                    `json:"skills_ref"`
	LastChecked   string                    `json:"last_checked,omitempty"`
	Skills        map[string]InstalledSkill `json:"skills"`
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
	return &state, nil
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
// Project scope is not yet implemented.
func ProjectSkillsDir(_ context.Context) (string, error) {
	return "", ErrNotImplemented
}
