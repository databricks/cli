package bundle_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleValidate(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(tmpDir, "databricks.yml"),
		`
bundle:
  name: "foobar"

resources:
  jobs:
    outer_loop:
      name: outer loop
      tasks:
        - task_key: my task
          run_job_task:
            job_id: ${resources.jobs.inner_loop.id}

    inner_loop:
      name: inner loop

`)

	ctx := context.Background()
	stdout, err := validateBundle(t, ctx, tmpDir)
	require.NoError(t, err)

	config := make(map[string]any)
	err = json.Unmarshal(stdout, &config)
	require.NoError(t, err)

	getValue := func(key string) any {
		v, err := convert.FromTyped(config, dyn.NilValue)
		require.NoError(t, err)
		v, err = dyn.GetByPath(v, dyn.MustPathFromString(key))
		require.NoError(t, err)
		return v.AsAny()
	}

	assert.Equal(t, "foobar", getValue("bundle.name"))
	assert.Equal(t, "outer loop", getValue("resources.jobs.outer_loop.name"))
	assert.Equal(t, "inner loop", getValue("resources.jobs.inner_loop.name"))
	assert.Equal(t, "my task", getValue("resources.jobs.outer_loop.tasks[0].task_key"))
	// Assert resource references are retained in the output.
	assert.Equal(t, "${resources.jobs.inner_loop.id}", getValue("resources.jobs.outer_loop.tasks[0].run_job_task.job_id"))
}
