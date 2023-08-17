package python

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const NOTEBOOK_TEMPLATE = `# Databricks notebook source
%python
{{range .Libraries}}
%pip install --force-reinstall {{.Whl}}
{{end}}

from contextlib import redirect_stdout
import io
import sys
sys.argv = [{{.Params}}]

import pkg_resources
_func = pkg_resources.load_entry_point("{{.Task.PackageName}}", "console_scripts", "{{.Task.EntryPoint}}")

f = io.StringIO()
with redirect_stdout(f):
	_func()
s = f.getvalue()
dbutils.notebook.exit(s)
`

// This mutator takes the wheel task and transforms it into notebook
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

func (m *transform) Apply(ctx context.Context, b *bundle.Bundle) error {
	wheelTasks := libraries.FindAllWheelTasks(b)
	for _, wheelTask := range wheelTasks {
		err := generateNotebookTrampoline(b, wheelTask)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateNotebookTrampoline(b *bundle.Bundle, wheelTask *jobs.Task) error {
	taskDefinition := wheelTask.PythonWheelTask
	libraries := wheelTask.Libraries

	wheelTask.PythonWheelTask = nil
	wheelTask.Libraries = nil

	filename, err := generateNotebookWrapper(b, taskDefinition, libraries)
	if err != nil {
		return err
	}

	internalDir, err := getInternalDir(b)
	if err != nil {
		return err
	}

	internalDirRel, err := filepath.Rel(b.Config.Path, internalDir)
	if err != nil {
		return err
	}

	parts := []string{b.Config.Workspace.FilesPath}
	parts = append(parts, strings.Split(internalDirRel, string(os.PathSeparator))...)
	parts = append(parts, filename)

	wheelTask.NotebookTask = &jobs.NotebookTask{
		NotebookPath: path.Join(parts...),
	}

	return nil
}

func getInternalDir(b *bundle.Bundle) (string, error) {
	cacheDir, err := b.CacheDir()
	if err != nil {
		return "", err
	}
	internalDir := filepath.Join(cacheDir, ".internal")
	return internalDir, nil
}

func generateNotebookWrapper(b *bundle.Bundle, task *jobs.PythonWheelTask, libraries []compute.Library) (string, error) {
	internalDir, err := getInternalDir(b)
	if err != nil {
		return "", err
	}

	notebookName := fmt.Sprintf("notebook_%s_%s", task.PackageName, task.EntryPoint)
	path := filepath.Join(internalDir, notebookName+".py")

	err = os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return "", err
	}

	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	params, err := generateParameters(task)
	if err != nil {
		return "", err
	}

	data := map[string]any{
		"Libraries": libraries,
		"Params":    params,
		"Task":      task,
	}

	t, err := template.New("notebook").Parse(NOTEBOOK_TEMPLATE)
	if err != nil {
		return "", err
	}
	return notebookName, t.Execute(f, data)
}

func generateParameters(task *jobs.PythonWheelTask) (string, error) {
	if task.Parameters != nil && task.NamedParameters != nil {
		return "", fmt.Errorf("not allowed to pass both paramaters and named_parameters")
	}
	params := append([]string{"python"}, task.Parameters...)
	for k, v := range task.NamedParameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}
	for i := range params {
		params[i] = `"` + params[i] + `"`
	}
	return strings.Join(params, ", "), nil
}
