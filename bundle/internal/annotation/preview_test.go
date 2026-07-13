package annotation_test

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/databricks/cli/internal/clijson"
	"github.com/stretchr/testify/assert"
)

func TestPreviewTag(t *testing.T) {
	tests := []struct {
		launchStage clijson.LaunchStage
		want        string
	}{
		{clijson.LaunchStagePublicPreview, "[Public Preview]"},
		{clijson.LaunchStagePublicBeta, "[Beta]"},
		{clijson.LaunchStagePrivatePreview, "[Private Preview]"},
		{clijson.LaunchStageGA, ""},
		{"", ""},
		{"SOMETHING_ELSE", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, annotation.PreviewTag(tc.launchStage))
	}
}

// Every stage in the contract's closed set must have a tag entry, so a stage
// added to clijson without a tag here is caught rather than rendering blank.
// GA intentionally maps to the empty prefix; every other stage must be tagged.
func TestPreviewTagCoversAllStages(t *testing.T) {
	for _, stage := range clijson.LaunchStages {
		if stage == clijson.LaunchStageGA {
			assert.Empty(t, annotation.PreviewTag(stage))
			continue
		}
		assert.NotEmpty(t, annotation.PreviewTag(stage), "stage %q has no preview tag", stage)
	}
}
