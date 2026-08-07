package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitializeDeploymentHistoryIsNoOpWithoutRecording(t *testing.T) {
	// Without recording there is no deployment to report, and the mutator must not
	// make the API calls that would find one.
	cases := []struct {
		name         string
		experimental *config.Experimental
	}{
		{"experimental unset", nil},
		{"recording disabled", &config.Experimental{RecordDeploymentHistory: false}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Experimental: tc.experimental,
				},
			}

			diags := bundle.ApplySeq(t.Context(), b, InitializeDeploymentHistory())
			require.NoError(t, diags.Error())
			assert.Nil(t, b.Config.Bundle.Deployment.History)
		})
	}
}
