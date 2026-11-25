package stages_test

import (
	"testing"

	_ "github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/stages"
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
	expected := map[stages.Stage][]string{
		stages.StageBeforeVariables: {
			"ValidateSecretValueIsVariable",
		},
		stages.StageBeforePresets: {
			"OverrideCompute",
		},
		stages.StageAfterDefaults: {
			"DashboardFixups",
			"GenieSpaceFixups",
		},
		stages.StageAfterResourceVariables: {
			"ExpandPipelineGlobPaths",
			"NormalizeMaterializationSetMaterializedFeatureMembers",
		},
		stages.StageAfterTargetMerge: {
			"MergeJobClusters",
			"MergeJobParameters",
			"MergeJobTasks",
			"MergePipelineClusters",
			"MergeApps",
		},
		stages.StageAfterUCDependencies: {
			"JobClustersFixups",
			"ModelServingEndpointFixups",
			"ClusterFixups",
			"ConfigureDashboardSerializedDashboard",
			"ConfigureGenieSpaceSerializedSpace",
			"ConfigureAppThumbnail",
			"LoadDBAlertFiles",
			"ConfigureClusterPolicyDefinition",
		},
		stages.StageAfterPython: {
			"ApplyDefaultTaskSource",
			"InitializeVolumePaths",
			"ResolveVariableReferences(resources)",
		},
		stages.StageAfterFieldValidation: {
			"ValidateJobRunTriggers",
		},
		stages.StageBeforeDirectEngine: {
			"SecretScopeFixups",
		},
		stages.StageAfterBuild: {
			"ResolveJobRunFileTriggers",
		},
	}

	assert.Equal(t, expected, stages.MutatorNamesByStage())
}
