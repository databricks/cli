package autoinstall

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

const (
	// versionEnv sets the CLI version auto-install sees. Acceptance tests use
	// it to simulate releases and upgrades: their binaries are development
	// builds, which never auto-install. Not a documented, user-facing setting.
	versionEnv = "DATABRICKS_AITOOLS_AUTO_INSTALL_CLI_VERSION"

	// cacheEnabledEnv mirrors libs/cache's env knob: with the cache turned
	// off, auto-install does nothing.
	cacheEnabledEnv = "DATABRICKS_CACHE_ENABLED"

	// lockFileName is the lock of a run, held while it installs.
	lockFileName = "auto-install.lock"

	// stateLockFileName is the lock of the state file, held while it is read
	// and rewritten.
	stateLockFileName = "auto-install.json.lock"

	// lockRetryInterval is how often a held lock is tried again.
	lockRetryInterval = 10 * time.Millisecond
)

// Version returns the CLI version auto-install runs for.
func Version(ctx context.Context) string {
	if v := env.Get(ctx, versionEnv); v != "" {
		return v
	}
	return build.GetInfo().Version
}

// CacheDisabled reports whether DATABRICKS_CACHE_ENABLED turns the CLI's cache
// off.
func CacheDisabled(ctx context.Context) bool {
	enabled, ok := env.GetBool(ctx, cacheEnabledEnv)
	return ok && !enabled
}

// GateOn reports whether the compatibility manifest turns auto-install on for
// version. It fails when the manifest could not be fetched: the gate is then
// unknown, not off.
func GateOn(ctx context.Context, version string) (bool, error) {
	return clicompat.ResolveAutoInstall(ctx, version)
}

// NextAttempt returns the number of the attempt a run for version makes after
// prior: 1 for a version the state has not failed for.
func NextAttempt(prior State, version string) int {
	if prior.LastVersion == version {
		return prior.Attempts + 1
	}
	return 1
}

// NewID returns a random machine identifier for State.ID.
func NewID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Bucket maps a machine identifier to its stable holdout bucket in [0, 100).
func Bucket(id string) int {
	sum := sha256.Sum256([]byte("aitools-auto-install:" + id))
	return int(binary.BigEndian.Uint64(sum[:8]) % 100)
}

// Record stores the outcome of a run for version: attempts is 0 when the
// version is settled, or the number of failed attempts so far. now is when the
// last of them started, or zero when no install started. It re-reads the state
// and changes only the fields a run owns, so an opt-out stored while the run
// was installing is kept.
func Record(ctx context.Context, version string, attempts int, id string, now time.Time) error {
	return updateState(ctx, func(s *State) { record(s, version, attempts, id, now) })
}

// StartAttempt records attempt as Record does, unless the state is opted out
// by then, and reports whether it did. A run counts the attempt right before
// it installs, so an opt-out stored after the run read the state (during the
// gate lookup or the install prompt, say) still stops it.
func StartAttempt(ctx context.Context, version string, attempt int, id string, now time.Time) (started bool, err error) {
	err = updateState(ctx, func(s *State) {
		if s.OptedOut {
			return
		}
		record(s, version, attempt, id, now)
		started = true
	})
	return started, err
}

// record is the change of Record.
func record(s *State, version string, attempts int, id string, now time.Time) {
	s.LastVersion = version
	s.Attempts = attempts
	if attempts == 0 {
		s.Installing = false
		s.InstallStateDigest = ""
		s.LastAttempt = time.Time{}
	} else {
		s.LastAttempt = now.UTC()
	}
	if s.ID == "" {
		s.ID = id
	}
}

// RecordUnfinished settles version as Record does with attempts 0, but keeps
// State.Installing and State.InstallStateDigest: the first install was
// refused the skills, so an agent did not get its tools, and a later version
// finishes the install.
func RecordUnfinished(ctx context.Context, version, id string) error {
	return updateState(ctx, func(s *State) {
		installing, digest := s.Installing, s.InstallStateDigest
		record(s, version, 0, id, time.Time{})
		s.Installing, s.InstallStateDigest = installing, digest
	})
}

