package configsync

import (
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
	ctx := logdiag.InitContext(t.Context())
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

	result, _, _, err := resolveSelectors("resources.jobs.test_job.name", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.name", result.String())
}

func TestResolveSelectors_NumericIndices(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
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

	result, _, _, err := resolveSelectors("resources.jobs.test_job.tasks[0].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].task_key", result.String())

	result, _, _, err = resolveSelectors("resources.jobs.test_job.tasks[1].task_key", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].task_key", result.String())
}

func TestResolveSelectors_KeyValueSelector(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
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

	result, _, _, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].notebook_task.notebook_path", result.String())

	result, _, _, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='setup'].notebook_task.notebook_path", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[0].notebook_task.notebook_path", result.String())
}

func TestResolveSelectors_SelectorNotFound(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
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

	_, _, _, err = resolveSelectors("resources.jobs.test_job.tasks[task_key='nonexistent'].notebook_task.notebook_path", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no array element found with task_key='nonexistent'")
}

func TestResolveSelectors_SelectorOnNonArray(t *testing.T) {
	ctx := cmdio.MockDiscard(logdiag.InitContext(t.Context()))
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

	_, _, _, err = resolveSelectors("resources.jobs.test_job[task_key='main'].name", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot apply [task_key='main'] selector to non-array value")
}

func TestResolveSelectors_NestedSelectors(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
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

	result, _, _, err := resolveSelectors("resources.jobs.test_job.tasks[task_key='main'].libraries[0].pypi.package", b, OperationReplace)
	require.NoError(t, err)
	assert.Equal(t, "resources.jobs.test_job.tasks[1].libraries[0].pypi.package", result.String())
}

func TestResolveSelectors_WildcardNotSupported(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
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

	_, _, _, err = resolveSelectors("resources.jobs.test_job.tasks.*.task_key", b, OperationReplace)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wildcards not allowed in path")
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
	// Single block: the sequence node is anchored at the block's line in the file.
	seqLocations := []dyn.Location{{File: "a.yml", Line: 5}}

	assert.Equal(t, 3, yamlFileIndex(seq, 0, seqLocations)) // extra: 3 elements before it in YAML
	assert.Equal(t, 0, yamlFileIndex(seq, 1, seqLocations)) // notebook_task: first in YAML
	assert.Equal(t, 2, yamlFileIndex(seq, 2, seqLocations)) // pipeline_task: 2 elements before it
	assert.Equal(t, 1, yamlFileIndex(seq, 3, seqLocations)) // python_wheel_task: 1 element before it
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
	// One anchor per file, each above its file's first element.
	seqLocations := []dyn.Location{{File: "a.yml", Line: 3}, {File: "b.yml", Line: 2}}

	// Indices are relative to each file
	assert.Equal(t, 0, yamlFileIndex(seq, 0, seqLocations)) // task_a: first in file A
	assert.Equal(t, 1, yamlFileIndex(seq, 1, seqLocations)) // task_b: second in file A
	assert.Equal(t, 0, yamlFileIndex(seq, 2, seqLocations)) // task_c: first in file B
	assert.Equal(t, 1, yamlFileIndex(seq, 3, seqLocations)) // task_d: second in file B
}

func TestYamlFileIndex_NoLocation(t *testing.T) {
	seq := []dyn.Value{
		dyn.NewValue(nil, nil),
		dyn.NewValue(nil, nil),
	}
	assert.Equal(t, 0, yamlFileIndex(seq, 0, nil))
	assert.Equal(t, 1, yamlFileIndex(seq, 1, nil))
}

func TestYamlFileIndex_SplitBlocksSameFile(t *testing.T) {
	// A job's tasks split across two blocks in the SAME file: a targets override
	// block (aaa, near the top) and the top-level resources block (bbb, ccc, lower
	// down). MergeJobTasks concatenates and sorts by task_key -> [aaa, bbb, ccc].
	// The tasks sequence node carries one anchor per block: top-level tasks at line
	// 20, target tasks at line 11 (order matches the real merge, which keeps the
	// top-level location first). Each element's index must be relative to its own
	// block, not the whole file.
	seq := []dyn.Value{
		dyn.NewValue(nil, []dyn.Location{{File: "databricks.yml", Line: 11}}), // aaa (target block)
		dyn.NewValue(nil, []dyn.Location{{File: "databricks.yml", Line: 20}}), // bbb (top-level block)
		dyn.NewValue(nil, []dyn.Location{{File: "databricks.yml", Line: 28}}), // ccc (top-level block)
	}
	seqLocations := []dyn.Location{{File: "databricks.yml", Line: 20}, {File: "databricks.yml", Line: 11}}

	assert.Equal(t, 0, yamlFileIndex(seq, 0, seqLocations)) // aaa: first in target block
	assert.Equal(t, 0, yamlFileIndex(seq, 1, seqLocations)) // bbb: first in top-level block (NOT 1)
	assert.Equal(t, 1, yamlFileIndex(seq, 2, seqLocations)) // ccc: second in top-level block
}

func TestElementInOverrideBlock(t *testing.T) {
	// seqLocations[0] is always the top-level anchor (merge keeps the reference
	// block first); seqLocations[1] is the target override anchor. This holds
	// regardless of which block appears first in the file.
	topFirst := []dyn.Location{{File: "databricks.yml", Line: 6}, {File: "databricks.yml", Line: 20}}
	assert.True(t, elementInOverrideBlock(dyn.Location{File: "databricks.yml", Line: 22}, topFirst)) // target block
	assert.False(t, elementInOverrideBlock(dyn.Location{File: "databricks.yml", Line: 8}, topFirst)) // top-level block

	// Target block precedes resources in the file: top-level anchor still first.
	targetFirst := []dyn.Location{{File: "databricks.yml", Line: 20}, {File: "databricks.yml", Line: 6}}
	assert.True(t, elementInOverrideBlock(dyn.Location{File: "databricks.yml", Line: 8}, targetFirst))   // target block
	assert.False(t, elementInOverrideBlock(dyn.Location{File: "databricks.yml", Line: 22}, targetFirst)) // top-level block

	// Single block (no override contribution) is never in an override block.
	single := []dyn.Location{{File: "databricks.yml", Line: 6}}
	assert.False(t, elementInOverrideBlock(dyn.Location{File: "databricks.yml", Line: 8}, single))
	assert.False(t, elementInOverrideBlock(dyn.Location{}, topFirst))

	// Multi-file split: a top-level block in a.yml and a target block in b.yml.
	multiFile := []dyn.Location{{File: "a.yml", Line: 6}, {File: "b.yml", Line: 4}}
	assert.True(t, elementInOverrideBlock(dyn.Location{File: "b.yml", Line: 6}, multiFile))  // target block, other file
	assert.False(t, elementInOverrideBlock(dyn.Location{File: "a.yml", Line: 8}, multiFile)) // top-level block
}
