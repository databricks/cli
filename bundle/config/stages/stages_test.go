package stages_test

import (
	"testing"

	_ "github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/stages"
	_ "github.com/databricks/cli/bundle/config/validate"
	"github.com/stretchr/testify/assert"
)

// TestStageMutatorOrder pins which mutators run in each stage and in which order. It
// lists every registration on purpose, so that adding or moving a mutator shows up here.
// Within a stage, mutators of different resource types must not depend on each other.
// The ordering rules that do matter:
//   - InitializeVolumePaths computes volume_path before references to it are resolved
//     (one call registers both, in this order).
//   - The StageAfterTargetMerge mutators only merge keyed sequences, because code that
//     loads a configuration without the initialize phase runs that stage on its own.
//   - The StageFastValidation mutators run in parallel, so they are read-only.
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
			"validate:pipeline_development_deprecated",
			"InitializeVolumePaths",
			"ResolveVariableReferences(resources)",
		},
		stages.StageAfterFieldValidation: {
			"ValidateJobRunTriggers",
			"validate:job_run_idempotency_token",
			"validate:validate_dashboard_etags",
			"validate:validate_genie_space_etags",
			"validate:postgres_data_api_settings",
		},
		stages.StageBeforePlan: {
			"SecretScopeFixups",
		},
		stages.StageAfterBuild: {
			"ResolveJobRunFileTriggers",
		},
		stages.StageFastValidation: {
			"validate:job_cluster_key_defined",
			"validate:job_task_cluster_spec",
		},
	}

	assert.Equal(t, expected, stages.MutatorNamesByStage())
}
