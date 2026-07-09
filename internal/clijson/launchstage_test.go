package clijson

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLaunchStage(t *testing.T) {
	tests := []struct {
		raw     string
		want    LaunchStage
		wantErr bool
	}{
		{"", LaunchStageGA, false},
		{"GA", LaunchStageGA, false},
		{"PUBLIC_PREVIEW", LaunchStagePublicPreview, false},
		{"PUBLIC_BETA", LaunchStagePublicBeta, false},
		{"PRIVATE_PREVIEW", LaunchStagePrivatePreview, false},
		{"SOMETHING_ELSE", "", true},
	}
	for _, tc := range tests {
		got, err := ParseLaunchStage(tc.raw)
		if tc.wantErr {
			assert.Error(t, err, "ParseLaunchStage(%q)", tc.raw)
			continue
		}
		require.NoError(t, err, "ParseLaunchStage(%q)", tc.raw)
		assert.Equal(t, tc.want, got)
	}
}

// Every constant must be a member of LaunchStages, so a constant added without
// updating the closed set (or vice versa) is caught here.
func TestLaunchStagesContainsEveryConst(t *testing.T) {
	for _, stage := range []LaunchStage{
		LaunchStageGA,
		LaunchStagePublicPreview,
		LaunchStagePublicBeta,
		LaunchStagePrivatePreview,
	} {
		got, err := ParseLaunchStage(string(stage))
		require.NoError(t, err, "stage %q", stage)
		assert.Equal(t, stage, got)
	}
}
