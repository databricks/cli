package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The deployment metadata service stamps a new version_id onto every job and
// pipeline on each deploy. These tests pin the invariant that a version_id-only
// change is classified as an ignored local change, so it never triggers an
// update on its own. They diff the real resource state structs so the field
// path the engine produces is verified against the rule, not just hard-coded.

func TestDeploymentVersionIDIgnoredForJobs(t *testing.T) {
	a := jobs.JobSettings{Deployment: &jobs.JobDeployment{Kind: jobs.JobDeploymentKindBundle, VersionId: "1"}}
	b := jobs.JobSettings{Deployment: &jobs.JobDeployment{Kind: jobs.JobDeploymentKindBundle, VersionId: "2"}}

	changes, err := structdiff.GetStructDiff(a, b, nil)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "deployment.version_id", changes[0].Path.String())

	cfg := dresources.GetResourceConfig("jobs")
	_, ok := findMatchingRule(changes[0].Path, cfg.IgnoreLocalChanges)
	assert.True(t, ok, "deployment.version_id must match a jobs ignore_local_changes rule")
}

func TestDeploymentVersionIDIgnoredForPipelines(t *testing.T) {
	a := pipelines.CreatePipeline{Deployment: &pipelines.PipelineDeployment{Kind: pipelines.DeploymentKindBundle, VersionId: "1"}}
	b := pipelines.CreatePipeline{Deployment: &pipelines.PipelineDeployment{Kind: pipelines.DeploymentKindBundle, VersionId: "2"}}

	changes, err := structdiff.GetStructDiff(a, b, nil)
	require.NoError(t, err)
	require.Len(t, changes, 1)
	require.Equal(t, "deployment.version_id", changes[0].Path.String())

	cfg := dresources.GetResourceConfig("pipelines")
	_, ok := findMatchingRule(changes[0].Path, cfg.IgnoreLocalChanges)
	assert.True(t, ok, "deployment.version_id must match a pipelines ignore_local_changes rule")
}
