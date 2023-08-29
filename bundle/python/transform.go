package python

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/libraries"
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
	return mutator.NewTrampoline(
		"python_wheel",
		getTasks,
		generateTemplateData,
		cleanUpTask,
		NOTEBOOK_TEMPLATE,
	)
}

func getTasks(b *bundle.Bundle) []*jobs.Task {
	return libraries.FindAllWheelTasks(b)
}

func generateTemplateData(task *jobs.Task) (map[string]any, error) {
	params, err := generateParameters(task.PythonWheelTask)
	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"Libraries": task.Libraries,
		"Params":    params,
		"Task":      task.PythonWheelTask,
	}

	return data, nil
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
		params[i] = strconv.Quote(params[i])
	}
	return strings.Join(params, ", "), nil
}

func cleanUpTask(task *jobs.Task) {
	task.PythonWheelTask = nil
	task.Libraries = nil
}
