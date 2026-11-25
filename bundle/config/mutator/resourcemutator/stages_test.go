package resourcemutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestStageMutatorOrder pins which mutators run in each stage and in which order.
// Within a stage, mutators of different resource types must not depend on each other.
// The ordering rules that do matter:
//   - InitializeVolumePaths computes volume_path before references to it are resolved
//     (one file registers both, in this order).
//   - The StageAfterTargetMerge mutators only merge keyed sequences, because code that
//     loads a configuration without the initialize phase runs that stage on its own.
func TestStageMutatorOrder(t *testing.T) {
	expected := map[Stage][]string{
		StageBeforeVariables: {
			"ValidateSecretValueIsVariable",
		},
		StageBeforePresets: {
			"OverrideCompute",
		},
		StageAfterDefaults: {
			"DashboardFixups",
			"GenieSpaceFixups",
		},
		StageAfterResourceVariables: {
			"ExpandPipelineGlobPaths",
			"NormalizeMaterializationSetMaterializedFeatureMembers",
		},
		StageAfterTargetMerge: {
			"MergeJobClusters",
			"MergeJobParameters",
			"MergeJobTasks",
			"MergePipelineClusters",
			"MergeApps",
		},
		StageAfterUCDependencies: {
			"JobClustersFixups",
			"ModelServingEndpointFixups",
			"ClusterFixups",
			"ConfigureDashboardSerializedDashboard",
			"ConfigureGenieSpaceSerializedSpace",
			"ConfigureAppThumbnail",
			"LoadDBAlertFiles",
			"ConfigureClusterPolicyDefinition",
		},
		StageAfterPython: {
			"ApplyDefaultTaskSource",
			"InitializeVolumePaths",
			"ResolveVariableReferences(resources)",
		},
		StageAfterFieldValidation: {
			"ValidateJobRunTriggers",
		},
		StageBeforeDirectEngine: {
			"SecretScopeFixups",
		},
		StageAfterBuild: {
			"ResolveJobRunFileTriggers",
		},
	}

	actual := map[Stage][]string{}
	for stage, mutators := range stageMutatorsByStage() {
		for _, m := range mutators {
			actual[stage] = append(actual[stage], m.newMutator().Name())
		}
	}
	assert.Equal(t, expected, actual)
}
