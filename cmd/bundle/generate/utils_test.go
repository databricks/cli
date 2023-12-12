package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	mocks_client "github.com/databricks/cli/internal/mocks/sdk/client"
	mock_workspace "github.com/databricks/cli/internal/mocks/sdk/service/workspace"
	"github.com/databricks/databricks-sdk-go/client"
	databrickscfg "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	_ "go.uber.org/mock/mockgen/model"
)

func TestDownloadNotebookAndReplaceTaskPath(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
		},
	}

	task := &jobs.Task{
		NotebookTask: &jobs.NotebookTask{
			NotebookPath: "/Shared/Notebook/notebook",
		},
	}

	w := b.WorkspaceClient()
	_, err := client.New(&databrickscfg.Config{})
	assert.NoError(t, err)

	ws := mock_workspace.NewWorkspaceService(t)
	ws.Mock.On(("GetStatus"), context.Background(), workspace.GetStatusRequest{Path: "/Shared/Notebook/notebook"}).Return(&workspace.ObjectInfo{
		ObjectType: "NOTEBOOK",
		Language:   "PYTHON",
	}, nil)

	client := mocks_client.NewDatabricksClientInterface(t)
	ws.Mock.On(("Client")).Return(client)

	client.Mock.On(("Do"),
		context.Background(),
		"GET",
		"/api/2.0/workspace/export",
		map[string]string{"Content-Type": "application/json"},
		map[string]interface{}{"direct_download": true, "path": "/Shared/Notebook/notebook"},
		mock.MatchedBy(func(buf *bytes.Buffer) bool {
			buf.WriteString("Hello from notebook")
			return true
		}),
	).Return(nil)

	w.Workspace.WithImpl(ws)
	outputDir := t.TempDir()
	err = downloadNotebookAndReplaceTaskPath(context.Background(), task, w, outputDir)
	assert.NoError(t, err)

	// test that the notebook was downloaded and in outputDir
	_, err = os.Stat(filepath.Join(outputDir, "notebook.py"))
	assert.NoError(t, err)

}
