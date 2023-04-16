package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictingResourceIdsNoSubconfig(t *testing.T) {
	_, err := bundle.Load("./conflicting_resource_ids/no_subconfigurations")
	assert.ErrorContains(t, err, "multiple resources named foo (job at conflicting_resource_ids/no_subconfigurations/bundle.yml, pipeline at conflicting_resource_ids/no_subconfigurations/bundle.yml)")
}

func TestConflictingResourceIdsOneSubconfig(t *testing.T) {
	b, err := bundle.Load("./conflicting_resource_ids/one_subconfiguration")
	require.NoError(t, err)
	err = bundle.Apply(context.Background(), b, mutator.DefaultMutators())
	assert.ErrorContains(t, err, "multiple resources named foo (job at conflicting_resource_ids/one_subconfiguration/bundle.yml, pipeline at conflicting_resource_ids/one_subconfiguration/resources.yml)")
}

func TestConflictingResourceIdsTwoSubconfigs(t *testing.T) {
	b, err := bundle.Load("./conflicting_resource_ids/two_subconfigurations")
	require.NoError(t, err)
	err = bundle.Apply(context.Background(), b, mutator.DefaultMutators())
	assert.ErrorContains(t, err, "multiple resources named foo (job at conflicting_resource_ids/two_subconfigurations/resources1.yml, pipeline at conflicting_resource_ids/two_subconfigurations/resources2.yml)")
}
