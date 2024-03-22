package config_tests

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConflictingResourceIdsNoSubconfig(t *testing.T) {
	ctx := context.Background()
	_, err := bundle.Load(ctx, "./conflicting_resource_ids/no_subconfigurations")
	bundleConfigPath := filepath.FromSlash("conflicting_resource_ids/no_subconfigurations/databricks.yml")
	assert.ErrorContains(t, err, fmt.Sprintf("multiple resources named foo (job at %s, pipeline at %s)", bundleConfigPath, bundleConfigPath))
}

func TestConflictingResourceIdsOneSubconfig(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./conflicting_resource_ids/one_subconfiguration")
	require.NoError(t, err)
	diags := bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	bundleConfigPath := filepath.FromSlash("conflicting_resource_ids/one_subconfiguration/databricks.yml")
	resourcesConfigPath := filepath.FromSlash("conflicting_resource_ids/one_subconfiguration/resources.yml")
	assert.ErrorContains(t, diags.Error(), fmt.Sprintf("multiple resources named foo (job at %s, pipeline at %s)", bundleConfigPath, resourcesConfigPath))
}

func TestConflictingResourceIdsTwoSubconfigs(t *testing.T) {
	ctx := context.Background()
	b, err := bundle.Load(ctx, "./conflicting_resource_ids/two_subconfigurations")
	require.NoError(t, err)
	diags := bundle.Apply(ctx, b, bundle.Seq(mutator.DefaultMutators()...))
	resources1ConfigPath := filepath.FromSlash("conflicting_resource_ids/two_subconfigurations/resources1.yml")
	resources2ConfigPath := filepath.FromSlash("conflicting_resource_ids/two_subconfigurations/resources2.yml")
	assert.ErrorContains(t, diags.Error(), fmt.Sprintf("multiple resources named foo (job at %s, pipeline at %s)", resources1ConfigPath, resources2ConfigPath))
}
