package locker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"slices"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/atomicfile"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/service/workspace"
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

	// deleteLockOnce deletes the lock file in a single request, without the SDK's own retries.
	// nil uses filer's unretried Delete, which suits filers that do not retry.
	deleteLockOnce func(ctx context.Context) error

	// ReleasedLocksPath is a local file listing the IDs of lockers that released the lock or gave
	// up on taking it. A lock that carries one of these IDs is stale: a lock write that timed out
	// can still be applied by the server after its command finished. The file also records the
	// deletes of stale locks that may still be applied. Empty disables the list.
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
	if err != nil {
		// We do not hold the lock, but a write that timed out may still create it later.
		locker.recordReleased(ctx, time.Time{})
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

	err = locker.writeLock(ctx, buf, isForced)
	if errors.Is(err, fs.ErrExist) && locker.removeStaleLock(ctx) {
		err = locker.writeLock(ctx, buf, isForced)
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

// writeLock writes the lock file with buf, overwriting an existing one only if isForced.
func (locker *Locker) writeLock(ctx context.Context, buf []byte, isForced bool) error {
	if isForced {
		// The retrying filer already retries overwrite writes, on the errors [retry.IsTransientRead]
		// covers; if the last attempt still fails, lock reads the lock back.
		return locker.filer.Write(ctx, LockFileName, bytes.NewReader(buf), filer.CreateParentDirectories, filer.OverwriteIfExists)
	}
	// The retrying filer does not repeat writes that do not overwrite. Repeating this one is
	// safe because the lock file carries our unique ID: if an attempt that failed with a
	// transient error did create the file, the retry fails with fs.ErrExist and
	// assertLockHeld tells whether the lock is ours. That holds for a connection cut
	// mid-response ("unexpected EOF") too, so this uses the wider check meant for reads.
	_, err := retry.With(ctx, retry.IsTransientRead, func() (struct{}, error) {
		return struct{}{}, locker.filer.Write(ctx, LockFileName, bytes.NewReader(buf), filer.CreateParentDirectories)
	})
	return err
}

func (locker *Locker) Unlock(ctx context.Context, opts ...UnlockOption) error {
	if !locker.Active {
		return errors.New("unlock called when lock is not held")
	}
	// Recorded even if the unlock fails: a lock with our ID left behind is stale either way.
	// deleteFailed is set when the delete failed and may still be applied.
	var deleteFailed time.Time
	defer func() { locker.recordReleased(ctx, deleteFailed) }()

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
		// The next command must not replace our lock while this delete may still land on its lock.
		// The window starts now, not when the delete was sent: the retrying filer and the SDK may
		// have been resending it for minutes.
		deleteFailed = timeNow()
		return err
	}
	locker.Active = false
	return nil
}

// maxReleasedLocks bounds the list in ReleasedLocksPath. A late write lands within minutes, so
// only the most recent entries matter.
const maxReleasedLocks = 20

// pendingDeleteWindow is how long after it failed a delete may still be applied by the server.
// Until it passes, no command sends another delete for that lock: if the first delete landed after
// a second one had removed the lock and a new lock was created, it would remove the new lock. Late
// requests seen so far landed within a few minutes.
const pendingDeleteWindow = 15 * time.Minute

// timeNow is replaced in tests.
var timeNow = time.Now

// releasedLocks is the content of ReleasedLocksPath.
type releasedLocks struct {
	// Released lists the IDs of lockers that released the lock or gave up on taking it, oldest
	// first.
	Released []uuid.UUID `json:"released"`

	// PendingDeletes maps the ID of a lock to the time a delete for it failed (or was sent, while
	// it is in flight), for deletes that may still be applied: a stale-lock delete, or the delete
	// of a failed Unlock.
	PendingDeletes map[uuid.UUID]time.Time `json:"pending_deletes,omitempty"`
}

func (locker *Locker) readReleased(ctx context.Context) releasedLocks {
	var released releasedLocks
	if locker.ReleasedLocksPath == "" {
		return released
	}
	data, err := os.ReadFile(locker.ReleasedLocksPath)
	if err == nil {
		err = json.Unmarshal(data, &released)
	}
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Warnf(ctx, "Failed to read released deploy locks: %v", err)
	}
	return released
}

