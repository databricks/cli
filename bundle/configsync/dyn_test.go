package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
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

	result, err := resolveSelectors("resources.jobs.test_job.name", b)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[0].task_key", b)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].task_key", result)

	result, err = resolveSelectors("resources.jobs.test_job.tasks[1].task_key", b)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].notebook_task.notebook_path", b)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].notebook_task.notebook_path", result)

	result, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='setup'].notebook_task.notebook_path", b)
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

	_, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='nonexistent'].notebook_task.notebook_path", b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no array element found with task_key='nonexistent'")
}

func TestResolveSelectors_SelectorOnNonArray(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	tmpDir := t.TempDir()

	yamlContent := `job:
  name: "Test Job"
`
	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	_, err = resolveSelectors("job[task_key='main'].name", b)
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

	result, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].libraries[0].pypi.package", b)
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

	_, err = resolveSelectors("resources.jobs.test_job.tasks.*.task_key", b)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard patterns are not supported")
}

func TestStructpathToDynPath_SimplePaths(t *testing.T) {
	path, err := structpathToDynPath("resources.jobs.test_job.name")
	require.NoError(t, err)
	assert.Equal(t, dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.Key("test_job"),
		dyn.Key("name"),
	), path)
}

func TestStructpathToDynPath_WithIndices(t *testing.T) {
	path, err := structpathToDynPath("tasks[0].name")
	require.NoError(t, err)
	assert.Equal(t, dyn.NewPath(
		dyn.Key("tasks"),
		dyn.Index(0),
		dyn.Key("name"),
	), path)

	path, err = structpathToDynPath("resources.jobs.test_job.tasks[2].timeout_seconds")
	require.NoError(t, err)
	assert.Equal(t, dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.Key("test_job"),
		dyn.Key("tasks"),
		dyn.Index(2),
		dyn.Key("timeout_seconds"),
	), path)
}

func TestStructpathToDynPath_ErrorOnUnresolvedSelector(t *testing.T) {
	_, err := structpathToDynPath("tasks[task_key='main'].name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved selector [task_key='main']")
	assert.Contains(t, err.Error(), "call resolveSelectors first")
}

func TestStructpathToDynPath_WildcardNotSupported(t *testing.T) {
	_, err := structpathToDynPath("tasks.*.name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard patterns are not supported")
}

func TestStrPathToJSONPointer_SimplePaths(t *testing.T) {
	pointer, err := strPathToJSONPointer("resources.jobs.test_job")
	require.NoError(t, err)
	assert.Equal(t, "/resources/jobs/test_job", pointer)
}

func TestStrPathToJSONPointer_WithIndices(t *testing.T) {
	pointer, err := strPathToJSONPointer("tasks[0].name")
	require.NoError(t, err)
	assert.Equal(t, "/tasks/0/name", pointer)

	pointer, err = strPathToJSONPointer("resources.jobs.test[0].tasks[1].timeout")
	require.NoError(t, err)
	assert.Equal(t, "/resources/jobs/test/0/tasks/1/timeout", pointer)
}

func TestStrPathToJSONPointer_RFC6902Escaping(t *testing.T) {
	pointer, err := strPathToJSONPointer("path.with~tilde")
	require.NoError(t, err)
	assert.Equal(t, "/path/with~0tilde", pointer)

	pointer, err = strPathToJSONPointer("path.with/slash")
	require.NoError(t, err)
	assert.Equal(t, "/path/with~1slash", pointer)

	pointer, err = strPathToJSONPointer("path.with~tilde/and~slash")
	require.NoError(t, err)
	assert.Equal(t, "/path/with~0tilde~1and~0slash", pointer)
}

func TestStrPathToJSONPointer_EmptyPath(t *testing.T) {
	pointer, err := strPathToJSONPointer("")
	require.NoError(t, err)
	assert.Equal(t, "", pointer)
}

func TestStrPathToJSONPointer_ErrorOnUnresolvedSelector(t *testing.T) {
	_, err := strPathToJSONPointer("tasks[task_key='main'].name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved selector [task_key='main']")
	assert.Contains(t, err.Error(), "call resolveSelectors first")
}

func TestStrPathToJSONPointer_WildcardNotSupported(t *testing.T) {
	_, err := strPathToJSONPointer("tasks.*.name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcard patterns are not supported")
}

func TestDynPathToJSONPointer_ExistingFunction(t *testing.T) {
	path := dyn.NewPath(
		dyn.Key("resources"),
		dyn.Key("jobs"),
		dyn.Key("test_job"),
	)
	pointer := dynPathToJSONPointer(path)
	assert.Equal(t, "/resources/jobs/test_job", pointer)

	path = dyn.NewPath(
		dyn.Key("tasks"),
		dyn.Index(1),
		dyn.Key("timeout"),
	)
	pointer = dynPathToJSONPointer(path)
	assert.Equal(t, "/tasks/1/timeout", pointer)
}

func TestDynPathToJSONPointer_EmptyPath(t *testing.T) {
	pointer := dynPathToJSONPointer(dyn.Path{})
	assert.Equal(t, "", pointer)
}
