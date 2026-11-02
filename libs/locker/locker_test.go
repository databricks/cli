package locker

import (
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"uuid"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/retry"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eofFiler fails the first `failures` writes with the error net/http returns when the connection
// is cut mid-response. The write numbered appliedOn (1-based, 0 for none) still writes the file
// before failing, like a request the server applied before the connection was cut.
type eofFiler struct {
	filer.Filer
	failures  int
	appliedOn int
	writes    int
}

func (f *eofFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	f.writes++
	if f.writes > f.failures {
		return f.Filer.Write(ctx, path, reader, mode...)
	}
	if f.writes == f.appliedOn {
		if err := f.Filer.Write(ctx, path, reader, mode...); err != nil {
			return err
		}
	}
	return &url.Error{Op: "Post", URL: "https://example.com/" + path, Err: io.ErrUnexpectedEOF}
}

func newTestLocker(t *testing.T, f filer.Filer) *Locker {
	return &Locker{
		filer:     f,
		TargetDir: t.TempDir(),
		State:     &LockState{ID: uuid.New(), User: "tester"},
	}
}

func TestLockRetriesUnexpectedEOF(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")

	tests := []struct {
		name      string
		failures  int
		appliedOn int
	}{
		// The retry creates the lock.
		{name: "not applied", failures: 1, appliedOn: 0},
		// The retry fails with "already exists" and the read-back finds our lock.
		{name: "applied", failures: 1, appliedOn: 1},
		// Retries run out, but the last attempt was applied: the read-back finds our lock.
		{name: "last attempt applied", failures: 3, appliedOn: 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			local, err := filer.NewLocalClient(t.TempDir())
			require.NoError(t, err)
			f := &eofFiler{Filer: local, failures: tc.failures, appliedOn: tc.appliedOn}
			locker := newTestLocker(t, f)

			require.NoError(t, locker.Lock(ctx, false))
			assert.True(t, locker.Active)
			assert.Equal(t, min(tc.failures+1, 3), f.writes)
		})
	}
}

func TestLockFailsWhenEveryAttemptHitsUnexpectedEOF(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	f := &eofFiler{Filer: local, failures: 3}
	locker := newTestLocker(t, f)

	err = locker.Lock(ctx, false)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.False(t, locker.Active)
	assert.Equal(t, 3, f.writes)
}

// failingReadFiler fails every Read, standing in for a lock-file read that timed out.
type failingReadFiler struct {
	filer.Filer
	err error
}

func (f failingReadFiler) Read(context.Context, string) (io.ReadCloser, error) {
	return nil, f.err
}

func TestUnlockReportsMissingLockAsNotHeld(t *testing.T) {
	l := &Locker{
		filer:  filer.NewFakeFiler(map[string]fakefs.FileInfo{}),
		Active: true,
		State:  &LockState{ID: uuid.New()},
	}
	err := l.Unlock(t.Context())
	assert.ErrorContains(t, err, "unlock called when lock is not held: no active lock on target dir")
}

func TestUnlockReportsReadFailureAsCheckFailure(t *testing.T) {
	readErr := errors.New(`Get "https://example.com/api/2.0/workspace/export": request timed out after 1m30s of inactivity`)
	l := &Locker{
		filer:  failingReadFiler{Filer: filer.NewFakeFiler(map[string]fakefs.FileInfo{LockFileName: {}}), err: readErr},
		Active: true,
		State:  &LockState{ID: uuid.New()},
	}
	err := l.Unlock(t.Context())
	assert.ErrorIs(t, err, readErr)
	assert.ErrorContains(t, err, "failed to check the deploy lock before releasing it")
	assert.NotContains(t, err.Error(), "not held")
}

func TestUnlockReportsLockDeletedBeforeDownloadAsNotHeld(t *testing.T) {
	// The workspace filer's stat succeeded, then the export found nothing.
	downloadErr := &apierr.APIError{StatusCode: 404, ErrorCode: "RESOURCE_DOES_NOT_EXIST", Message: "Path does not exist"}
	l := &Locker{
		filer:  failingReadFiler{Filer: filer.NewFakeFiler(map[string]fakefs.FileInfo{LockFileName: {}}), err: downloadErr},
		Active: true,
		State:  &LockState{ID: uuid.New()},
	}
	err := l.Unlock(t.Context())
	assert.ErrorContains(t, err, "unlock called when lock is not held: no active lock on target dir")
}
