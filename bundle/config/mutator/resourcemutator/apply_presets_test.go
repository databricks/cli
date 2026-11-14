package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/serving"
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

func TestApplyPresetsNormalizedPrefixPreservesUnderscores(t *testing.T) {
	tests := []struct {
		name       string
		prefix     string
		wantPrefix string
	}{
		{"trailing underscore", "my_user_", "my_user_"},
		{"repeated underscores", "my__user__", "my__user__"},
		{"bracketed underscores", "[qa__team] ", "qa__team_"},
		{"bracketed trailing underscore", "[qa_] ", "qa_"},
		{"bracketed repeated trailing underscores", "[qa__] ", "qa__"},
		{"bracketed prefix", "[prod] ", "prod_"},
		{"other separators", "a-b", "a_b"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Presets: config.Presets{NamePrefix: test.prefix},
					Resources: config.Resources{
						Schemas: map[string]*resources.Schema{
							"qc": {CreateSchema: catalog.CreateSchema{CatalogName: "my_catalog", Name: "qc"}},
						},
						RegisteredModels: map[string]*resources.RegisteredModel{
							"model": {CreateRegisteredModelRequest: catalog.CreateRegisteredModelRequest{Name: "model"}},
						},
						ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
							"endpoint": {CreateServingEndpoint: serving.CreateServingEndpoint{Name: "endpoint"}},
						},
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, ApplyPresets())
			require.NoError(t, diags.Error())
			assert.Equal(t, test.wantPrefix+"qc", b.Config.Resources.Schemas["qc"].Name)
			assert.Equal(t, test.wantPrefix+"model", b.Config.Resources.RegisteredModels["model"].Name)
			assert.Equal(t, test.wantPrefix+"endpoint", b.Config.Resources.ModelServingEndpoints["endpoint"].Name)
		})
	}
}
