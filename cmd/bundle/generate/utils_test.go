package generate

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	_ "go.uber.org/mock/mockgen/model"
)

func TestDownloadNotebookAndReplaceTaskPath(t *testing.T) {
	task := &jobs.Task{
		NotebookTask: &jobs.NotebookTask{
			NotebookPath: "/Shared/Notebook/notebook",
		},
	}

	w := mocks.NewWorkspaceClient(t, &config.Config{})
	mockW := mocks.GetMockWorkspaceAPI(w)
	mockW.On(
		"GetStatusByPath",
		context.Background(),
		"/Shared/Notebook/notebook",
	).Return(&workspace.ObjectInfo{
		ObjectType: "NOTEBOOK",
		Language:   "PYTHON",
	}, nil)

	mockW.On(
		"Download",
		context.Background(),
		"/Shared/Notebook/notebook",
	).Return(io.NopCloser(strings.NewReader("Hello from notebook")), nil)

	outputDir := t.TempDir()
	err := downloadNotebookAndReplaceTaskPath(context.Background(), task, w, outputDir)
	assert.NoError(t, err)

	// test that the notebook was downloaded and in outputDir
	_, err = os.Stat(filepath.Join(outputDir, "notebook.py"))
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(outputDir, "notebook.py"))
	assert.NoError(t, err)

	assert.Equal(t, "Hello from notebook", string(data))
}
