package locker

import (
	"context"
	"errors"
	"io"
	"testing"
	"uuid"

	"github.com/databricks/cli/libs/fakefs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
)

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
