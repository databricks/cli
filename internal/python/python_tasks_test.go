package python

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

const PY_CONTENT = `# Databricks notebook source
import os
import sys
import json

out = {"PYTHONPATH": sys.path, "CWD": os.getcwd()}
json_object = json.dumps(out, indent = 4)
dbutils.notebook.exit(json_object)
`

const SPARK_PY_CONTENT = `
import os
import sys
import json

out = {"PYTHONPATH": sys.path, "CWD": os.getcwd()}
json_object = json.dumps(out, indent = 4)
print(json_object)
`

type testOutput struct {
	PythonPath []string `json:"PYTHONPATH"`
	Cwd        string   `json:"CWD"`
}

type testFiles struct {
	w               *databricks.WorkspaceClient
	pyNotebookPath  string
	sparkPythonPath string
	wheelPath       string
}

type testOpts struct {
	name                    string
	includeNotebookTasks    bool
	includeSparkPythonTasks bool
	includeWheelTasks       bool
	wheelSparkVersions      []string
}

var sparkVersions = []string{
	"11.3.x-scala2.12",
	"12.2.x-scala2.12",
	"13.0.x-scala2.12",
	"13.1.x-scala2.12",
	"13.2.x-scala2.12",
	"13.3.x-scala2.12",
	"14.0.x-scala2.12",
	"14.1.x-scala2.12",
}

func TestAccRunPythonTaskWorkspace(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	internal.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")
	internal.GetEnvOrSkipTest(t, "CLOUD_ENV")

	unsupportedSparkVersionsForWheel := []string{
		"11.3.x-scala2.12",
		"12.2.x-scala2.12",
		"13.0.x-scala2.12",
	}
	runPythonTasks(t, prepareWorkspaceFiles(t), testOpts{
		name:                    "Python tasks from WSFS",
		includeNotebookTasks:    true,
		includeSparkPythonTasks: true,
		includeWheelTasks:       true,
		wheelSparkVersions: slices.DeleteFunc(slices.Clone(sparkVersions), func(s string) bool {
			return slices.Contains(unsupportedSparkVersionsForWheel, s)
		}),
	})
}

func TestAccRunPythonTaskDBFS(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	internal.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")
	internal.GetEnvOrSkipTest(t, "CLOUD_ENV")

	runPythonTasks(t, prepareDBFSFiles(t), testOpts{
		name:                    "Python tasks from DBFS",
		includeNotebookTasks:    false,
		includeSparkPythonTasks: true,
		includeWheelTasks:       true,
	})
}

func TestAccRunPythonTaskRepo(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	internal.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")
	internal.GetEnvOrSkipTest(t, "CLOUD_ENV")

	runPythonTasks(t, prepareRepoFiles(t), testOpts{
		name:                    "Python tasks from Repo",
		includeNotebookTasks:    true,
		includeSparkPythonTasks: true,
		includeWheelTasks:       false,
	})
}

func runPythonTasks(t *testing.T, tw *testFiles, opts testOpts) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	w := tw.w

	nodeTypeId := internal.GetNodeTypeId(env)
	tasks := make([]jobs.SubmitTask, 0)
	if opts.includeNotebookTasks {
		tasks = append(tasks, internal.GenerateNotebookTasks(tw.pyNotebookPath, sparkVersions, nodeTypeId)...)
	}

	if opts.includeSparkPythonTasks {
		tasks = append(tasks, internal.GenerateSparkPythonTasks(tw.sparkPythonPath, sparkVersions, nodeTypeId)...)
	}

	if opts.includeWheelTasks {
		versions := sparkVersions
		if len(opts.wheelSparkVersions) > 0 {
			versions = opts.wheelSparkVersions
		}
		tasks = append(tasks, internal.GenerateWheelTasks(tw.wheelPath, versions, nodeTypeId)...)
	}

	ctx := context.Background()
	run, err := w.Jobs.Submit(ctx, jobs.SubmitRun{
		RunName: opts.name,
		Tasks:   tasks,
	})
	require.NoError(t, err)

	_, err = w.Jobs.WaitGetRunJobTerminatedOrSkipped(ctx, run.RunId, time.Hour, nil)
	require.NoError(t, err)

	output, err := output.GetJobOutput(ctx, w, run.RunId)
	require.NoError(t, err)

	result := make(map[string]testOutput, 0)
	for _, out := range output.TaskOutputs {
		s, err := out.Output.String()
		require.NoError(t, err)

		tOut := testOutput{}
		err = json.Unmarshal([]byte(s), &tOut)
		if err != nil {
			continue
		}
		result[out.TaskKey] = tOut
	}

	out, err := json.MarshalIndent(result, "", "    ")
	require.NoError(t, err)

	t.Log("==== Run output ====")
	t.Log(string(out))
}

