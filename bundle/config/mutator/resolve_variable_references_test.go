package mutator

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveVariableReferencesDetectsNestedVarRef(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "${var.env_${var.region}}",
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesWithoutResources())
	// The nested reference won't resolve, but we should still detect it.
	_ = diags

	assert.Contains(t, b.Metrics.BoolValues, protos.BoolMapEntry{Key: "nested_var_reference_used", Value: true})
}

func TestResolveVariableReferencesNoNestedVarRef(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "${var.env}",
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesWithoutResources())
	_ = diags

	for _, entry := range b.Metrics.BoolValues {
		assert.NotEqual(t, "nested_var_reference_used", entry.Key)
	}
}

func TestResolveVariableReferencesWithSourceLinkedDeployment(t *testing.T) {
	testCases := []struct {
		enabled bool
		assert  func(t *testing.T, b *bundle.Bundle)
	}{
		{
			true,
			func(t *testing.T, b *bundle.Bundle) {
				// Variables that use workspace file path should have SyncRootValue during resolution phase
				require.Equal(t, "sync/root/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])

				// The file path itself should remain the same
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
		{
			false,
			func(t *testing.T, b *bundle.Bundle) {
				require.Equal(t, "file/path", b.Config.Resources.Pipelines["pipeline1"].Configuration["source"])
				require.Equal(t, "file/path", b.Config.Workspace.FilePath)
			},
		},
	}

	for _, testCase := range testCases {
		b := &bundle.Bundle{
			SyncRootPath: "sync/root/path",
			Config: config.Root{
				Presets: config.Presets{
					SourceLinkedDeployment: &testCase.enabled,
				},
				Workspace: config.Workspace{
					FilePath: "file/path",
				},
				Resources: config.Resources{
					Pipelines: map[string]*resources.Pipeline{
						"pipeline1": {
							CreatePipeline: pipelines.CreatePipeline{
								Configuration: map[string]string{
									"source": "${workspace.file_path}",
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesOnlyResources("workspace"))
		require.NoError(t, diags.Error())
		testCase.assert(t, b)
	}
}

func TestResolveVariableReferencesRoundsNoReferences(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "literal-name",
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesWithoutResources())
	require.NoError(t, diags.Error())

	// No references means a single round with no updates, so gt_1 should not be set.
	for _, entry := range b.Metrics.BoolValues {
		assert.NotEqual(t, "variable_resolution_rounds_gt_1", entry.Key)
		assert.NotEqual(t, "variable_resolution_rounds_gt_3", entry.Key)
	}
}

func TestResolveVariableReferencesRoundsGt1MultiRound(t *testing.T) {
	// Set up a chain: bundle.name -> var.a -> var.b -> literal.
	// This requires 2 rounds to fully resolve.
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				Name: "${var.a}",
			},
			Variables: map[string]*variable.Variable{
				"a": {
					Value: "${var.b}",
				},
				"b": {
					Value: "final",
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ResolveVariableReferencesWithoutResources())
	require.NoError(t, diags.Error())
	assert.Equal(t, "final", b.Config.Bundle.Name)

	assert.Contains(t, b.Metrics.BoolValues, protos.BoolMapEntry{Key: "variable_resolution_rounds_gt_1", Value: true})
	// 2 rounds should not trigger gt_3.
	for _, entry := range b.Metrics.BoolValues {
		assert.NotEqual(t, "variable_resolution_rounds_gt_3", entry.Key)
	}
}
