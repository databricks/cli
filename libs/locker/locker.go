package locker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
)

type UnlockOption int

const (
	AllowLockFileNotExist UnlockOption = iota
)

const LockFileName = "deploy.lock"

// Locker object enables exclusive access to TargetDir's scope for a client. This
// enables multiple clients to deploy to the same scope (ie TargetDir) in an atomic
// manner
//
// Here are some of the details of the locking protocol used here:
//
//  1. Potentially multiple clients race to create a deploy.lock file in
//     TargetDir/.bundle directory with unique ID. The deploy.lock file
//     is a json file containing the State from the locker
//
//  2. Clients read the remote deploy.lock file and if it's ID matches, the client
//     assumes it has the lock on TargetDir. The client is now free to read/write code
//     asserts and deploy databricks assets scoped under TargetDir
//
//  3. To sidestep clients failing to relinquish a lock during a failed deploy attempt
//     we allow clients to forcefully acquire a lock on TargetDir. However forcefully acquired
//     locks come with the following caveats:
//
//     a.  a forcefully acquired lock does not guarantee exclusive access to
//     TargetDir's scope
//     b.  forcefully acquiring a lock(s) on TargetDir can break the assumption
//     of exclusive access that other clients with non forcefully acquired
//     locks might have
type Locker struct {
	filer filer.Filer

	// lockFiler writes the lock file. Unlike filer it does not retry writes: Lock retries them
	// itself, so that it knows when an attempt failed in a way that may still be applied later.
	lockFiler filer.Filer

	// createUncertain is set when a lock write failed without a definite answer from the server
	// (a timeout, a 5xx, a cut connection, a cancelled request). The server may still apply such a
	// write after we release the lock, leaving behind a lock with our ID.
	createUncertain bool

	// ReleasedLocksPath is a local file listing the IDs of locks released after an uncertain
	// write (see createUncertain). A lock that carries one of these IDs is stale: its owner has
	// already released it. Empty disables the list.
	ReleasedLocksPath string

	// scope of the locker
	TargetDir string
	// Active == true implies exclusive access to TargetDir for the client.
	// This implication break down if locks are forcefully acquired by a user
	Active bool
	// if locker is active, this information about the locker is uploaded onto
	// the workspace so as to let other clients details about the active locker
	State *LockState
}

type LockState struct {
	// unique identifier for the locker
	ID uuid.UUID
	// last timestamp when locker was active
	AcquisitionTime time.Time
	// Only relevant for active lockers
	// IsForced == true implies the lock was acquired forcefully
	IsForced bool
	// creator of this locker
	User string
}

// GetActiveLockState returns current lock state, irrespective of us holding it.
func (locker *Locker) GetActiveLockState(ctx context.Context) (*LockState, error) {
	reader, err := locker.filer.Read(ctx, LockFileName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	remoteLock := LockState{}
	err = json.Unmarshal(bytes, &remoteLock)
	if err != nil {
		return nil, err
	}

	return &remoteLock, nil
}

// asserts whether lock is held by locker. Returns descriptive error with current
// holder details if locker does not hold the lock
func (locker *Locker) assertLockHeld(ctx context.Context) error {
	activeLockState, err := locker.GetActiveLockState(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no active lock on target dir: %w", err)
	}
	if err != nil {
		return err
	}
	return locker.checkHolder(activeLockState)
}

// checkHolder returns a descriptive error naming the holder if activeLockState is not ours.
func (locker *Locker) checkHolder(activeLockState *LockState) error {
	if activeLockState.ID != locker.State.ID && !activeLockState.IsForced {
		return fmt.Errorf("deploy lock acquired by %s at %v.\n"+
			"Another deployment may be in progress. Use --force-lock to override, but this may\n"+
			"conflict with the other deployment if it is still active",
			activeLockState.User, activeLockState.AcquisitionTime)
	}
	if activeLockState.ID != locker.State.ID && activeLockState.IsForced {
		return fmt.Errorf("deploy lock force-acquired by %s at %v.\n"+
			"Another deployment may be in progress. Use --force-lock to override, but this may\n"+
			"conflict with the other deployment if it is still active",
			activeLockState.User, activeLockState.AcquisitionTime)
	}
	return nil
}

// idempotent function since overwrite is set to true
func (locker *Locker) Write(ctx context.Context, pathToFile string, content []byte) error {
	if !locker.Active {
		return errors.New("failed to put file. deploy lock not held")
	}
	return locker.filer.Write(ctx, pathToFile, bytes.NewReader(content), filer.OverwriteIfExists, filer.CreateParentDirectories)
}

func (locker *Locker) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	if !locker.Active {
		return nil, errors.New("failed to get file. deploy lock not held")
	}
	return locker.filer.Read(ctx, path)
}

func (locker *Locker) Lock(ctx context.Context, isForced bool) error {
	err := locker.lock(ctx, isForced)
	if err != nil && locker.createUncertain {
		// We do not hold the lock, but a write that timed out may still create it later.
		locker.recordReleased(ctx)
	}
	return err
}

