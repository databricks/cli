package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/require"
)

func TestComplexVariables(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex", "default")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesInComplexVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())

	require.Equal(t, "13.2.x-scala2.11", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkVersion)
	require.Equal(t, "Standard_DS3_v2", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NodeTypeId)
	require.Equal(t, "some-policy-id", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.PolicyId)
	require.Equal(t, 2, b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NumWorkers)
	require.Equal(t, "true", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.speculation"])
	require.Equal(t, "true", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.random"])

	require.Equal(t, 3, len(b.Config.Resources.Jobs["my_job"].Tasks[0].Libraries))
	require.Contains(t, b.Config.Resources.Jobs["my_job"].Tasks[0].Libraries, compute.Library{
		Jar: "/path/to/jar",
	})
	require.Contains(t, b.Config.Resources.Jobs["my_job"].Tasks[0].Libraries, compute.Library{
		Egg: "/path/to/egg",
	})
	require.Contains(t, b.Config.Resources.Jobs["my_job"].Tasks[0].Libraries, compute.Library{
		Whl: "/path/to/whl",
	})

	require.Equal(t, "task with spark version 13.2.x-scala2.11 and jar /path/to/jar", b.Config.Resources.Jobs["my_job"].Tasks[0].TaskKey)
}

func TestComplexVariablesOverride(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex", "dev")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesInComplexVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())

	require.Equal(t, "14.2.x-scala2.11", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkVersion)
	require.Equal(t, "Standard_DS3_v3", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NodeTypeId)
	require.Equal(t, 4, b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.NumWorkers)
	require.Equal(t, "false", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.speculation"])

	// Making sure the variable is overriden and not merged / extended
	// These properties are set in the default target but not set in override target
	// So they should be empty
	require.Equal(t, "", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.SparkConf["spark.random"])
	require.Equal(t, "", b.Config.Resources.Jobs["my_job"].JobClusters[0].NewCluster.PolicyId)
}

func TestComplexVariablesOverrideWithMultipleFiles(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex_multiple_files", "dev")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesInComplexVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	for _, cluster := range b.Config.Resources.Jobs["my_job"].JobClusters {
		require.Equalf(t, "14.2.x-scala2.11", cluster.NewCluster.SparkVersion, "cluster: %v", cluster.JobClusterKey)
		require.Equalf(t, "Standard_DS3_v2", cluster.NewCluster.NodeTypeId, "cluster: %v", cluster.JobClusterKey)
		require.Equalf(t, 4, cluster.NewCluster.NumWorkers, "cluster: %v", cluster.JobClusterKey)
		require.Equalf(t, "false", cluster.NewCluster.SparkConf["spark.speculation"], "cluster: %v", cluster.JobClusterKey)
	}
}

func TestComplexVariablesOverrideWithFullSyntax(t *testing.T) {
	b, diags := loadTargetWithDiags("variables/complex", "dev")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, bundle.Seq(
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesInComplexVariables(),
		mutator.ResolveVariableReferences(
			"variables",
		),
	))
	require.NoError(t, diags.Error())
	require.Empty(t, diags)

	complexvar := b.Config.Variables["complexvar"].Value
	require.Equal(t, map[string]interface{}{"key1": "1", "key2": "2", "key3": "3"}, complexvar)
}