// writeReleased writes released to ReleasedLocksPath, dropping old entries.
func (locker *Locker) writeReleased(released releasedLocks) error {
	if len(released.Released) > maxReleasedLocks {
		released.Released = released.Released[len(released.Released)-maxReleasedLocks:]
	}
	for id, sent := range released.PendingDeletes {
		if timeNow().Sub(sent) >= pendingDeleteWindow || !slices.Contains(released.Released, id) {
			delete(released.PendingDeletes, id)
		}
	}
	data, err := json.Marshal(released)
	if err != nil {
		return err
	}
	return atomicfile.Write(locker.ReleasedLocksPath, data, 0o600, atomicfile.MkDir(0o700))
}

// recordReleased adds our ID to ReleasedLocksPath, with a pending delete of our lock that failed
// at deleteFailed unless it is zero. Failing to record it only means that a late write, if one lands, needs
// --force-lock to clear, so errors are logged and not returned.
func (locker *Locker) recordReleased(ctx context.Context, deleteFailed time.Time) {
	if locker.ReleasedLocksPath == "" {
		return
	}
	id := locker.State.ID
	released := locker.readReleased(ctx)
	if slices.Contains(released.Released, id) && deleteFailed.IsZero() {
		return
	}
	if !slices.Contains(released.Released, id) {
		released.Released = append(released.Released, id)
	}
	if !deleteFailed.IsZero() {
		if released.PendingDeletes == nil {
			released.PendingDeletes = map[uuid.UUID]time.Time{}
		}
		released.PendingDeletes[id] = deleteFailed
	}
	if err := locker.writeReleased(released); err != nil {
		log.Warnf(ctx, "Failed to record released deploy lock: %v", err)
	}
}

// errLockNotStale stops removeStaleLock when the lock no longer carries a released ID.
var errLockNotStale = errors.New("deploy lock is not stale")

// errStaleLockDeletePending makes removeStaleLock read the lock again after a failed delete.
var errStaleLockDeletePending = errors.New("stale deploy lock delete may still be applied")

// deletePendingError is a failed delete that may still be applied. It reads as the delete's error
// and matches errStaleLockDeletePending too.
type deletePendingError struct{ error }

func (e deletePendingError) Unwrap() []error { return []error{e.error, errStaleLockDeletePending} }

// removeStaleLock deletes the remote lock if it carries an ID from ReleasedLocksPath: a write
// that timed out in an earlier command was applied after that command released the lock. It
// reports whether the stale lock is gone. A lock taken between the read and the delete that
// follows it is lost: one from --force-lock, which already gives up exclusive access, or one from
// a second CLI process in this checkout that removed the same stale lock first.
func (locker *Locker) removeStaleLock(ctx context.Context) bool {
	released := locker.readReleased(ctx)
	if len(released.Released) == 0 {
		return false
	}
	// The delete is sent at most once. A delete that failed, for example with a timeout, may
	// still be applied later; a second delete, or one replayed by the retrying filer or the SDK,
	// could then remove a lock that another deployer, or this Lock, has taken since. After a
	// failed delete only reads follow, until the lock is gone or no longer stale. The delete is
	// recorded before it is sent, so that later commands do not send another one within
	// pendingDeleteWindow either.
	deleteOnce := locker.deleteLockOnce
	if deleteOnce == nil {
		deleteFiler := locker.filer
		if u, ok := deleteFiler.(interface{ Unwrap() filer.Filer }); ok {
			deleteFiler = u.Unwrap()
		}
		deleteOnce = func(ctx context.Context) error { return deleteFiler.Delete(ctx, LockFileName) }
	}
	sentDelete := false
	_, err := retry.With(ctx, isRetriableStaleLockError, func() (struct{}, error) {
		activeLockState, err := locker.GetActiveLockState(ctx)
		if err != nil {
			return struct{}{}, err
		}
		id := activeLockState.ID
		if !slices.Contains(released.Released, id) {
			return struct{}{}, errLockNotStale
		}
		if sentDelete {
			return struct{}{}, errStaleLockDeletePending
		}
		if sent, ok := released.PendingDeletes[id]; ok && timeNow().Sub(sent) < pendingDeleteWindow {
			log.Infof(ctx, "Not removing stale deploy lock %s: a delete that failed at %v may still be applied", id, sent)
			return struct{}{}, errLockNotStale
		}
		if released.PendingDeletes == nil {
			released.PendingDeletes = map[uuid.UUID]time.Time{}
		}
		released.PendingDeletes[id] = timeNow()
		if err := locker.writeReleased(released); err != nil {
			log.Warnf(ctx, "Not removing stale deploy lock %s: failed to record the delete: %v", id, err)
			return struct{}{}, errLockNotStale
		}
		sentDelete = true
		log.Infof(ctx, "Removing stale deploy lock %s acquired at %v: it was released before", id, activeLockState.AcquisitionTime)
		err = deleteOnce(ctx)
		if err == nil {
			delete(released.PendingDeletes, id)
			if err := locker.writeReleased(released); err != nil {
				log.Warnf(ctx, "Failed to record released deploy locks: %v", err)
			}
		} else {
			// The window starts when the delete returned.
			released.PendingDeletes[id] = timeNow()
			if err := locker.writeReleased(released); err != nil {
				log.Warnf(ctx, "Failed to record released deploy locks: %v", err)
			}
			if isRetriableStaleLockError(ctx, err) {
				return struct{}{}, deletePendingError{err}
			}
		}
		return struct{}{}, err
	})
	return err == nil || errors.Is(err, fs.ErrNotExist)
}

