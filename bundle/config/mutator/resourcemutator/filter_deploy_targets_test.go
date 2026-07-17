package resourcemutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
)

func filterDeployTargetsBundle(target string) *bundle.Bundle {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					// No deploy_targets: deployed everywhere.
					"everywhere": {
						JobSettings: jobs.JobSettings{Name: "everywhere"},
					},
					// Restricted to dev and staging.
					"dev_and_staging": {
						BaseResource: resources.BaseResource{DeployTargets: []string{"dev", "staging"}},
						JobSettings:  jobs.JobSettings{Name: "dev_and_staging"},
					},
					// Restricted to prod only.
					"prod_only": {
						BaseResource: resources.BaseResource{DeployTargets: []string{"prod"}},
						JobSettings:  jobs.JobSettings{Name: "prod_only"},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					// A different resource type, restricted to prod, to confirm
					// filtering is applied uniformly across resource kinds.
					"pipeline_prod_only": {
						BaseResource:   resources.BaseResource{DeployTargets: []string{"prod"}},
						CreatePipeline: pipelines.CreatePipeline{Name: "pipeline_prod_only"},
					},
				},
			},
		},
	}
	b.Config.Bundle.Target = target
	return b
}

func TestFilterDeployTargetsDev(t *testing.T) {
	b := filterDeployTargetsBundle("dev")

	diags := bundle.Apply(t.Context(), b, resourcemutator.FilterDeployTargets())
	assert.NoError(t, diags.Error())

	// everywhere has no list so it stays; dev_and_staging includes dev so it
	// stays; prod_only does not include dev so it is dropped.
	assert.Contains(t, b.Config.Resources.Jobs, "everywhere")
	assert.Contains(t, b.Config.Resources.Jobs, "dev_and_staging")
	assert.NotContains(t, b.Config.Resources.Jobs, "prod_only")
	assert.Len(t, b.Config.Resources.Jobs, 2)

	// The prod-only pipeline is dropped in dev as well.
	assert.NotContains(t, b.Config.Resources.Pipelines, "pipeline_prod_only")
	assert.Empty(t, b.Config.Resources.Pipelines)
}

func TestFilterDeployTargetsProd(t *testing.T) {
	b := filterDeployTargetsBundle("prod")

	diags := bundle.Apply(t.Context(), b, resourcemutator.FilterDeployTargets())
	assert.NoError(t, diags.Error())

	// everywhere stays; dev_and_staging does not include prod so it is dropped;
	// prod_only includes prod so it stays.
	assert.Contains(t, b.Config.Resources.Jobs, "everywhere")
	assert.NotContains(t, b.Config.Resources.Jobs, "dev_and_staging")
	assert.Contains(t, b.Config.Resources.Jobs, "prod_only")
	assert.Len(t, b.Config.Resources.Jobs, 2)

	// The prod-only pipeline is kept in prod.
	assert.Contains(t, b.Config.Resources.Pipelines, "pipeline_prod_only")
	assert.Len(t, b.Config.Resources.Pipelines, 1)
}

func TestFilterDeployTargetsUnlistedTarget(t *testing.T) {
	// A target that appears in no deploy_targets list keeps only the resources
	// that have no list at all.
	b := filterDeployTargetsBundle("qa")

	diags := bundle.Apply(t.Context(), b, resourcemutator.FilterDeployTargets())
	assert.NoError(t, diags.Error())

	assert.Contains(t, b.Config.Resources.Jobs, "everywhere")
	assert.NotContains(t, b.Config.Resources.Jobs, "dev_and_staging")
	assert.NotContains(t, b.Config.Resources.Jobs, "prod_only")
	assert.Len(t, b.Config.Resources.Jobs, 1)
	assert.Empty(t, b.Config.Resources.Pipelines)
}

func TestFilterDeployTargetsNoFieldsSet(t *testing.T) {
	// When no resource declares deploy_targets, the mutator makes no changes.
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"a": {JobSettings: jobs.JobSettings{Name: "a"}},
					"b": {JobSettings: jobs.JobSettings{Name: "b"}},
				},
			},
		},
	}
	b.Config.Bundle.Target = "dev"

	diags := bundle.Apply(t.Context(), b, resourcemutator.FilterDeployTargets())
	assert.NoError(t, diags.Error())
	assert.Len(t, b.Config.Resources.Jobs, 2)
}
