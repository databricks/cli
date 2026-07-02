package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectNullTelemetry(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected bool
	}{
		{
			name: "no null under targets",
			yaml: `
bundle:
  name: test
targets:
  dev:
    workspace:
      host: https://example.test
`,
			expected: false,
		},
		{
			name: "null map value under targets",
			yaml: `
targets:
  dev:
    resources:
`,
			expected: true,
		},
		{
			name: "null nested deep under targets",
			yaml: `
targets:
  dev:
    resources:
      jobs:
        my_job:
`,
			expected: true,
		},
		{
			name: "scalar null under targets is dropped by normalization",
			yaml: `
targets:
  dev:
    workspace:
      host:
`,
			expected: false,
		},
		{
			name: "null outside targets is ignored",
			yaml: `
targets:
  dev:
    workspace:
      host: https://example.test
variables:
  foo:
`,
			expected: false,
		},
		{
			name: "no targets section",
			yaml: `
bundle:
  name: test
`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, diags := config.LoadFromBytes("databricks.yml", []byte(tt.yaml))
			require.NoError(t, diags.Error())

			b := &bundle.Bundle{Config: *root}
			applyDiags := bundle.Apply(t.Context(), b, mutator.CollectNullTelemetry())
			require.Empty(t, applyDiags)

			assert.Equal(t, []protos.BoolMapEntry{
				{Key: "null-in-targets", Value: tt.expected},
			}, b.Metrics.BoolValues)
		})
	}
}
