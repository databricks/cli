package deployer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/databricks/bricks/utilities"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

// Locker object enables exclusive access to TargetDir's scope for a client. This
// enables multiple clients to deploy to the same scope (ie TargetDir) in an atomic
// manner
//
// Here are some of the details of the locking protocol used here:
//
// 1. Potentially multiple clients race to create a deploy.lock file in
//    TargetDir/.bundle directory with unique ID. The deploy.lock file
//    is a json file containing the State from the locker
//
// 2. Clients read the remote deploy.lock file and if it's ID matches, the client
//    assumes it has the lock on TargetDir. The client is now free to read/write code
//    asserts and deploy databricks assets scoped under TargetDir
//
// 3. To sidestep clients failing to relinquish a lock during a failed deploy attempt
//    we allow clients to forcefully acquire a lock on TargetDir. However forcefully acquired
//    locks come with the following caveats:
//
//       a.  a forcefully acquired lock does not guarentee exclusive access to
//           TargetDir's scope
//       b.  forcefully acquiring a lock(s) on TargetDir can break the assumption
//           of exclusive access that other clients with non forcefully acquired
//           locks might have
type Locker struct {
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

// don't need to hold lock on TargetDir to read locker state
func GetActiveLockState(ctx context.Context, wsc *databricks.WorkspaceClient, path string) (*LockState, error) {
	bytes, err := utilities.GetRawJsonFileContent(ctx, wsc, path)
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
func (locker *Locker) assertLockHeld(ctx context.Context, wsc *databricks.WorkspaceClient) error {
	activeLockState, err := GetActiveLockState(ctx, wsc, locker.RemotePath())
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
func (locker *Locker) PutFile(ctx context.Context, wsc *databricks.WorkspaceClient, pathToFile string, content []byte) error {
	if !locker.Active {
		return fmt.Errorf("failed to put file. deploy lock not held")
	}
	apiClient, err := client.New(wsc.Config)
	if err != nil {
		return err
	}
	apiPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		strings.TrimLeft(pathToFile, "/"))

	err = apiClient.Do(ctx, http.MethodPost, apiPath, bytes.NewReader(content), nil)
	if err != nil {
		// retry after creating parent dirs
		err = wsc.Workspace.MkdirsByPath(ctx, path.Dir(pathToFile))
		if err != nil {
			return fmt.Errorf("could not mkdir to put file: %s", err)
		}
		err = apiClient.Do(ctx, http.MethodPost, apiPath, bytes.NewReader(content), nil)
	}
	return err
}

func (locker *Locker) GetRawJsonFileContent(ctx context.Context, wsc *databricks.WorkspaceClient, path string) ([]byte, error) {
	if !locker.Active {
		return nil, fmt.Errorf("failed to get file. deploy lock not held")
	}
	return utilities.GetRawJsonFileContent(ctx, wsc, path)
}

func (locker *Locker) Lock(ctx context.Context, wsc *databricks.WorkspaceClient, isForced bool) error {
	newLockerState := LockState{
		ID:              locker.State.ID,
		AcquisitionTime: time.Now(),
		IsForced:        isForced,
		User:            locker.State.User,
	}
	bytes, err := json.Marshal(newLockerState)
	if err != nil {
		return err
	}
	err = utilities.WriteFile(ctx, wsc, locker.RemotePath(), bytes, isForced)
	if err != nil && !strings.Contains(err.Error(), fmt.Sprintf("%s already exists", locker.RemotePath())) {
		return err
	}
	err = locker.assertLockHeld(ctx, wsc)
	if err != nil {
		return err
	}

	locker.State = &newLockerState
	locker.Active = true
	return nil
}

func (locker *Locker) Unlock(ctx context.Context, wsc *databricks.WorkspaceClient) error {
	if !locker.Active {
		return fmt.Errorf("unlock called when lock is not held")
	}
	err := locker.assertLockHeld(ctx, wsc)
	if err != nil {
		return fmt.Errorf("unlock called when lock is not held: %s", err)
	}
	err = wsc.Workspace.Delete(ctx,
		workspace.Delete{
			Path:      locker.RemotePath(),
			Recursive: false,
		},
	)
	if err != nil {
		return err
	}
	locker.Active = false
	return nil
}

func (locker *Locker) RemotePath() string {
	return path.Join(locker.TargetDir, ".bundle/deploy.lock")
}

func CreateLocker(user string, targetDir string) *Locker {
	return &Locker{
		TargetDir: targetDir,
		Active:    false,
		State: &LockState{
			ID:   uuid.New(),
			User: user,
		},
	}
}
