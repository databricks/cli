package snapshot

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestClient(t *testing.T) *databricks.WorkspaceClient {
	t.Helper()
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: math.MaxInt,
	})
	require.NoError(t, err)
	return client
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
			// The SyncDefaultPath mutator sets this to ["."] during initialize;
			// set it here since these tests bypass the mutator pipeline. Empty
			// sync paths select no files.
			Sync: config.Sync{Paths: []string{"."}},
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
	b.SetWorkpaceClient(setupTestClient(t))
	m := &snapshotUpload{}

	diags := m.Apply(testContext(t), b)

	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, fmt.Sprintf("%d files", fileLimitWarning+1))
}

func TestUploadNoWarningBelowFileLimit(t *testing.T) {
	b := makeBundle(t, 5)
	b.SetWorkpaceClient(setupTestClient(t))
	m := &snapshotUpload{}

	diags := m.Apply(testContext(t), b)

	assert.True(t, diags.HasError() == false && len(diags) == 0, "expected no diagnostics")
}
