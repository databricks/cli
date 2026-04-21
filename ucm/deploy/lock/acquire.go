package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
)

// ErrLockHeld is returned by Acquire when another client already holds the
// lock. It carries the contending lock state so callers (or the human) can
// decide whether --force is warranted.
type ErrLockHeld struct {
	Holder          string
	AcquisitionTime time.Time
	IsForced        bool
}

// Error implements error. The message format is deliberately close to the
// bundle/libs/locker message so CI logs and user muscle memory carry over.
func (e *ErrLockHeld) Error() string {
	if e.IsForced {
		return fmt.Sprintf("deploy lock force acquired by %s at %v. Use --force-lock to override", e.Holder, e.AcquisitionTime)
	}
	return fmt.Sprintf("deploy lock acquired by %s at %v. Use --force-lock to override", e.Holder, e.AcquisitionTime)
}

// GetActiveLockState returns the lock record currently written at TargetDir,
// irrespective of whether this locker holds it. Returns fs.ErrNotExist if no
// lock is currently held.
func (l *Locker) GetActiveLockState(ctx context.Context) (*State, error) {
	reader, err := l.filer.Read(ctx, LockFileName)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	remote := State{}
	if err := json.Unmarshal(data, &remote); err != nil {
		return nil, err
	}
	return &remote, nil
}

// assertLockHeld verifies that the lock file under TargetDir matches this
// locker's ID. Returns ErrLockHeld if another client is the current holder.
func (l *Locker) assertLockHeld(ctx context.Context) error {
	active, err := l.GetActiveLockState(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("no active lock on target dir: %w", err)
	}
	if err != nil {
		return err
	}
	if active.ID != l.LocalState.ID {
		return &ErrLockHeld{
			Holder:          active.User,
			AcquisitionTime: active.AcquisitionTime,
			IsForced:        active.IsForced,
		}
	}
	return nil
}

// Acquire writes a lock record under TargetDir. If force is true it
// overwrites any existing record; otherwise it races via an atomic create
// and returns *ErrLockHeld on contention.
func (l *Locker) Acquire(ctx context.Context, force bool) error {
	log.Infof(ctx, "Acquiring deployment lock (force: %v)", force)

	newState := State{
		ID:              l.LocalState.ID,
		AcquisitionTime: time.Now(),
		IsForced:        force,
		User:            l.LocalState.User,
	}
	buf, err := json.Marshal(newState)
	if err != nil {
		return err
	}

	modes := []filer.WriteMode{filer.CreateParentDirectories}
	if force {
		modes = append(modes, filer.OverwriteIfExists)
	}

	err = l.filer.Write(ctx, LockFileName, bytes.NewReader(buf), modes...)
	if err != nil {
		// If the write failed because the lock file already exists, fall
		// through to assertLockHeld so the caller gets the ErrLockHeld with
		// the contending holder's identity rather than a bare fs.ErrExist.
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
	}

	if err := l.assertLockHeld(ctx); err != nil {
		return err
	}

	l.LocalState = &newState
	l.Active = true
	return nil
}
