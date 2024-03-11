package config_tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
)

func TestRelativePathsWithIncludes(t *testing.T) {
	b := loadTarget(t, "./relative_path_with_includes", "default")

	m := mutator.TranslatePaths()
	err := bundle.Apply(context.Background(), b, m)
	assert.NoError(t, err)

	assert.Equal(t, "artifact_a", b.Config.Artifacts["test_a"].Path)
	assert.Equal(t, filepath.Join("subfolder", "artifact_b"), b.Config.Artifacts["test_b"].Path)

	assert.ElementsMatch(
		t,
		[]string{
			filepath.Join("folder_a", "*.*"),
			filepath.Join("subfolder", "folder_c", "*.*"),
		},
		b.Config.Sync.Include,
	)
	assert.ElementsMatch(
		t,
		[]string{
			filepath.Join("folder_b", "*.*"),
			filepath.Join("subfolder", "folder_d", "*.*"),
		},
		b.Config.Sync.Exclude,
	)

	assert.Equal(t, filepath.Join("dist", "job_a.whl"), b.Config.Resources.Jobs["job_a"].Tasks[0].Libraries[0].Whl)
	assert.Equal(t, filepath.Join("subfolder", "dist", "job_b.whl"), b.Config.Resources.Jobs["job_b"].Tasks[0].Libraries[0].Whl)
}
