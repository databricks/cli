package acceptance_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workspaceTmpDir(ctx context.Context, t *testing.T) (*databricks.WorkspaceClient, filer.Filer, string) {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	currentUser, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	timestamp := time.Now().Format("2006-01-02T15:04:05Z")
	tmpDir := fmt.Sprintf(
		"/Workspace/Users/%s/acceptance/%s/%s",
		currentUser.UserName,
		timestamp,
		uuid.New().String(),
	)

	t.Cleanup(func() {
		err := w.Workspace.Delete(ctx, workspace.Delete{
			Path:      tmpDir,
			Recursive: true,
		})
		assert.NoError(t, err)
	})

	err = w.Workspace.MkdirsByPath(ctx, tmpDir)
	require.NoError(t, err)

	f, err := filer.NewWorkspaceFilesClient(w, tmpDir)
	require.NoError(t, err)

	return w, f, tmpDir
}
