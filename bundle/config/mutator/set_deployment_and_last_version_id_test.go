package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDeploymentAndLastVersionIDIsNoOpWhenNothingIsRecorded(t *testing.T) {
	// The mutator must not make the API call that would find a deployment when there is none to
	// report: recording off, an engine that does not record, or a bundle never deployed.
	cases := []struct {
		name         string
		engine       engine.EngineType
		experimental *config.Experimental
	}{
		{"experimental unset", engine.EngineDirect, nil},
		{"recording disabled", engine.EngineDirect, &config.Experimental{RecordDeploymentHistory: false}},
		{"terraform records nothing", engine.EngineTerraform, &config.Experimental{RecordDeploymentHistory: true}},
		{"never deployed", engine.EngineDirect, &config.Experimental{RecordDeploymentHistory: true}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Experimental: tc.experimental,
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, SetDeploymentAndLastVersionID(tc.engine))
			require.NoError(t, diags.Error())
			assert.Nil(t, b.Config.Bundle.Deployment.History)
		})
	}
}
