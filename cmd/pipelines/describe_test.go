package pipelines

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"3fb8e5a1-0d2c-4a6b-9f1e-2c7d8e9f0a1b", true},
		{"my_pipeline", false},
		{"my-pipeline", false},
		{"3FB8E5A1-0D2C-4A6B-9F1E-2C7D8E9F0A1B", false}, // uppercase is treated as a KEY
		{"3fb8e5a1-0d2c-4a6b-9f1e-2c7d8e9f0a1", false},  // too short
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, LooksLikeUUID(tt.in), "looksLikeUUID(%q)", tt.in)
	}
}

// The populated render is golden-tested in acceptance/pipelines/describe/basic;
// this covers only the has-never-run branch.
func TestPipelineDescribeTemplateNoRuns(t *testing.T) {
	data := pipelineDescribeData{
		Pipeline: &pipelines.GetPipelineResponse{
			Name:       "Fresh Pipeline",
			PipelineId: "def-456",
			Spec:       &pipelines.PipelineSpec{},
		},
	}

	ctx, out := cmdio.NewTestContextWithStdout(t.Context())
	require.NoError(t, cmdio.RenderWithTemplate(ctx, data, "", pipelineDescribeTemplate))

	got := out.String()
	assert.Contains(t, got, "Pipeline: Fresh Pipeline")
	assert.Contains(t, got, "No runs yet.")
	assert.NotContains(t, got, "Update ID:")
}
