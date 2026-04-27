package deploy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
)

// Push mirrors the local state cache into the remote StateFiler. It refuses
// to overwrite a remote that has advanced past the Seq we observed locally,
// and writes atomically-ish by writing ucm-state.json last so a partial push
// leaves the remote tfstate readable but clearly un-acked.
//
// Push assumes the local ucm-state.json has already been advanced by the
// StateUpdate mutator. The expected shape is Pull → StateUpdate → Push: a
// caller pushing without a prior bump would write a remote with the same
// Seq, defeating the conflict-detection contract.
//
// On a fresh local (no ucm-state.json under LocalStateDir), Push returns an
// error — pushing without a pull means the caller has no baseline Seq to
// reason about.
func Push(ctx context.Context, u *ucm.Ucm, b Backend) error {
	if u == nil {
		return errors.New("ucm state: Push called with nil Ucm")
	}
	if b.StateFiler == nil || b.LockFiler == nil {
		return errors.New("ucm state: Push requires StateFiler and LockFiler in Backend")
	}

	l := newLocker(b, ".")
	if err := l.Acquire(ctx, b.ForceLock); err != nil {
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

	if err := writeRemote(ctx, b.StateFiler, LocalTfStatePath(u), local); err != nil {
		return err
	}

	log.Infof(ctx, "ucm state: pushed state (seq %d) for target %s", local.Seq, u.Config.Ucm.Target)
	return nil
}

// assertRemoteNotAhead fails with ErrStaleState when the remote Seq is at
// or past the local (post-bump) Seq. A healthy Push has local strictly
// greater than remote because StateUpdate already advanced local; equality
// means a peer beat us to this Seq slot, and greater-than means a peer is
// strictly ahead. Missing remote state means a first Push and always
// succeeds.
func assertRemoteNotAhead(ctx context.Context, f filer.StateFiler, local *State) error {
	remote, err := readRemoteUcmState(ctx, f)
	if err != nil {
		return fmt.Errorf("ucm state: inspect remote: %w", err)
	}
	if remote == nil {
		return nil
	}
	if remote.Seq >= local.Seq {
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
