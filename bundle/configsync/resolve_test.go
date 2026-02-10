package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveSelectors_NoSelectors(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      name: "Test Job"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	result, _, err := resolveSelectors("resources.jobs.test_job.name", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.name", result.String())
}

func TestResolveSelectors_NumericIndices(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      tasks:
        - task_key: "task1"
        - task_key: "task2"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	result, _, err := resolveSelectors("resources.jobs.test_job.tasks[0].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].task_key", result.String())

	result, _, err = resolveSelectors("resources.jobs.test_job.tasks[1].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].task_key", result.String())
}

func TestResolveSelectors_KeyValueSelector(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      tasks:
        - task_key: "setup"
          notebook_task:
            notebook_path: "/setup"
        - task_key: "main"
          notebook_task:
            notebook_path: "/main"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	result, _, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].notebook_task.notebook_path", result.String())

	result, _, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='setup'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].notebook_task.notebook_path", result.String())
}

func TestResolveSelectors_SelectorNotFound(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      tasks:
        - task_key: "setup"
          notebook_task:
            notebook_path: "/setup"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	_, _, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='nonexistent'].notebook_task.notebook_path", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no array element found with task_key='nonexistent'")
}

func TestResolveSelectors_SelectorOnNonArray(t *testing.T) {
	ctx := cmdio.MockDiscard(logdiag.InitContext(context.Background()))
	tmpDir := t.TempDir()

	yamlContent := `resources:
		jobs:
			test_job:
      	name: "Test Job"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	_, _, err = resolveSelectors("resources.jobs.test_job[task_key='main'].name", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot apply [task_key='main'] selector to non-array value")
}

func TestResolveSelectors_NestedSelectors(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      tasks:
        - task_key: "setup"
          libraries:
            - pypi:
                package: "pandas"
        - task_key: "main"
          libraries:
            - pypi:
                package: "numpy"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	result, _, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].libraries[0].pypi.package", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].libraries[0].pypi.package", result.String())
}

func TestResolveSelectors_WildcardNotSupported(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `resources:
  jobs:
    test_job:
      tasks:
        - task_key: "task1"
          notebook_task:
            notebook_path: "/notebook"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	_, _, err = resolveSelectors("resources.jobs.test_job.tasks.*.task_key", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard patterns are not supported")
}

func TestYamlFileIndex(t *testing.T) {
	// Simulate a sequence that was sorted alphabetically by a mutator.
	// Original YAML order: notebook_task (line 10), python_wheel_task (line 20), pipeline_task (line 30), extra (line 40)
	// Sorted order:        extra (line 40), notebook_task (line 10), pipeline_task (line 30), python_wheel_task (line 20)
	seq := []dyn.Value{
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 40}}), // extra
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 10}}), // notebook_task
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 30}}), // pipeline_task
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 20}}), // python_wheel_task
	}

	assert.Equal(t, 3, yamlFileIndex(seq, 0)) // extra: 3 elements before it in YAML
	assert.Equal(t, 0, yamlFileIndex(seq, 1)) // notebook_task: first in YAML
	assert.Equal(t, 2, yamlFileIndex(seq, 2)) // pipeline_task: 2 elements before it
	assert.Equal(t, 1, yamlFileIndex(seq, 3)) // python_wheel_task: 1 element before it
}

func TestYamlFileIndex_MultipleFiles(t *testing.T) {
	// Tasks from two different files, sorted alphabetically by mutator.
	// File A (lines 10, 20): task_a, task_b
	// File B (lines 5, 15): task_c, task_d
	// Sorted order: task_a (A:10), task_b (A:20), task_c (B:5), task_d (B:15)
	seq := []dyn.Value{
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 10}}), // task_a
		dyn.NewValue(nil, []dyn.Location{{File: "a.yml", Line: 20}}), // task_b
		dyn.NewValue(nil, []dyn.Location{{File: "b.yml", Line: 5}}),  // task_c
		dyn.NewValue(nil, []dyn.Location{{File: "b.yml", Line: 15}}), // task_d
	}

	// Indices are relative to each file
	assert.Equal(t, 0, yamlFileIndex(seq, 0)) // task_a: first in file A
	assert.Equal(t, 1, yamlFileIndex(seq, 1)) // task_b: second in file A
	assert.Equal(t, 0, yamlFileIndex(seq, 2)) // task_c: first in file B
	assert.Equal(t, 1, yamlFileIndex(seq, 3)) // task_d: second in file B
}

func TestYamlFileIndex_NoLocation(t *testing.T) {
	seq := []dyn.Value{
		dyn.NewValue(nil, nil),
		dyn.NewValue(nil, nil),
	}
	assert.Equal(t, 0, yamlFileIndex(seq, 0))
	assert.Equal(t, 1, yamlFileIndex(seq, 1))
}
