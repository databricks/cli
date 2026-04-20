// Package lock provides deployment lock primitives for ucm. It is forked
// from bundle/deploy/lock to avoid importing bundle internals; the underlying
// wire protocol (a JSON lock file at <stateDir>/deploy.lock) is deliberately
// identical to libs/locker so ucm and bundle can later share the same state
// root if needed. Forking rather than wrapping libs/locker lets tests inject
// an arbitrary filer.Filer without requiring a live workspace client.
package lock

import (
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/google/uuid"
)

// LockFileName is the on-the-wire name of the deploy lock file under the
// state directory. Kept identical to libs/locker.LockFileName so ucm and
// bundle lockers are mutually exclusive on a shared state root.
const LockFileName = "deploy.lock"

// Goal identifies the operation that holds the lock. Release uses it to
// decide how strict to be about missing lock files — a destroy that already
// deleted the state dir is expected to find no lock.
type Goal string

const (
	GoalDeploy  = Goal("deploy")
	GoalDestroy = Goal("destroy")
	GoalBind    = Goal("bind")
	GoalUnbind  = Goal("unbind")
)

// State is the on-the-wire lock record: who holds the lock, when they took
// it, and whether it was forcefully acquired. Field names match
// libs/locker.LockState so ucm and bundle can deserialize each other's locks.
type State struct {
	ID              uuid.UUID `json:"ID"`
	AcquisitionTime time.Time `json:"AcquisitionTime"`
	IsForced        bool      `json:"IsForced"`
	User            string    `json:"User"`
}

// Locker enables exclusive access to a remote state directory for one client.
// Multiple clients race to create LockFileName under targetDir; the first
// write wins. Lock holders may force-release a stale lock with the force
// flag, at the cost of the exclusivity guarantee.
type Locker struct {
	filer filer.Filer

	// TargetDir is the scope of the lock (state directory root).
	TargetDir string
	// Active is true when this locker holds the lock. Forced acquisitions
	// may break exclusivity for other holders.
	Active bool
	// LocalState is this client's record of the lock; uploaded to TargetDir
	// on Acquire.
	LocalState *State
}

// NewLocker creates a Locker backed by a workspace-files filer rooted at
// targetDir. user is embedded into the lock record so contending clients can
// see who currently holds it.
func NewLocker(user, targetDir string, w *databricks.WorkspaceClient) (*Locker, error) {
	f, err := filer.NewWorkspaceFilesClient(w, targetDir)
	if err != nil {
		return nil, err
	}
	return newLocker(user, targetDir, f), nil
}

// NewLockerWithFiler lets callers inject an arbitrary filer.Filer (e.g. a
// local-disk filer in tests, or a future s3/adls/gcs state filer).
// Production workspace-files callers should use NewLocker.
func NewLockerWithFiler(user, targetDir string, f filer.Filer) *Locker {
	return newLocker(user, targetDir, f)
}

func newLocker(user, targetDir string, f filer.Filer) *Locker {
	return &Locker{
		filer:     f,
		TargetDir: targetDir,
		Active:    false,
		LocalState: &State{
			ID:   uuid.New(),
			User: user,
		},
	}
}
