package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestRelativePathsWithIncludes(t *testing.T) {
	b := load(t, "./relative_path_with_includes")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, err)

	assert.Equal(t, "artifact_a", b.Config.Artifacts["test_a"].Path)
	assert.Equal(t, "subfolder/artifact_b", b.Config.Artifacts["test_b"].Path)

	assert.ElementsMatch(t, []string{"./folder_a/*.*", "subfolder/folder_c/*.*"}, b.Config.Sync.Include)
	assert.ElementsMatch(t, []string{"./folder_b/*.*", "subfolder/folder_d/*.*"}, b.Config.Sync.Exclude)

	assert.Equal(t, "dist/job_a.whl", b.Config.Resources.Jobs["job_a"].Tasks[0].Libraries[0].Whl)
	assert.Equal(t, "subfolder/dist/job_b.whl", b.Config.Resources.Jobs["job_b"].Tasks[0].Libraries[0].Whl)
}