// MarkInstalling sets State.Installing before a first install starts, and
// State.InstalledWithoutConsent when the user has not answered.
func MarkInstalling(ctx context.Context) error {
	return updateState(ctx, func(s *State) {
		s.Installing = true
		if s.Consent() == ConsentUnknown {
			s.InstalledWithoutConsent = true
		}
	})
}

// SaveInstallStateDigest sets State.InstallStateDigest after a first install
// that did not finish.
func SaveInstallStateDigest(ctx context.Context, digest string) error {
	return updateState(ctx, func(s *State) { s.InstallStateDigest = digest })
}

// SaveID stores id as the machine identifier unless one is stored already,
// without settling a version, and returns the stored identifier. Runs that
// draw an ID save it before using it, so concurrent runs agree on the
// machine's holdout bucket.
func SaveID(ctx context.Context, id string) (string, error) {
	err := updateState(ctx, func(s *State) {
		if s.ID == "" {
			s.ID = id
		}
		id = s.ID
	})
	return id, err
}

// Accept records that the user answered "yes" at the install prompt of
// version (State.AcceptedVersion).
func Accept(ctx context.Context, version string) error {
	return updateState(ctx, func(s *State) { s.AcceptedVersion = version })
}

// TryLock claims the machine-wide auto-install lock so that concurrent CLI
// processes (agents often run several at once) do not install at the same
// time. ok is false when another process holds it or it cannot be taken; that
// run then skips auto-install. It is an OS file lock, so a process that dies
// mid-run releases it.
func TryLock(ctx context.Context) (unlock func(), ok bool) {
	unlock, err := lock(ctx, lockFileName, 0)
	if err != nil {
		log.Debugf(ctx, "auto-install: could not take the lock: %v", err)
		return nil, false
	}
	return unlock, true
}

// Lock takes the lock TryLock takes, without waiting, so that a command that
// removes what a run installs (a global uninstall) never runs while a run
// installs. It fails when a run holds it. Where the lock cannot be taken at
// all (a filesystem without locks), no run can take it either, so Lock
// returns an unlock that does nothing.
func Lock(ctx context.Context) (unlock func(), err error) {
	unlock, err = lock(ctx, lockFileName, 0)
	if err != nil && !errors.Is(err, errLockHeld) {
		log.Debugf(ctx, "Going ahead without the auto-install lock: %v", err)
		return func() {}, nil
	}
	return unlock, err
}

// lock takes the exclusive lock file name next to the state. It retries for
// up to wait, so a holder that is stopped (a suspended process, say) delays
// the caller by at most wait, then fails it. The file is never removed: a
// process that removed it could let a third one lock a new file while a
// second still holds the old.
func lock(ctx context.Context, name string, wait time.Duration) (unlock func(), err error) {
	path, err := statePath(ctx)
	if err != nil {
		return nil, err
	}
	path = filepath.Join(filepath.Dir(path), name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(wait)
	for {
		err = tryLock(f)
		if err == nil {
			return func() { _ = f.Close() }, nil
		}
		// Only a lock another process holds can come free; any other error
		// (a filesystem without locks, say) fails the same way on every try.
		if !errors.Is(err, errLockHeld) {
			_ = f.Close()
			return nil, fmt.Errorf("lock %s: %w", filepath.ToSlash(path), err)
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			_ = f.Close()
			return nil, fmt.Errorf("%s: %w", filepath.ToSlash(path), err)
		}
		time.Sleep(lockRetryInterval)
	}
}

// stateLockWait bounds the wait for the state lock. A live holder keeps it
// only to read or rewrite a small file. Tests that queue many holders raise it.
var stateLockWait = time.Second

// errLockHeld is the error of tryLockFile when another process holds the lock.
var errLockHeld = errors.New("held by another CLI process")

// tryLock is tryLockFile; tests replace it.
var tryLock = tryLockFile
