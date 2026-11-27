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
	"time"

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

	// LastAttempt is when the last of Attempts started. The next attempt
	// waits until a delay after it has passed (see Decide). It is zero once
	// LastVersion is settled, and when no install started (a failed gate
	// lookup uses up the attempts of LastVersion without one).
	LastAttempt time.Time `json:"last_attempt,omitzero"`

	// OptedOut turns auto-install off for every later version. It is set by
	// `databricks aitools auto-install disable`, by a global-scope
	// `databricks aitools uninstall`, by declining the install prompt and by a
	// "no" to the skills prompt of `databricks apps init`.
	OptedOut bool `json:"opted_out,omitempty"`

	// AcceptedVersion is the CLI version whose install prompt the user
	// answered "yes". Retries of that version install without asking again;
	// any other version asks.
	AcceptedVersion string `json:"accepted_version,omitempty"`

	// Installing is set while auto-install makes a first install, and stays
	// set when the install did not finish, including one that was refused
	// the skills (see RecordUnfinished). The skills such an install left
	// have no install state, like the ones the user put there, so a later run
	// uses it to finish the install instead of leaving the skills alone.
	Installing bool `json:"installing,omitempty"`

	// InstallStateDigest is the digest of the install state that an
	// unfinished first install saved (one that failed only for some agents,
	// or that installed the plugins and was refused the skills).
	// While the install state still has it, nothing else wrote it, so a later
	// run finishes the install as a first install instead of updating it.
	// Install state that anything else wrote is updated. It is cleared with
	// Installing.
	InstallStateDigest string `json:"install_state_digest,omitempty"`

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
	// The retry wait shifts by the count, which panics when it is negative.
	if s.Attempts < 0 {
		return State{}, fmt.Errorf("parse %s: negative attempts %d", filepath.ToSlash(path), s.Attempts)
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
// explicit choice matters more than a broken record. Where the state lock
// cannot be taken at all (a filesystem without locks), it writes without it:
// auto-install cannot run there, so nothing else writes the state.
func SetOptedOut(ctx context.Context, optedOut bool) error {
	unlock, err := lock(ctx, stateLockFileName, stateLockWait)
	switch {
	case errors.Is(err, errLockHeld):
		return err
	case err != nil:
		log.Debugf(ctx, "Saving the auto-install state without its lock: %v", err)
	default:
		defer unlock()
	}
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

// ResetDeclined forgets a Decline in this process. Tests that decline use it
// to clean up.
func ResetDeclined() {
	declined.Store(false)
}

// ChildEnv returns environ for a CLI process this one starts, with
// auto-install turned off when the user declined in this process, so the child
// honors the answer even if storing it failed.
func ChildEnv(environ []string) []string {
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

// EnvOptedOut reports whether EnvVar turns auto-install off.
func EnvOptedOut(ctx context.Context) bool {
	enabled, ok := env.GetBool(ctx, EnvVar)
	return ok && !enabled
}
