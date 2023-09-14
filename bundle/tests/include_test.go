package config_tests

import (
	"context"
	"path/filepath"
	"sort"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
)

func TestIncludeInvalid(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./include_invalid")
	require.NoError(t, err)
	err = bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notexists.yml defined in 'include' section does not match any files")
}

func TestIncludeWithGlob(t *testing.T) {
	b := load(t, "./include_with_glob")

	keys := maps.Keys(b.Config.Resources.Jobs)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_job"}, keys)

	job := b.Config.Resources.Jobs["my_job"]
	assert.Equal(t, "1", job.ID)
	assert.Equal(t, "include_with_glob/job.yml", filepath.ToSlash(job.ConfigFilePath))
}

func TestIncludeDefault(t *testing.T) {
	b := load(t, "./include_default")

	// No jobs should have been loaded
	assert.Empty(t, b.Config.Resources.Jobs)
}

func TestIncludeForMultipleMatches(t *testing.T) {
	b := load(t, "./include_multiple")

	// Test that both jobs were loaded.
	keys := maps.Keys(b.Config.Resources.Jobs)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_first_job", "my_second_job"}, keys)

	first := b.Config.Resources.Jobs["my_first_job"]
	assert.Equal(t, "1", first.ID)
	assert.Equal(t, "include_multiple/my_first_job/resource.yml", filepath.ToSlash(first.ConfigFilePath))

	second := b.Config.Resources.Jobs["my_second_job"]
	assert.Equal(t, "2", second.ID)
	assert.Equal(t, "include_multiple/my_second_job/resource.yml", filepath.ToSlash(second.ConfigFilePath))
}