func CreateLocker(user, targetDir string, w *databricks.WorkspaceClient) (*Locker, error) {
	filer, err := filer.NewWorkspaceFilesClient(w, targetDir)
	if err != nil {
		return nil, err
	}

	deleteLockOnce, err := newDeleteLockOnce(w, path.Join(targetDir, LockFileName))
	if err != nil {
		return nil, err
	}

	locker := &Locker{
		filer:          retry.Filer(filer),
		deleteLockOnce: deleteLockOnce,

		TargetDir: targetDir,
		Active:    false,
		State: &LockState{
			ID:   uuid.New(),
			User: user,
		},
	}

	return locker, nil
}

// isRetriableStaleLockError reports whether removeStaleLock should read the lock again. A retry
// only reads after the first delete, so it is safe after any error that may be transient: the ones
// [retry.IsTransient] covers, the ones the SDK would retry itself (deleteLockOnce turns SDK retries
// off), and transport errors.
func isRetriableStaleLockError(ctx context.Context, err error) bool {
	if errors.Is(err, errStaleLockDeletePending) || retry.IsTransient(ctx, err) {
		return true
	}
	if _, ok := errors.AsType[*url.Error](err); ok {
		return true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.IsRetriable(ctx)
}

// newDeleteLockOnce returns a function that deletes lockPath with one request. The SDK retries a
// 504, 503 or 429 on its own, and a replayed delete can remove a lock that another deployer took
// after the first delete was applied. The SDK retries some connection errors whatever
// ErrorRetriable says, so a minimal retry window is needed to stop after the first attempt.
func newDeleteLockOnce(w *databricks.WorkspaceClient, lockPath string) (func(ctx context.Context) error, error) {
	clientCfg, err := config.HTTPClientConfigFromConfig(w.Config)
	if err != nil {
		return nil, err
	}
	clientCfg.ErrorRetriable = func(context.Context, error) bool { return false }
	clientCfg.TransientErrors = nil
	clientCfg.RetryTimeout = time.Nanosecond
	apiClient, err := client.NewWithClient(w.Config, httpclient.NewApiClient(clientCfg))
	if err != nil {
		return nil, err
	}
	return func(ctx context.Context) error {
		err := apiClient.Do(ctx, http.MethodPost, "/api/2.0/workspace/delete", auth.WorkspaceIDHeaders(w.Config), nil, workspace.Delete{Path: lockPath}, nil)
		if errors.Is(err, apierr.ErrNotFound) {
			return fs.ErrNotExist
		}
		return err
	}, nil
}
