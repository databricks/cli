package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	registerTargetModeExamples("pipelines", func(r *config.Resources) {
		r.Pipelines = map[string]*resources.Pipeline{
			"pipeline1": {CreatePipeline: pipelines.CreatePipeline{Name: "pipeline1", Continuous: true}}, //nolint:staticcheck // SA1019: pipeline continuous is deprecated in the SDK but remains a supported bundle config field
		}
	})
}

func TestPipelinesDevelopmentDisabled(t *testing.T) {
	b := mockBundle(config.Development)
	notEnabled := false
	b.Config.Presets.PipelinesDevelopment = &notEnabled

	diags := bundle.ApplySeq(t.Context(), b, ApplyTargetMode(), ApplyPresets())
	require.NoError(t, diags.Error())

	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].Development) //nolint:staticcheck // SA1019: pipeline development is deprecated in the SDK but remains a supported bundle config field
}
