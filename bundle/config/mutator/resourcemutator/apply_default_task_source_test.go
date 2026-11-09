package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultTaskSourceGitJob(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"git": {
						JobSettings: jobs.JobSettings{
							GitSource: &jobs.GitSource{GitUrl: "https://example.invalid/repo", GitProvider: "gitHub", GitBranch: "main"},
							Tasks: []jobs.Task{
								{TaskKey: "py", SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "src/main.py"}},
								{TaskKey: "nb_explicit", NotebookTask: &jobs.NotebookTask{NotebookPath: "nb", Source: jobs.SourceWorkspace}},
								{TaskKey: "foreach", ForEachTask: &jobs.ForEachTask{Task: jobs.Task{
									TaskKey:         "inner",
									SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "src/inner.py"},
								}}},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyDefaultTaskSource())
	require.NoError(t, diags.Error())

	tasks := b.Config.Resources.Jobs["git"].Tasks
	// Unset source on a git job is defaulted to GIT.
	assert.Equal(t, jobs.SourceGit, tasks[0].SparkPythonTask.Source)
	// An explicit source is preserved.
	assert.Equal(t, jobs.SourceWorkspace, tasks[1].NotebookTask.Source)
	// The nested for_each task is handled too.
	assert.Equal(t, jobs.SourceGit, tasks[2].ForEachTask.Task.SparkPythonTask.Source)
}

func TestApplyDefaultTaskSourceNullGitSourceUntouched(t *testing.T) {
	// An explicit `git_source: null` reads back as a nil value, and TranslatePaths
	// treats it as absent (typed nil pointer). The mutator must do the same, or it
	// would set source: GIT on a job whose paths are translated to workspace paths.
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    nullgit:
      name: nullgit
      git_source:
      tasks:
        - task_key: main
          spark_python_task:
            python_file: src/main.py
`))
	require.NoError(t, diags.Error())
	b := &bundle.Bundle{Config: *root}

	diags = bundle.Apply(t.Context(), b, ApplyDefaultTaskSource())
	require.NoError(t, diags.Error())

	assert.Empty(t, string(b.Config.Resources.Jobs["nullgit"].Tasks[0].SparkPythonTask.Source))
}

func TestApplyDefaultTaskSourceNonGitJobUntouched(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"plain": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{TaskKey: "py", SparkPythonTask: &jobs.SparkPythonTask{PythonFile: "/Workspace/main.py"}},
							},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyDefaultTaskSource())
	require.NoError(t, diags.Error())

	// A job without git_source is left untouched; the backend defaults source to WORKSPACE.
	assert.Empty(t, string(b.Config.Resources.Jobs["plain"].Tasks[0].SparkPythonTask.Source))
}
