package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/cmdio"
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

	result, err := resolveSelectors("resources.jobs.test_job.name", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.name", result)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[0].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].task_key", result)

	result, err = resolveSelectors("resources.jobs.test_job.tasks[1].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].task_key", result)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].notebook_task.notebook_path", result)

	result, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='setup'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].notebook_task.notebook_path", result)
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

	_, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='nonexistent'].notebook_task.notebook_path", b, OperationReplace)
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

	_, err = resolveSelectors("resources.jobs.test_job[task_key='main'].name", b, OperationReplace)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].libraries[0].pypi.package", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].libraries[0].pypi.package", result)
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

	_, err = resolveSelectors("resources.jobs.test_job.tasks.*.task_key", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard patterns are not supported")
}
