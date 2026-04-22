package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/google/uuid"
)

// Push mirrors the local state cache into the remote StateFiler. It bumps
// Seq, refuses to overwrite a remote that has advanced past the Seq we
// observed locally, and writes atomically-ish by writing ucm-state.json last
// so a partial push leaves the remote tfstate readable but clearly un-acked.
//
// On a fresh local (no ucm-state.json under LocalStateDir), Push returns an
// error — the expected shape is Pull → mutate → Push, and pushing without a
// pull means the caller has no baseline Seq to reason about.
func Push(ctx context.Context, u *ucm.Ucm, b Backend) error {
	if u == nil {
		return errors.New("ucm state: Push called with nil Ucm")
	}
	if b.StateFiler == nil || b.LockFiler == nil {
		return errors.New("ucm state: Push requires StateFiler and LockFiler in Backend")
	}

	l := newLocker(b, ".")
	if err := l.Acquire(ctx, false); err != nil {
		return fmt.Errorf("ucm state: acquire lock: %w", err)
	}
	defer releaseBestEffort(ctx, l, lock.GoalDeploy)

	localDir := LocalStateDir(u)
	local, err := readLocalState(localDir)
	if err != nil {
		return fmt.Errorf("ucm state: read local %s: %w", UcmStateFileName, err)
	}

	if err := assertRemoteNotAhead(ctx, b.StateFiler, local); err != nil {
		return err
	}

	// Bump Seq before serialising so the on-remote record matches the
	// intent of this Push. A crash after writeRemote but before the
	// bookkeeping below leaves the remote ahead of local; the next Pull
	// catches up.
	next := *local
	next.Version = StateVersion
	next.Seq = local.Seq + 1
	if next.ID == uuid.Nil {
		next.ID = uuid.New()
	}
	next.Timestamp = time.Now().UTC()

	if err := writeRemote(ctx, b.StateFiler, LocalTfStatePath(u), &next); err != nil {
		return err
	}

	// Mirror the bumped Seq into the local cache so the next Push starts
	// from an accurate baseline without requiring an intervening Pull.
	if err := writeLocalState(localDir, &next); err != nil {
		return fmt.Errorf("ucm state: refresh local %s: %w", UcmStateFileName, err)
	}
	log.Infof(ctx, "ucm state: pushed state (seq %d -> %d) for target %s", local.Seq, next.Seq, u.Config.Ucm.Target)
	return nil
}

// assertRemoteNotAhead fails with ErrStaleState when the remote Seq exceeds
// the local Seq. Missing remote state counts as Seq=-1 for comparison so a
// first Push always succeeds.
func assertRemoteNotAhead(ctx context.Context, f filer.StateFiler, local *State) error {
	remote, err := readRemoteUcmState(ctx, f)
	if err != nil {
		return fmt.Errorf("ucm state: inspect remote: %w", err)
	}
	if remote == nil {
		return nil
	}
	if remote.Seq > local.Seq {
		return &ErrStaleState{LocalSeq: local.Seq, RemoteSeq: remote.Seq}
	}
	return nil
}

// writeRemote uploads the local terraform.tfstate (if any) first, then
// ucm-state.json. ucm-state.json is written last so a crash between the two
// leaves the remote in a shape the next Pull can still interpret as
// "remote ahead of us, need to advance" rather than "blank slate".
//
// tfStatePath is the absolute local path that terraform wrote its state to
// (canonically LocalTfStatePath(u)). A missing file there is treated as a
// benign "nothing to upload" — the first Push before any terraform apply
// runs hits this path.
func writeRemote(ctx context.Context, f filer.StateFiler, tfStatePath string, next *State) error {
	if data, err := os.ReadFile(tfStatePath); err == nil {
		if err := f.Write(ctx, TfStateFileName, bytes.NewReader(data), filer.WriteModeOverwrite|filer.WriteModeCreateParents); err != nil {
			return fmt.Errorf("ucm state: write remote %s: %w", TfStateFileName, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("ucm state: read local %s: %w", TfStateFileName, err)
	}

	blob, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	if err := f.Write(ctx, UcmStateFileName, bytes.NewReader(blob), filer.WriteModeOverwrite|filer.WriteModeCreateParents); err != nil {
		return fmt.Errorf("ucm state: write remote %s: %w", UcmStateFileName, err)
	}
	return nil
}
