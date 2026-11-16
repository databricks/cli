package annotation

import (
	"testing"

	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/assert"
)

// No resource is overridden at the moment, so the test installs its own
// override to keep the raising and more-restrictive paths covered.
func TestOverrideLaunchStage(t *testing.T) {
	const overridden = "github.com/databricks/cli/bundle/config/resources.Job"
	saved := launchStageOverrides
	launchStageOverrides = map[string]clijson.LaunchStage{overridden: clijson.LaunchStagePublicBeta}
	t.Cleanup(func() { launchStageOverrides = saved })

	tests := []struct {
		name     string
		typePath string
		stage    clijson.LaunchStage
		want     clijson.LaunchStage
	}{
		{"GA field is raised to the override", overridden, "", clijson.LaunchStagePublicBeta},
		{"less-restrictive public-preview field is raised", overridden, clijson.LaunchStagePublicPreview, clijson.LaunchStagePublicBeta},
		{"Beta field is unchanged", overridden, clijson.LaunchStagePublicBeta, clijson.LaunchStagePublicBeta},
		{"more-restrictive private-preview field is kept", overridden, clijson.LaunchStagePrivatePreview, clijson.LaunchStagePrivatePreview},
		{"field of a type with no override is unchanged", "github.com/databricks/cli/bundle/config/resources.Pipeline", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, OverrideLaunchStage(tc.typePath, tc.stage))
		})
	}
}
