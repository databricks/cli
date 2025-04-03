package python_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
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

func TestRunPythonTaskWorkspace(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	testutil.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")

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

func TestRunPythonTaskDBFS(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	testutil.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")

	runPythonTasks(t, prepareDBFSFiles(t), testOpts{
		name:                    "Python tasks from DBFS",
		includeNotebookTasks:    false,
		includeSparkPythonTasks: true,
		includeWheelTasks:       true,
	})
}

func TestRunPythonTaskRepo(t *testing.T) {
	// TODO: remove RUN_PYTHON_TASKS_TEST when ready to be executed as part of nightly
	testutil.GetEnvOrSkipTest(t, "RUN_PYTHON_TASKS_TEST")

	runPythonTasks(t, prepareRepoFiles(t), testOpts{
		name:                    "Python tasks from Repo",
		includeNotebookTasks:    true,
		includeSparkPythonTasks: true,
		includeWheelTasks:       false,
	})
}

func runPythonTasks(t *testing.T, tw *testFiles, opts testOpts) {
	w := tw.w

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	var tasks []jobs.SubmitTask
	if opts.includeNotebookTasks {
		tasks = append(tasks, GenerateNotebookTasks(tw.pyNotebookPath, sparkVersions, nodeTypeId)...)
	}

	if opts.includeSparkPythonTasks {
		tasks = append(tasks, GenerateSparkPythonTasks(tw.sparkPythonPath, sparkVersions, nodeTypeId)...)
	}

	if opts.includeWheelTasks {
		versions := sparkVersions
		if len(opts.wheelSparkVersions) > 0 {
			versions = opts.wheelSparkVersions
		}
		tasks = append(tasks, GenerateWheelTasks(tw.wheelPath, versions, nodeTypeId)...)
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
	var err error
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	baseDir := acc.TemporaryWorkspaceDir(wt, "python-tasks-")

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
	var err error
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	baseDir := acc.TemporaryDbfsDir(wt, "python-tasks-")

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
		sparkPythonPath: "dbfs:" + path.Join(baseDir, "spark.py"),
		wheelPath:       "dbfs:" + path.Join(baseDir, "my_test_code-0.0.1-py3-none-any.whl"),
	}
}

func prepareRepoFiles(t *testing.T) *testFiles {
	_, wt := acc.WorkspaceTest(t)
	w := wt.W

	baseDir := acc.TemporaryRepo(wt, "https://github.com/databricks/cli")

	packagePath := "internal/python/testdata"
	return &testFiles{
		w:               w,
		pyNotebookPath:  path.Join(baseDir, packagePath, "test"),
		sparkPythonPath: path.Join(baseDir, packagePath, "spark.py"),
		wheelPath:       path.Join(baseDir, packagePath, "my_test_code-0.0.1-py3-none-any.whl"),
	}
}

func GenerateNotebookTasks(notebookPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	var tasks []jobs.SubmitTask
	for i := range versions {
		task := jobs.SubmitTask{
			TaskKey: "notebook_" + strings.ReplaceAll(versions[i], ".", "_"),
			NotebookTask: &jobs.NotebookTask{
				NotebookPath: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func GenerateSparkPythonTasks(notebookPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	var tasks []jobs.SubmitTask
	for i := range versions {
		task := jobs.SubmitTask{
			TaskKey: "spark_" + strings.ReplaceAll(versions[i], ".", "_"),
			SparkPythonTask: &jobs.SparkPythonTask{
				PythonFile: notebookPath,
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}

func GenerateWheelTasks(wheelPath string, versions []string, nodeTypeId string) []jobs.SubmitTask {
	var tasks []jobs.SubmitTask
	for i := range versions {
		task := jobs.SubmitTask{
			TaskKey: "whl_" + strings.ReplaceAll(versions[i], ".", "_"),
			PythonWheelTask: &jobs.PythonWheelTask{
				PackageName: "my_test_code",
				EntryPoint:  "run",
			},
			NewCluster: &compute.ClusterSpec{
				SparkVersion:     versions[i],
				NumWorkers:       1,
				NodeTypeId:       nodeTypeId,
				DataSecurityMode: compute.DataSecurityModeUserIsolation,
			},
			Libraries: []compute.Library{
				{Whl: wheelPath},
			},
		}
		tasks = append(tasks, task)
	}

	return tasks
}
