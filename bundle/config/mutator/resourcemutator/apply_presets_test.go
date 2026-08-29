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
