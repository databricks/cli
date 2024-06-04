package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/stretchr/testify/require"
)

func TestComplexVariables(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex", "default")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())

	require.Equal(t, "13.2.x-scala2.11", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkVersion)
	require.Equal(t, 2, b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NumWorkers)
	require.Equal(t, "true", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.speculation"])
}

func TestComplexVariablesOverride(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex", "dev")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())

	require.Equal(t, "14.2.x-scala2.11", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkVersion)
	require.Equal(t, 4, b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NumWorkers)
	require.Equal(t, "false", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.speculation"])
}
