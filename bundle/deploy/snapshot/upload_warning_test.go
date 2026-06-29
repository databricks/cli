package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUploader struct{ path string }

func (m *mockUploader) Upload(_ context.Context, _, _ string, _ []ACLEntry, _ []byte) (*SnapshotInfo, error) {
	return &SnapshotInfo{Path: m.path}, nil
}

func makeBundle(t *testing.T, nFiles int) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	for i := range nFiles {
		p := filepath.Join(dir, fmt.Sprintf("f%d.py", i))
		require.NoError(t, os.WriteFile(p, []byte("x"), 0o644))
	}
	root := vfs.MustNew(dir)
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRoot:       root,
		WorktreeRoot:   root,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{UserName: "test@example.test"},
				},
			},
		},
	}
	return b
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	return logdiag.InitContext(cmdio.MockDiscard(t.Context()))
}

func TestUploadWarnsAboveFileLimit(t *testing.T) {
	b := makeBundle(t, fileLimitWarning+1)
	m := &snapshotUpload{uploader: &mockUploader{path: "/snapshots/test"}}

	diags := m.Apply(testContext(t), b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, fmt.Sprintf("%d files", fileLimitWarning+1))
	assert.Equal(t, "/snapshots/test", b.Config.Workspace.SnapshotPath)
}

func TestUploadNoWarningBelowFileLimit(t *testing.T) {
	b := makeBundle(t, 5)
	m := &snapshotUpload{uploader: &mockUploader{path: "/snapshots/test"}}

	diags := m.Apply(testContext(t), b)

	assert.True(t, diags.HasError() == false && len(diags) == 0, "expected no diagnostics")
}
