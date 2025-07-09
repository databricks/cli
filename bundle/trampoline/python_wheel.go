package trampoline

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

const NOTEBOOK_TEMPLATE = `# Databricks notebook source
%python
{{range .Libraries}}
%pip install --force-reinstall {{.Whl}}
{{end}}

dbutils.library.restartPython()

try:
	from importlib import metadata
except ImportError: # for Python<3.8
	import subprocess
	import sys

	subprocess.check_call([sys.executable, "-m", "pip", "install", "importlib-metadata"])
	import importlib_metadata as metadata

from contextlib import redirect_stdout
import io
import sys
import json

params = []
try:
	python_params = dbutils.widgets.get("__python_params")
	if python_params:
		params = json.loads(python_params)
except Exception as e:
	print(e)

sys.argv = [{{.Params}}]

if params:
	sys.argv = [sys.argv[0]] + params

entry = [ep for ep in metadata.distribution("{{.Task.PackageName}}").entry_points if ep.name == "{{.Task.EntryPoint}}"]

f = io.StringIO()
with redirect_stdout(f):
	if entry:
		entry[0].load()()
	else:
		raise ImportError("Entry point '{{.Task.EntryPoint}}' not found")
s = f.getvalue()
dbutils.notebook.exit(s)
`

type transformWheelTask struct{}

func (transformWheelTask) Name() string {
	return "TransformWheelTask"
}

// This mutator takes the wheel task and transforms it into notebook
// which installs uploaded wheels using %pip and then calling corresponding
// entry point.
func TransformWheelTask() bundle.Mutator {
	return transformWheelTask{}
}

func (transformWheelTask) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	isEnabled := b.Config.Experimental != nil && b.Config.Experimental.PythonWheelWrapper
	b.Metrics.AddBoolValue(metrics.ExperimentalPythonWheelWrapperIsSet, isEnabled)
	if !isEnabled {
		return nil
	}

	bundle.ApplyContext(ctx, b, NewTrampoline(
		"python_wheel",
		&pythonTrampoline{},
		NOTEBOOK_TEMPLATE,
	))

	return nil
}

type pythonTrampoline struct{}

func (t *pythonTrampoline) CleanUp(task *jobs.Task) error {
	task.PythonWheelTask = nil

	var nonWheelLibraries []compute.Library
	for _, l := range task.Libraries {
		if l.Whl == "" {
			nonWheelLibraries = append(nonWheelLibraries, l)
		}
	}
	task.Libraries = nonWheelLibraries

	return nil
}

func (t *pythonTrampoline) GetTasks(b *bundle.Bundle) []TaskWithJobKey {
	r := b.Config.Resources
	var result []TaskWithJobKey
	for k := range b.Config.Resources.Jobs {
		tasks := r.Jobs[k].JobSettings.Tasks
		for i := range tasks {
			task := &tasks[i]

			// Keep only Python wheel tasks with workspace libraries referenced.
			// At this point of moment we don't have local paths in Libraries sections anymore
			// Local paths have been replaced with the remote when the artifacts where uploaded
			// in artifacts.UploadAll mutator.
			if task.PythonWheelTask == nil || !needsTrampoline(*task) {
				continue
			}

			result = append(result, TaskWithJobKey{
				JobKey: k,
				Task:   task,
			})
		}
	}
	return result
}

func needsTrampoline(task jobs.Task) bool {
	return libraries.IsTaskWithWorkspaceLibraries(task)
}

func (t *pythonTrampoline) GetTemplateData(task *jobs.Task) (map[string]any, error) {
	params, err := t.generateParameters(task.PythonWheelTask)
	var whlLibraries []compute.Library
	for _, l := range task.Libraries {
		if l.Whl != "" {
			whlLibraries = append(whlLibraries, l)
		}
	}

	if err != nil {
		return nil, err
	}

	data := map[string]any{
		"Libraries": whlLibraries,
		"Params":    params,
		"Task":      task.PythonWheelTask,
	}

	return data, nil
}

func (t *pythonTrampoline) generateParameters(task *jobs.PythonWheelTask) (string, error) {
	if task.Parameters != nil && task.NamedParameters != nil {
		return "", errors.New("not allowed to pass both paramaters and named_parameters")
	}
	params := append([]string{task.PackageName}, task.Parameters...)
	for k, v := range task.NamedParameters {
		params = append(params, fmt.Sprintf("%s=%s", k, v))
	}

	for i := range params {
		params[i] = strconv.Quote(params[i])
	}
	return strings.Join(params, ", "), nil
}
