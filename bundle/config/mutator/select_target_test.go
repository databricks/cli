package mutator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectTarget(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "foo",
			},
			Targets: map[string]*config.Target{
				"default": {
					Workspace: &config.Workspace{
						Host: "bar",
					},
				},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, mutator.SelectTarget("default"))
	require.NoError(t, diags.Error())
	assert.Equal(t, "bar", b.Config.Workspace.Host)
}

func TestSelectTargetNotFound(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"default": {},
			},
		},
	}
	diags := bundle.Apply(t.Context(), b, mutator.SelectTarget("doesnt-exist"))
	require.Error(t, diags.Error(), "no targets defined")
}

func TestSelectTargetMergesComplexVariableTaskOverride(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: testing-bundle
variables:
  job_tasks:
    type: complex
    default:
      - task_key: Task_One
        job_cluster_key: Small
resources:
  jobs:
    job_one:
      tasks: ${var.job_tasks}
    job_two:
      tasks: ${var.job_tasks}
targets:
  dev:
    resources:
      jobs:
        job_one:
          tasks:
            - task_key: Task_One
              job_cluster_key: Medium
`))
	require.NoError(t, diags.Error())

	dir := t.TempDir()
	path := filepath.Join(dir, ".databricks", "bundle", "dev", "variable-overrides.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"job_tasks":[{"task_key":"Task_One","job_cluster_key":"Small"},{"task_key":"Task_Two","job_cluster_key":"Small"},{"task_key":"Task_Three","job_cluster_key":"Small"}]}`), 0o600))

	b := &bundle.Bundle{BundleRootPath: dir, Config: *root}
	diags = bundle.ApplySeq(t.Context(), b,
		mutator.SelectTarget("dev"),
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesOnlyResources(),
		resourcemutator.MergeJobTasks(),
	)
	require.NoError(t, diags.Error())

	jobOne := b.Config.Resources.Jobs["job_one"].Tasks
	require.Len(t, jobOne, 3)
	assert.Equal(t, "Task_One", jobOne[0].TaskKey)
	assert.Equal(t, "Medium", jobOne[0].JobClusterKey)
	assert.ElementsMatch(t, []string{"Task_Two", "Task_Three"}, []string{jobOne[1].TaskKey, jobOne[2].TaskKey})

	jobTwo := b.Config.Resources.Jobs["job_two"].Tasks
	require.Len(t, jobTwo, 3)
	assert.Equal(t, "Small", jobTwo[0].JobClusterKey)
}

func TestSelectTargetRewritesComplexVariableTaskOverridePath(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
bundle:
  name: testing-bundle
workspace:
  file_path: /Workspace/files
variables:
  job_tasks:
    type: complex
    default:
      - task_key: Task_One
        notebook_task:
          notebook_path: /base
resources:
  jobs:
    job_one:
      tasks: ${var.job_tasks}
targets:
  dev:
    resources:
      jobs:
        job_one:
          tasks:
            - task_key: Task_One
              notebook_task:
                notebook_path: /Workspace${workspace.file_path}/nb
`))
	require.NoError(t, diags.Error())

	b := &bundle.Bundle{BundleRootPath: t.TempDir(), Config: *root}
	diags = bundle.ApplySeq(t.Context(), b,
		mutator.SelectTarget("dev"),
		mutator.RewriteWorkspacePrefix(),
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesOnlyResources(),
		resourcemutator.MergeJobTasks(),
	)
	require.NoError(t, diags.Error())
	require.Len(t, b.Config.Resources.Jobs["job_one"].Tasks, 1)
	assert.Equal(t, "/Workspace/files/nb", b.Config.Resources.Jobs["job_one"].Tasks[0].NotebookTask.NotebookPath)
}
