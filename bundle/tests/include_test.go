package config_tests

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncludeInvalid(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())
	logdiag.SetCollect(ctx, true)
	b, err := bundle.Load(ctx, "./include_invalid")
	require.NoError(t, err)
	phases.Load(ctx, b)
	diags := logdiag.FlushCollected(ctx)
	require.Error(t, diags.Error())
	assert.ErrorContains(t, diags.Error(), "notexists.yml defined in 'include' section does not match any files")
}

func TestIncludeWithGlob(t *testing.T) {
	b := load(t, "./include_with_glob")

	keys := utils.SortedKeys(b.Config.Resources.Jobs)
	assert.Equal(t, []string{"my_job"}, keys)

	job := b.Config.Resources.Jobs["my_job"]
	assert.Equal(t, "1", job.ID)
	l := b.Config.GetLocation("resources.jobs.my_job")
	assert.Equal(t, "include_with_glob/job.yml", filepath.ToSlash(l.File))
}

func TestIncludeDefault(t *testing.T) {
	b := load(t, "./include_default")

	// No jobs should have been loaded
	assert.Empty(t, b.Config.Resources.Jobs)
}

func TestIncludeForMultipleMatches(t *testing.T) {
	b := load(t, "./include_multiple")

	// Test that both jobs were loaded.
	keys := utils.SortedKeys(b.Config.Resources.Jobs)
	assert.Equal(t, []string{"my_first_job", "my_second_job"}, keys)

	first := b.Config.Resources.Jobs["my_first_job"]
	assert.Equal(t, "1", first.ID)
	fl := b.Config.GetLocation("resources.jobs.my_first_job")
	assert.Equal(t, "include_multiple/my_first_job/resource.yml", filepath.ToSlash(fl.File))

	second := b.Config.Resources.Jobs["my_second_job"]
	assert.Equal(t, "2", second.ID)
	sl := b.Config.GetLocation("resources.jobs.my_second_job")
	assert.Equal(t, "include_multiple/my_second_job/resource.yml", filepath.ToSlash(sl.File))
}
