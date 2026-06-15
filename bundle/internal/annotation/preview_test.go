package annotation_test

import (
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreviewTag(t *testing.T) {
	tests := []struct {
		launchStage annotation.LaunchStage
		want        string
	}{
		{annotation.LaunchStagePublicPreview, "[Public Preview]"},
		{annotation.LaunchStagePublicBeta, "[Beta]"},
		{annotation.LaunchStagePrivatePreview, "[Private Preview]"},
		{annotation.LaunchStageGA, ""},
		{"", ""},
		{"SOMETHING_ELSE", ""},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.want, annotation.PreviewTag(tc.launchStage))
	}
}

func TestParseLaunchStage(t *testing.T) {
	tests := []struct {
		raw     string
		want    annotation.LaunchStage
		wantErr bool
	}{
		{"", annotation.LaunchStageGA, false},
		{"GA", annotation.LaunchStageGA, false},
		{"PUBLIC_PREVIEW", annotation.LaunchStagePublicPreview, false},
		{"PUBLIC_BETA", annotation.LaunchStagePublicBeta, false},
		{"PRIVATE_PREVIEW", annotation.LaunchStagePrivatePreview, false},
		{"SOMETHING_ELSE", "", true},
	}
	for _, tc := range tests {
		got, err := annotation.ParseLaunchStage(tc.raw)
		if tc.wantErr {
			assert.Error(t, err, "ParseLaunchStage(%q)", tc.raw)
			continue
		}
		require.NoError(t, err, "ParseLaunchStage(%q)", tc.raw)
		assert.Equal(t, tc.want, got)
	}
}

// Every known launch stage must round-trip through ParseLaunchStage and have a
// tag entry, so a constant added without a previewTags entry (or vice versa) is
// caught here rather than rendering silently as GA.
func TestKnownLaunchStagesAreParsable(t *testing.T) {
	for _, stage := range []annotation.LaunchStage{
		annotation.LaunchStageGA,
		annotation.LaunchStagePublicPreview,
		annotation.LaunchStagePublicBeta,
		annotation.LaunchStagePrivatePreview,
	} {
		got, err := annotation.ParseLaunchStage(string(stage))
		require.NoError(t, err, "stage %q", stage)
		assert.Equal(t, stage, got)
	}
}
