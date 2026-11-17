// Package autoinstall holds what the CLI remembers about installing Databricks
// AI tools automatically on the first run of a new CLI version.
package autoinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// stateFileName is the state file under ~/.databricks/aitools. It is not a
// libs/cache entry: cache entries are namespaced by CLI version and expire,
// and this state must outlive upgrades and `databricks cache clear`.
const stateFileName = "auto-install.json"

// EnvVar turns auto-install off when set to anything but a true value (true,
// 1, yes, on), including an empty or unparseable value.
const EnvVar = "DATABRICKS_AITOOLS_AUTO_INSTALL"

// State is what auto-install remembers across CLI versions.
type State struct {
	// LastVersion is the CLI version auto-install last ran for.
	LastVersion string `json:"last_version,omitempty"`

	// Attempts counts the failed attempts for LastVersion. It is 0 once a run
	// for LastVersion succeeded or was skipped for good.
	Attempts int `json:"attempts,omitempty"`

	// OptedOut turns auto-install off for every later version. It is set by
	// `databricks aitools auto-install disable` and by a global-scope
	// `databricks aitools uninstall`.
	OptedOut bool `json:"opted_out,omitempty"`
}

// statePath returns ~/.databricks/aitools/auto-install.json.
func statePath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "aitools", stateFileName), nil
}

// Load reads the state. A missing file is the zero State.
func Load(ctx context.Context) (State, error) {
	path, err := statePath(ctx)
	if err != nil {
		return State{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, fmt.Errorf("parse %s: %w", filepath.ToSlash(path), err)
	}
	return s, nil
}

// Save writes the state atomically.
func Save(ctx context.Context, s State) error {
	path, err := statePath(ctx)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(path, append(data, '\n'), 0o600, atomicfile.MkDir(0o755))
}

// SetOptedOut turns auto-install off (true) or back on (false), keeping the
// rest of the state. A state file that cannot be read is replaced: the user's
// explicit choice matters more than a broken record.
func SetOptedOut(ctx context.Context, optedOut bool) error {
	s, err := Load(ctx)
	if err != nil {
		log.Debugf(ctx, "Replacing unreadable auto-install state: %v", err)
		s = State{}
	}
	s.OptedOut = optedOut
	return Save(ctx, s)
}

// EnvOptedOut reports whether EnvVar turns auto-install off.
func EnvOptedOut(ctx context.Context) bool {
	enabled, ok := env.GetBool(ctx, EnvVar)
	return ok && !enabled
}
