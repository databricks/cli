package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPresetsSkipsNilQualityMonitors(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Presets: config.Presets{
				TriggerPauseStatus: config.Paused,
			},
			Resources: config.Resources{
				QualityMonitors: map[string]*resources.QualityMonitor{
					// Python-generated configs can contain null resource entries that
					// bypass AllResourcesHaveValues, so ApplyPresets must skip them.
					"nil_monitor": nil,
					"monitor": {
						TableName: "monitor",
						CreateMonitor: catalog.CreateMonitor{
							OutputSchemaName: "catalog.schema",
							Schedule:         &catalog.MonitorCronSchedule{},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())
	assert.Nil(t, b.Config.Resources.QualityMonitors["monitor"].Schedule)
}

func TestApplyPresetsNamePrefixReferenceToName(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Presets: config.Presets{
				NamePrefix: "pre-",
			},
			Experimental: &config.Experimental{
				SkipNamePrefixForSchema: true,
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"base":      {JobSettings: jobs.JobSettings{Name: "base"}},
					"to_name":   {JobSettings: jobs.JobSettings{Name: "${resources.jobs.base.name}"}},
					"to_id":     {JobSettings: jobs.JobSettings{Name: "${resources.jobs.base.id}"}},
					"to_schema": {JobSettings: jobs.JobSettings{Name: "${resources.schemas.s.name}"}},
					"not_pure":  {JobSettings: jobs.JobSettings{Name: "${resources.jobs.base.name} copy"}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"to_job": {CreatePipeline: pipelines.CreatePipeline{Name: "${resources.jobs.base.name}"}},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"to_experiment": {CreateExperiment: ml.CreateExperiment{Name: "${resources.experiments.e.name}"}},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"to_model": {CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{Name: "${resources.registered_models.m.name}"}},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyPresets())
	require.NoError(t, diags.Error())

	r := b.Config.Resources
	assert.Equal(t, "pre-base", r.Jobs["base"].Name)
	assert.Equal(t, "${resources.jobs.base.name}", r.Jobs["to_name"].Name)
	assert.Equal(t, "pre-${resources.jobs.base.id}", r.Jobs["to_id"].Name)
	// Schemas are not prefixed with skip_name_prefix_for_schema, so the reference is.
	assert.Equal(t, "pre-${resources.schemas.s.name}", r.Jobs["to_schema"].Name)
	assert.Equal(t, "pre-${resources.jobs.base.name} copy", r.Jobs["not_pure"].Name)
	assert.Equal(t, "${resources.jobs.base.name}", r.Pipelines["to_job"].Name)
	assert.Equal(t, "${resources.experiments.e.name}", r.Experiments["to_experiment"].Name)
	assert.Equal(t, "${resources.registered_models.m.name}", r.RegisteredModels["to_model"].Name)
}
