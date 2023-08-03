package python

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// This mutator takes the wheel task and trasnforms it into notebook
// which installs uploaded wheels using %pip and then calling corresponding
// entry point.
func TransformWheelTask() bundle.Mutator {
	return &transform{}
}

type transform struct {
}

func (m *transform) Name() string {
	return "python.TransformWheelTask"
}

const INSTALL_WHEEL_CODE = `%%pip install --force-reinstall %s`

const NOTEBOOK_CODE = `
%%python
%s

from contextlib import redirect_stdout
import io
import sys
sys.argv = [%s]

import pkg_resources
_func = pkg_resources.load_entry_point("%s", "console_scripts", "%s")

f = io.StringIO()
with redirect_stdout(f):
	_func()
s = f.getvalue()
dbutils.notebook.exit(s)
`

func (m *transform) Apply(ctx context.Context, b *bundle.Bundle) error {
	// TODO: do the transformaton only for DBR < 13.1 and (maybe?) existing clusters
	wheelTasks := libraries.FindAllWheelTasks(b)
	for _, wheelTask := range wheelTasks {
		taskDefinition := wheelTask.PythonWheelTask
		libraries := wheelTask.Libraries

		wheelTask.PythonWheelTask = nil
		wheelTask.Libraries = nil

		path, err := generateNotebookWrapper(taskDefinition, libraries)
		if err != nil {
			return err
		}

		remotePath, err := artifacts.UploadNotebook(context.Background(), path, b)
		if err != nil {
			return err
		}

		os.Remove(path)

		wheelTask.NotebookTask = &jobs.NotebookTask{
			NotebookPath: remotePath,
		}
	}
	return nil
}

func generateNotebookWrapper(task *jobs.PythonWheelTask, libraries []compute.Library) (string, error) {
	pipInstall := ""
	for _, lib := range libraries {
		pipInstall = pipInstall + "\n" + fmt.Sprintf(INSTALL_WHEEL_CODE, lib.Whl)
	}
	content := fmt.Sprintf(NOTEBOOK_CODE, pipInstall, generateParameters(task), task.PackageName, task.EntryPoint)

	tmpDir := os.TempDir()
	filename := fmt.Sprintf("notebook_%s_%s.ipynb", task.PackageName, task.EntryPoint)
	path := filepath.Join(tmpDir, filename)

	err := os.WriteFile(path, bytes.NewBufferString(content).Bytes(), 0644)
	return path, err
}

func generateParameters(task *jobs.PythonWheelTask) string {
	params := append([]string{"python"}, task.Parameters...)
	for k, v := range task.NamedParameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	for i := range params {
		params[i] = `"` + params[i] + `"`
	}
	return strings.Join(params, ", ")
}
