// Package deploy owns the remote-state lifecycle for ucm: the pull/push/ops
// glue that sits between the pluggable StateFiler (U1) and the deploy Locker
// (U2). It is forked from bundle/deploy; keeping the shapes close eases
// future cross-checks without introducing a bundle/** import.
package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"time"

	libsfiler "github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/google/uuid"
)

// Remote and local filenames for the state artifacts. StateFileName lives
// next to TfStateFileName in the remote state dir; Seq-based conflict
// detection reads UcmStateFileName only.
const (
	// UcmStateFileName is the ucm-specific sidecar that tracks Seq/Version
	// alongside the opaque terraform.tfstate blob.
	UcmStateFileName = "ucm-state.json"

	// TfStateFileName is the terraform-managed state blob that ucm mirrors
	// locally so the terraform wrapper can drive plan/apply offline.
	TfStateFileName = "terraform.tfstate"

	// LocalCacheDir is the root under the ucm project directory used to
	// mirror remote state. The per-target sub-directory is appended by
	// LocalStateDir.
	LocalCacheDir = ".databricks/ucm"

	// StateVersion is bumped on incompatible changes to the on-wire State
	// shape. Pull treats a remote Version greater than this as an error so
	// older CLIs refuse to overwrite newer state blobs.
	StateVersion = 1
)

// State is the ucm-side sidecar stored as ucm-state.json in the remote state
// directory. It carries just enough metadata to detect stale overwrites and
// identify the CLI that produced the blob. The opaque terraform.tfstate lives
// separately so terraform tooling can read it without understanding Seq.
type State struct {
	// Version is bumped on incompatible changes to this struct. A remote
	// Version greater than StateVersion fails the pull.
	Version int `json:"version"`

	// Seq is incremented on every successful Push. Push refuses to overwrite
	// a remote whose Seq is greater than the Seq we saw at Pull time.
	Seq int `json:"seq"`

	// ID uniquely identifies the sequence of deployments for this target.
	// Rotating it starts a fresh Seq chain (e.g. after --force).
	ID uuid.UUID `json:"id"`

	// CliVersion is the ucm/cli version that wrote the blob. Informational
	// only — Seq is the source of truth for conflict detection.
	CliVersion string `json:"cli_version,omitempty"`

	// Timestamp is when the state was produced. Informational only.
	Timestamp time.Time `json:"timestamp"`
}

// ErrStaleState is returned by Push when the remote Seq is greater than the
// Seq we observed at Pull time. Callers can unwrap via errors.As to surface
// the two Seq values to the user.
type ErrStaleState struct {
	// LocalSeq is the Seq we believe we are advancing from.
	LocalSeq int
	// RemoteSeq is the Seq currently on the remote, which is ahead of
	// LocalSeq.
	RemoteSeq int
}

// Error formats the conflict for human consumption.
func (e *ErrStaleState) Error() string {
	return fmt.Sprintf("ucm state: local state is stale (local seq %d < remote seq %d); pull before pushing", e.LocalSeq, e.RemoteSeq)
}

// Backend bundles the remote-IO dependencies needed by Pull and Push. It is
// supplied by the caller so tests can inject local-disk filers without a live
// workspace client. U6 will build the production Backend from
// ucm.state.backend config; until then callers compose it directly.
type Backend struct {
	// StateFiler is where terraform.tfstate and ucm-state.json are read
	// from and written to. Typically rooted at the per-target workspace
	// state path.
	StateFiler filer.StateFiler

	// LockFiler is where the Locker writes deploy.lock. In the v1 workspace
	// backend StateFiler and LockFiler point at the same remote dir; the
	// split exists because the Locker API consumes libs/filer.Filer rather
	// than the ucm StateFiler.
	LockFiler libsfiler.Filer

	// User is embedded into the lock record so contending clients can see
	// who currently holds the lock. Empty strings are allowed for tests.
	User string

	// ForceLock tells Pull/Push to override an existing deploy lock instead
	// of failing with ErrLockHeld. Set by the --force-lock flag on
	// plan/deploy/destroy; mirrors bundle.Deployment.Lock.Force.
	ForceLock bool
}

// loadState reads a State from r.
func loadState(r io.Reader) (*State, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("ucm state: parse: %w", err)
	}
	return &s, nil
}

// readRemoteUcmState loads ucm-state.json via the StateFiler, returning
// (nil, nil) when the file is absent (first-run case).
func readRemoteUcmState(ctx context.Context, f filer.StateFiler) (*State, error) {
	rc, err := f.Read(ctx, UcmStateFileName)
	if err != nil {
		if errors.Is(err, filer.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	defer rc.Close()
	return loadState(rc)
}

// validateCompatibility fails when the remote State was written by a newer
// CLI with an incompatible schema version.
func validateCompatibility(s *State) error {
	if s.Version > StateVersion {
		return fmt.Errorf("ucm state: remote version %d > supported %d; upgrade the CLI", s.Version, StateVersion)
	}
	return nil
}

// LocalTfStatePath returns the canonical local path for the terraform state
// blob: <LocalStateDir>/terraform/terraform.tfstate. This is where terraform
// natively writes its state (its working directory), so treating the nested
// path as canonical means Pull/Push/summary all observe the file terraform
// actually produces. Matches bundle.Bundle.StateFilenameTerraform's local-path
// return — ucm and DAB deliberately share the convention.
func LocalTfStatePath(u *ucm.Ucm) string {
	return filepath.Join(LocalStateDir(u), "terraform", TfStateFileName)
}

// newLocker constructs a Locker bound to the backend's LockFiler. Kept
// package-private because the only callers are Pull and Push.
func newLocker(b Backend, targetDir string) *lock.Locker {
	return lock.NewLockerWithFiler(b.User, targetDir, b.LockFiler)
}