func prepareWorkspaceFiles(t *testing.T) *testFiles {
	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	baseDir := internal.TemporaryWorkspaceDir(t, w)
	pyNotebookPath := path.Join(baseDir, "test.py")

	err = w.Workspace.Import(ctx, workspace.Import{
		Path:      pyNotebookPath,
		Overwrite: true,
		Language:  workspace.LanguagePython,
		Format:    workspace.ImportFormatSource,
		Content:   base64.StdEncoding.EncodeToString([]byte(PY_CONTENT)),
	})
	require.NoError(t, err)

	sparkPythonPath := path.Join(baseDir, "spark.py")
	err = w.Workspace.Import(ctx, workspace.Import{
		Path:      sparkPythonPath,
		Overwrite: true,
		Format:    workspace.ImportFormatAuto,
		Content:   base64.StdEncoding.EncodeToString([]byte(SPARK_PY_CONTENT)),
	})
	require.NoError(t, err)

	raw, err := os.ReadFile("./testdata/my_test_code-0.0.1-py3-none-any.whl")
	require.NoError(t, err)

	wheelPath := path.Join(baseDir, "my_test_code-0.0.1-py3-none-any.whl")
	err = w.Workspace.Import(ctx, workspace.Import{
		Path:      path.Join(baseDir, "my_test_code-0.0.1-py3-none-any.whl"),
		Overwrite: true,
		Format:    workspace.ImportFormatAuto,
		Content:   base64.StdEncoding.EncodeToString(raw),
	})
	require.NoError(t, err)

	return &testFiles{
		w:               w,
		pyNotebookPath:  pyNotebookPath,
		sparkPythonPath: sparkPythonPath,
		wheelPath:       path.Join("/Workspace", wheelPath),
	}
}

func prepareDBFSFiles(t *testing.T) *testFiles {
	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	baseDir := internal.TemporaryDbfsDir(t, w)
	f, err := filer.NewDbfsClient(w, baseDir)
	require.NoError(t, err)

	err = f.Write(ctx, "test.py", strings.NewReader(PY_CONTENT))
	require.NoError(t, err)

	err = f.Write(ctx, "spark.py", strings.NewReader(SPARK_PY_CONTENT))
	require.NoError(t, err)

	raw, err := os.ReadFile("./testdata/my_test_code-0.0.1-py3-none-any.whl")
	require.NoError(t, err)

	err = f.Write(ctx, "my_test_code-0.0.1-py3-none-any.whl", bytes.NewReader(raw))
	require.NoError(t, err)

	return &testFiles{
		w:               w,
		pyNotebookPath:  path.Join(baseDir, "test.py"),
		sparkPythonPath: fmt.Sprintf("dbfs:%s", path.Join(baseDir, "spark.py")),
		wheelPath:       fmt.Sprintf("dbfs:%s", path.Join(baseDir, "my_test_code-0.0.1-py3-none-any.whl")),
	}
}

func prepareRepoFiles(t *testing.T) *testFiles {
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	repo := internal.TemporaryRepo(t, w)
	return &testFiles{
		w:               w,
		pyNotebookPath:  path.Join(repo, "test"),
		sparkPythonPath: path.Join(repo, "spark.py"),
		wheelPath:       path.Join(repo, "my_test_code-0.0.1-py3-none-any.whl"),
	}
}
