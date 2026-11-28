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
	// `databricks aitools uninstall` and by a "no" to the skills prompt of
	// `databricks apps init` or to the question of `databricks auth login`.
	OptedOut bool `json:"opted_out,omitempty"`

	// Consented records that the user agreed to installing the tools and
	// keeping them up to date. It is set by `databricks aitools auto-install
	// enable`, by a global-scope `databricks aitools install` that set an agent
	// up, by a "yes" to the skills prompt of `databricks apps init` or to the
	// question of `databricks auth login`, and by a run that finds a global
	// installation made by the CLI. It is never set together with OptedOut
	// (see Consent), so a CLI that knows only OptedOut still reads an opt-out
	// as one.
	Consented bool `json:"consented,omitempty"`

	// ConsentAsked records that `databricks auth login` asked the user for
	// the consent, so that it never asks again, whatever the answer, and also
	// when there was none. It is set before the question is shown. A CLI that
	// does not know it ignores it when it reads the state, and drops it when
	// it rewrites the state; the question can then be asked once more.
	ConsentAsked bool `json:"consent_asked,omitempty"`

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

// Consent is the user's answer to installing Databricks AI tools and keeping
// them up to date. It is one answer per machine, for every CLI version.
type Consent int

const (
	// ConsentUnknown is a machine whose user never answered.
	ConsentUnknown Consent = iota
	// ConsentGiven is a machine whose user agreed.
	ConsentGiven
	// ConsentOptedOut is a machine whose user refused.
	ConsentOptedOut
)

// Consent returns the stored answer. A state with both OptedOut and Consented
// set, which only a hand edit makes, is opted out.
func (s State) Consent() Consent {
	switch {
	case s.OptedOut:
		return ConsentOptedOut
	case s.Consented:
		return ConsentGiven
	}
	return ConsentUnknown
}

// setConsent stores c in s, clearing the other answer.
func setConsent(s *State, c Consent) {
	s.OptedOut = c == ConsentOptedOut
	s.Consented = c == ConsentGiven
}

// SetConsent stores the user's answer c, keeping the rest of the state, and
// returns the answer stored before. A state file that cannot be read is
// replaced: the user's explicit choice matters more than a broken record.
// Where the state lock cannot be taken at all (a filesystem without locks),
// it writes without it: auto-install cannot run there, so nothing else writes
// the state.
func SetConsent(ctx context.Context, c Consent) (previous Consent, err error) {
	unlock, err := lock(ctx, stateLockFileName, stateLockWait)
	switch {
	case errors.Is(err, errLockHeld):
		return ConsentUnknown, err
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
	previous = s.Consent()
	setConsent(&s, c)
	return previous, save(ctx, s)
}

// GiveConsentIfUnknown stores ConsentGiven unless the user answered already,
// and returns the answer stored before. It is for actions that count as
// consent without asking for it, such as an install, so an earlier opt-out
// stays. For the same reason it fails on a state file it cannot read, rather
// than replacing it: the file may hold an opt-out.
func GiveConsentIfUnknown(ctx context.Context) (previous Consent, err error) {
	err = updateState(ctx, func(s *State) {
		previous = s.Consent()
		if previous == ConsentUnknown {
			setConsent(s, ConsentGiven)
		}
	})
	return previous, err
}

// MarkConsentAsked sets State.ConsentAsked, unless the user answered already
// or was asked before, and reports whether it did. The question is asked only
// when it did, so that of two CLI processes that would ask at once, one does.
// It fails rather than replace a state file it cannot read.
func MarkConsentAsked(ctx context.Context) (marked bool, err error) {
	err = updateState(ctx, func(s *State) {
		if s.Consent() == ConsentUnknown && !s.ConsentAsked {
			s.ConsentAsked = true
			marked = true
		}
	})
	return marked && err == nil, err
}

// declined is set once the user turned the tools down in this process.
var declined atomic.Bool

// Decline turns auto-install off after the user turned the tools down: for
// this process at once, and for later ones through the stored opt-out. It
// returns the answer stored before. The error is that of storing the opt-out;
// this process honors the answer either way.
func Decline(ctx context.Context) (previous Consent, err error) {
	declined.Store(true)
	return SetConsent(ctx, ConsentOptedOut)
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
