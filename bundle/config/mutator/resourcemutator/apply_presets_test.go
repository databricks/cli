package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/catalog"
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

func TestApplyPresetsSchemaPrefixPreservesUnderscores(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		expected string
	}{
		{"trailing underscore", "my_user_", "my_user_qc"},
		{"repeated underscores", "my__user__", "my__user__qc"},
		{"bracketed underscores", "[qa__team] ", "qa__team_qc"},
		{"bracketed prefix", "[prod] ", "prod_qc"},
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
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, ApplyPresets())
			require.NoError(t, diags.Error())
			assert.Equal(t, test.expected, b.Config.Resources.Schemas["qc"].Name)
		})
	}

	b := &bundle.Bundle{
		Config: config.Root{
			Bundle:  config.Bundle{Mode: config.Development},
			Presets: config.Presets{Tags: map[string]string{"dev": "my__user"}},
			Workspace: config.Workspace{
				CurrentUser: &config.User{ShortName: "my__user"},
			},
			Resources: config.Resources{
				Schemas: map[string]*resources.Schema{
					"qc": {CreateSchema: catalog.CreateSchema{CatalogName: "my_catalog", Name: "qc"}},
				},
			},
		},
	}

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())
	assert.Equal(t, "dev_my_user_qc", b.Config.Resources.Schemas["qc"].Name)
}