func (locker *Locker) lock(ctx context.Context, isForced bool) error {
	newLockerState := LockState{
		ID:              locker.State.ID,
		AcquisitionTime: time.Now(),
		IsForced:        isForced,
		User:            locker.State.User,
	}
	buf, err := json.Marshal(newLockerState)
	if err != nil {
		return err
	}

	// Repeating a create that does not overwrite is safe because the lock file carries our unique
	// ID: if an attempt that failed with a transient error did create the file, the retry fails
	// with fs.ErrExist and assertLockHeld below tells whether the lock is ours. That holds for a
	// connection cut mid-response ("unexpected EOF") too, so this uses the wider check meant for
	// reads. Forced writes overwrite and are idempotent.
	mode := []filer.WriteMode{filer.CreateParentDirectories}
	if isForced {
		mode = append(mode, filer.OverwriteIfExists)
	}
	write := func() error {
		_, err := retry.With(ctx, retry.IsTransientRead, func() (struct{}, error) {
			err := locker.lockFiler.Write(ctx, LockFileName, bytes.NewReader(buf), mode...)
			if err != nil && !errors.Is(err, fs.ErrExist) && !errors.Is(err, fs.ErrPermission) && !errors.Is(err, fs.ErrNotExist) {
				locker.createUncertain = true
			}
			return struct{}{}, err
		})
		return err
	}

	err = write()
	if errors.Is(err, fs.ErrExist) && locker.removeStaleLock(ctx) {
		err = write()
	}

	switch {
	case err == nil || errors.Is(err, fs.ErrExist):
		// If the lock file already exists, [assertLockHeld] returns a more descriptive
		// error that names the current holder of the lock.
		if err := locker.assertLockHeld(ctx); err != nil {
			return err
		}
	case retry.IsTransientRead(ctx, err):
		// The last attempt may have been applied even though its response was lost. Read the
		// lock back so that such a write does not leave behind a lock nobody releases. This
		// covers forced locks too.
		activeLockState, readErr := locker.GetActiveLockState(ctx)
		if readErr != nil {
			// No lock, or the read failed too: the write error says more.
			return err
		}
		if holderErr := locker.checkHolder(activeLockState); holderErr != nil {
			if isForced {
				// The holder error suggests --force-lock, which is already in use.
				return err
			}
			return holderErr
		}
	default:
		return err
	}

	locker.State = &newLockerState
	locker.Active = true
	return nil
}

func (locker *Locker) Unlock(ctx context.Context, opts ...UnlockOption) error {
	if !locker.Active {
		return errors.New("unlock called when lock is not held")
	}
	if locker.createUncertain {
		// Recorded even if the unlock fails: a lock with our ID left behind is stale either way.
		defer locker.recordReleased(ctx)
	}

	// if allowLockFileNotExist is set, do not throw an error if the lock file does
	// not exist. This is helpful when destroying a bundle in which case the lock
	// file will be deleted before we have a chance to unlock
	if _, err := locker.filer.Stat(ctx, LockFileName); errors.Is(err, fs.ErrNotExist) && slices.Contains(opts, AllowLockFileNotExist) {
		locker.Active = false
		return nil
	}

	err := locker.assertLockHeld(ctx)
	if err != nil {
		return fmt.Errorf("unlock called when lock is not held: %w", err)
	}
	err = locker.filer.Delete(ctx, LockFileName)
	if err != nil {
		return err
	}
	locker.Active = false
	return nil
}

// maxReleasedLocks bounds the list in ReleasedLocksPath. A late write lands within minutes, so
// only the most recent entries matter.
const maxReleasedLocks = 20

func (locker *Locker) readReleased(ctx context.Context) []uuid.UUID {
	if locker.ReleasedLocksPath == "" {
		return nil
	}
	data, err := os.ReadFile(locker.ReleasedLocksPath)
	var ids []uuid.UUID
	if err == nil {
		err = json.Unmarshal(data, &ids)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "Failed to read released deploy locks: %v", err)
	}
	return ids
}

// recordReleased adds our ID to ReleasedLocksPath. Failing to record it only means that a late
// write, if one lands, needs --force-lock to clear, so errors are logged and not returned.
func (locker *Locker) recordReleased(ctx context.Context) {
	if locker.ReleasedLocksPath == "" {
		return
	}
	ids := append(locker.readReleased(ctx), locker.State.ID)
	if len(ids) > maxReleasedLocks {
		ids = ids[len(ids)-maxReleasedLocks:]
	}
	data, err := json.Marshal(ids)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(locker.ReleasedLocksPath), 0o700)
	}
	if err == nil {
		// Write to a temporary file and rename it, so that a concurrent reader never sees a
		// partial list.
		tmp := locker.ReleasedLocksPath + "." + uuid.New().String() + ".tmp"
		err = os.WriteFile(tmp, data, 0o600)
		if err == nil {
			err = os.Rename(tmp, locker.ReleasedLocksPath)
		}
		if err != nil {
			_ = os.Remove(tmp)
		}
	}
	if err != nil {
		log.Warnf(ctx, "Failed to record released deploy lock: %v", err)
	}
}

// removeStaleLock deletes the remote lock if it carries an ID from ReleasedLocksPath: a write
// that timed out in an earlier command was applied after that command released the lock. It
// reports whether it deleted the lock. A --force-lock from another client between the read and
// the delete loses its lock here; forced locks already give up exclusive access.
func (locker *Locker) removeStaleLock(ctx context.Context) bool {
	released := locker.readReleased(ctx)
	if len(released) == 0 {
		return false
	}
	activeLockState, err := locker.GetActiveLockState(ctx)
	if err != nil || !slices.Contains(released, activeLockState.ID) {
		return false
	}
	log.Infof(ctx, "Removing stale deploy lock %s acquired at %v: it was released before", activeLockState.ID, activeLockState.AcquisitionTime)
	return locker.filer.Delete(ctx, LockFileName) == nil
}

func CreateLocker(user, targetDir string, w *databricks.WorkspaceClient) (*Locker, error) {
	filer, err := filer.NewWorkspaceFilesClient(w, targetDir)
	if err != nil {
		return nil, err
	}

	locker := &Locker{
		filer:     retry.Filer(filer),
		lockFiler: filer,

		TargetDir: targetDir,
		Active:    false,
		State: &LockState{
			ID:   uuid.New(),
			User: user,
		},
	}

	return locker, nil
}
