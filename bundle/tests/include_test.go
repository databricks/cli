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
	b, err := bundle.Load("./include_invalid")
	require.NoError(t, err)
	err = bundle.Apply(context.Background(), b, bundle.Seq(mutator.DefaultMutators()...))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notexists.yml defined in 'include' section does not match any files")
}

func TestIncludeWithGlob(t *testing.T) {
	b := load(t, "./include_with_glob")

	// Test that both jobs were loaded.
	keys := maps.Keys(b.Config.Resources.Jobs)
	sort.Strings(keys)
	assert.Equal(t, []string{"my_job"}, keys)

	job := b.Config.Resources.Jobs["my_job"]
	assert.Equal(t, "1", job.ID)
	assert.Equal(t, "include_with_glob/job.yml", filepath.ToSlash(job.ConfigFilePath))
}
