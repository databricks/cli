package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/google/uuid"
)

// LocalStateDir returns the local directory where state artifacts are mirrored
// for u. Always forward-slash on the wire; callers that write to disk should
// pass the return value through filepath.FromSlash when interacting with the
// OS filesystem.
func LocalStateDir(u *ucm.Ucm) string {
	target := u.Config.Ucm.Target
	return filepath.Join(u.RootPath, filepath.FromSlash(LocalCacheDir), target)
}

// Pull copies terraform.tfstate and ucm-state.json from the remote StateFiler
// into the per-target local cache directory. On a first run the remote files
// are absent and a fresh local state with Seq=0 is written instead.
//
// The deploy lock is acquired for the duration of the pull so that a
// concurrent Push cannot swap the remote blob out from under us mid-read.
func Pull(ctx context.Context, u *ucm.Ucm, b Backend) error {
	if u == nil {
		return errors.New("ucm state: Pull called with nil Ucm")
	}
	if b.StateFiler == nil || b.LockFiler == nil {
		return errors.New("ucm state: Pull requires StateFiler and LockFiler in Backend")
	}

	l := newLocker(b, ".")
	if err := l.Acquire(ctx, b.ForceLock); err != nil {
		return fmt.Errorf("ucm state: acquire lock: %w", err)
	}
	defer releaseBestEffort(ctx, l, lock.GoalDeploy)

	localDir := LocalStateDir(u)
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("ucm state: create local cache dir: %w", err)
	}

	remoteUcm, err := readRemoteUcmState(ctx, b.StateFiler)
	if err != nil {
		return fmt.Errorf("ucm state: read remote %s: %w", UcmStateFileName, err)
	}

	if remoteUcm == nil {
		log.Infof(ctx, "ucm state: remote is empty, initializing fresh local state at %s", filepath.ToSlash(localDir))
		return writeFreshLocal(ctx, localDir)
	}

	if err := validateCompatibility(remoteUcm); err != nil {
		return err
	}

	log.Infof(ctx, "ucm state: pulling remote state (seq %d) into %s", remoteUcm.Seq, filepath.ToSlash(localDir))
	if err := writeLocalState(localDir, remoteUcm); err != nil {
		return fmt.Errorf("ucm state: write local %s: %w", UcmStateFileName, err)
	}

	localTfPath := LocalTfStatePath(u)
	if err := os.MkdirAll(filepath.Dir(localTfPath), 0o755); err != nil {
		return fmt.Errorf("ucm state: create terraform working dir: %w", err)
	}
	if err := copyRemoteToLocal(ctx, b.StateFiler, TfStateFileName, localTfPath); err != nil {
		if errors.Is(err, filer.ErrNotFound) {
			// A ucm-state.json without a sibling terraform.tfstate is a
			// recoverable first-run shape (ucm pulled once but never
			// applied terraform). Leave local tfstate absent.
			log.Infof(ctx, "ucm state: no remote %s yet; skipping", TfStateFileName)
			return nil
		}
		return fmt.Errorf("ucm state: copy remote %s: %w", TfStateFileName, err)
	}
	return nil
}

// writeFreshLocal writes a Seq=0 ucm-state.json to localDir and nothing else.
// The absence of a local terraform.tfstate is the signal downstream phases
// use to recognise first-run behaviour.
func writeFreshLocal(ctx context.Context, localDir string) error {
	fresh := newFreshState()
	if err := writeLocalState(localDir, fresh); err != nil {
		return fmt.Errorf("ucm state: write fresh local state: %w", err)
	}
	log.Infof(ctx, "ucm state: wrote fresh local state (seq 0) to %s", filepath.ToSlash(localDir))
	return nil
}

// newFreshState builds a Seq=0 State with a newly-generated ID, used on
// first-run and after destroy.
func newFreshState() *State {
	return &State{
		Version: StateVersion,
		Seq:     0,
		ID:      uuid.New(),
	}
}

// writeLocalState writes s to <localDir>/ucm-state.json with 0600 perms so
// that workspace-side credentials embedded in terraform.tfstate (if any later
// land there) aren't world-readable on shared dev machines.
func writeLocalState(localDir string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(localDir, UcmStateFileName), data, 0o600)
}

// copyRemoteToLocal streams a single remote file into a local file,
// truncating any existing local copy. The buffered ReadAll keeps the
// implementation simple; state files are small.
func copyRemoteToLocal(ctx context.Context, f filer.StateFiler, remote, localPath string) error {
	rc, err := f.Read(ctx, remote)
	if err != nil {
		return err
	}
	defer rc.Close()

	buf, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath, buf, 0o600)
}

// releaseBestEffort is deferred by Pull/Push so the lock is released on every
// exit path including errors. Release failures are logged rather than
// surfaced because the primary error already communicates the deploy outcome
// and a stuck lock is more operator-visible than a swallowed one.
func releaseBestEffort(ctx context.Context, l *lock.Locker, goal lock.Goal) {
	if err := l.Release(ctx, goal); err != nil {
		log.Warnf(ctx, "ucm state: release lock (goal %s): %v", goal, err)
	}
}

// readLocalState is used by Push to learn the Seq we think we're advancing.
// Shared here so state_pull_test.go can exercise the same file format.
func readLocalState(localDir string) (*State, error) {
	data, err := os.ReadFile(filepath.Join(localDir, UcmStateFileName))
	if err != nil {
		return nil, err
	}
	return loadState(bytes.NewReader(data))
}
