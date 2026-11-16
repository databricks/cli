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
	"sync/atomic"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// stateFileName is the state file under ~/.databricks/aitools. It is not a
// libs/cache entry: cache entries are namespaced by CLI version and expire,
// and this state must outlive upgrades and `databricks cache clear`.
const stateFileName = "auto-install.json"

// State is what auto-install remembers across CLI versions.
type State struct {
	// LastVersion is the CLI version auto-install last ran for.
	LastVersion string `json:"last_version,omitempty"`

	// Attempts counts the failed attempts for LastVersion. It is 0 once a run
	// for LastVersion succeeded or was skipped for good.
	Attempts int `json:"attempts,omitempty"`

	// OptedOut turns auto-install off for every later version. It is set by
	// `databricks aitools auto-install disable`, by a global-scope
	// `databricks aitools uninstall`, by declining the install prompt and by a
	// "no" to the skills prompt of `databricks apps init`.
	OptedOut bool `json:"opted_out,omitempty"`

	// ID is a random per-machine identifier, created on the first run that
	// decides anything. Only its Bucket is used, and it never leaves the
	// machine. The CLI has no other stable machine identifier.
	ID string `json:"id,omitempty"`
}

// statePath returns ~/.databricks/aitools/auto-install.json.
func statePath(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "aitools", stateFileName), nil
}

// Load reads the state. A missing file is the zero State. It holds the state
// lock while it reads: on Windows a file another process has open cannot be
// replaced, so an unlocked read could make a concurrent write fail.
func Load(ctx context.Context) (State, error) {
	unlock, err := lock(ctx, stateLockFileName, stateLockWait)
	if err != nil {
		return State{}, err
	}
	defer unlock()
	return load(ctx)
}

// load reads the state; the caller holds the state lock.
func load(ctx context.Context) (State, error) {
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

// save writes the state atomically; the caller holds the state lock.
func save(ctx context.Context, s State) error {
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
	unlock, err := lock(ctx, stateLockFileName, stateLockWait)
	if err != nil {
		return err
	}
	defer unlock()
	s, err := load(ctx)
	if err != nil {
		log.Debugf(ctx, "Replacing unreadable auto-install state: %v", err)
		s = State{}
	}
	s.OptedOut = optedOut
	return save(ctx, s)
}

// declined is set once the user turned the tools down in this process.
var declined atomic.Bool

// Decline turns auto-install off after the user turned the tools down: for
// this process at once, and for later ones through the stored opt-out. The
// error is that of storing the opt-out; this process honors the answer either
// way.
func Decline(ctx context.Context) error {
	declined.Store(true)
	return SetOptedOut(ctx, true)
}

// Declined reports whether Decline ran in this process.
func Declined() bool {
	return declined.Load()
}

// ChildEnv returns the environment for a CLI process this one starts: the
// current environment, with auto-install turned off when the user declined in
// this process, so the child honors the answer even if storing it failed.
func ChildEnv() []string {
	environ := os.Environ()
	if Declined() {
		environ = append(environ, EnvVar+"=false")
	}
	return environ
}

// updateState applies change to the stored state and saves it. Every read and
// change of the state goes through the state lock, so that concurrent CLI
// processes do not overwrite each other's changes.
func updateState(ctx context.Context, change func(*State)) error {
	unlock, err := lock(ctx, stateLockFileName, stateLockWait)
	if err != nil {
		return err
	}
	defer unlock()
	s, err := load(ctx)
	if err != nil {
		return err
	}
	change(&s)
	return save(ctx, s)
}

// EnvVar turns auto-install off when set to false (or 0, no, off).
const EnvVar = "DATABRICKS_AITOOLS_AUTO_INSTALL"

// EnvOptedOut reports whether EnvVar turns auto-install off.
func EnvOptedOut(ctx context.Context) bool {
	enabled, ok := env.GetBool(ctx, EnvVar)
	return ok && !enabled
}
