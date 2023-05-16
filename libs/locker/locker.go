package locker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/google/uuid"
)

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
//     a.  a forcefully acquired lock does not guarentee exclusive access to
//     TargetDir's scope
//     b.  forcefully acquiring a lock(s) on TargetDir can break the assumption
//     of exclusive access that other clients with non forcefully acquired
//     locks might have
type Locker struct {
	filer filer.Filer

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
	reader, err := locker.filer.Read(ctx, locker.RemotePath())
	if err != nil {
		return nil, err
	}

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
	if err != nil && strings.Contains(err.Error(), "File not found.") {
		return fmt.Errorf("no active lock on target dir: %s", err)
	}
	if err != nil {
		return err
	}
	if activeLockState.ID != locker.State.ID && !activeLockState.IsForced {
		return fmt.Errorf("deploy lock acquired by %s at %v. Use --force to override", activeLockState.User, activeLockState.AcquisitionTime)
	}
	if activeLockState.ID != locker.State.ID && activeLockState.IsForced {
		return fmt.Errorf("deploy lock force acquired by %s at %v. Use --force to override", activeLockState.User, activeLockState.AcquisitionTime)
	}
	return nil
}

// idempotent function since overwrite is set to true
func (locker *Locker) PutFile(ctx context.Context, pathToFile string, content []byte) error {
	if !locker.Active {
		return fmt.Errorf("failed to put file. deploy lock not held")
	}
	return locker.filer.Write(ctx, pathToFile, bytes.NewReader(content), filer.OverwriteIfExists, filer.CreateParentDirectories)
}

func (locker *Locker) GetRawJsonFileContent(ctx context.Context, path string) ([]byte, error) {
	if !locker.Active {
		return nil, fmt.Errorf("failed to get file. deploy lock not held")
	}
	reader, err := locker.filer.Read(ctx, path)
	if err != nil {
		return nil, err
	}
	return io.ReadAll(reader)
}

func (locker *Locker) Lock(ctx context.Context, isForced bool) error {
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

	var modes = []filer.WriteMode{
		// Always create parent directory if it doesn't yet exist.
		filer.CreateParentDirectories,
	}

	// Only overwrite lock file if `isForced`.
	if isForced {
		modes = append(modes, filer.OverwriteIfExists)
	}

	err = locker.filer.Write(ctx, locker.RemotePath(), bytes.NewReader(buf), modes...)
	if err != nil {
		// If the write failed because the lock file already exists, don't return
		// the error and instead fall through to [assertLockHeld] below.
		// This function will return a more descriptive error message that includes
		// details about the current holder of the lock.
		if !errors.As(err, &filer.FileAlreadyExistsError{}) {
			return err
		}
	}

	err = locker.assertLockHeld(ctx)
	if err != nil {
		return err
	}

	locker.State = &newLockerState
	locker.Active = true
	return nil
}

func (locker *Locker) Unlock(ctx context.Context) error {
	if !locker.Active {
		return fmt.Errorf("unlock called when lock is not held")
	}
	err := locker.assertLockHeld(ctx)
	if err != nil {
		return fmt.Errorf("unlock called when lock is not held: %s", err)
	}
	err = locker.filer.Delete(ctx, locker.RemotePath())
	if err != nil {
		return err
	}
	locker.Active = false
	return nil
}

func (locker *Locker) RemotePath() string {
	// Note: remote paths are scoped to `targetDir`. Also see [CreateLocker].
	return "deploy.lock"
}

func CreateLocker(user string, targetDir string, w *databricks.WorkspaceClient) (*Locker, error) {
	filer, err := filer.NewWorkspaceFilesClient(w, targetDir)
	if err != nil {
		return nil, err
	}

	locker := &Locker{
		filer: filer,

		TargetDir: targetDir,
		Active:    false,
		State: &LockState{
			ID:   uuid.New(),
			User: user,
		},
	}

	return locker, nil
}
