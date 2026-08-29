package sync

import (
	"testing"

	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
)

func TestApplyPutDryRunSkipsLocalRead(t *testing.T) {
	s := &Sync{
		SyncOptions: &SyncOptions{
			LocalRoot: vfs.MustNew(t.TempDir()),
			DryRun:    true,
		},
		notifier: &NopNotifier{},
	}

	// The missing file must not fail a dry run; the nil filer guarantees no write is attempted.
	err := s.applyPut(t.Context(), "missing.txt")
	assert.NoError(t, err)
}
