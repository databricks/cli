package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundleWithResources(t *testing.T, resources map[string]map[string]any) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{}
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		typed := make(map[string]dyn.Value)
		for resourceType, byName := range resources {
			named := make(map[string]dyn.Value)
			for name, r := range byName {
				dv, err := convert.FromTyped(r, dyn.NilValue)
				require.NoError(t, err)
				named[name] = dv
			}
			typed[resourceType] = dyn.NewValue(named, nil)
		}
		return dyn.Set(v, "resources", dyn.NewValue(typed, nil))
	})
	require.NoError(t, err)
	return b
}

func TestCollectResourceCountsAndSizes_GroupsByType(t *testing.T) {
	b := bundleWithResources(t, map[string]map[string]any{
		"jobs": {
			"a": map[string]any{"name": "a"},
			"b": map[string]any{"name": "ab"},
		},
		"pipelines": {
			"p": map[string]any{"name": "p1"},
		},
	})

	counts, sizes, fileSize := collectResourceCountsAndSizes(t.Context(), b)
	assert.Equal(t, map[string]int64{"jobs": 2, "pipelines": 1}, counts)
	assert.Len(t, sizes["jobs"], 2)
	assert.Len(t, sizes["pipelines"], 1)
	// fileSize is the simulated dstate.Database byte length, which includes
	// the envelope (state_version, cli_version, lineage, serial, the state
	// map wrapper, and per-entry __id__) on top of each resource's bytes.
	// So fileSize must exceed the per-resource sum.
	var sum int64
	for _, list := range sizes {
		for _, s := range list {
			assert.Positive(t, s)
			sum += s
		}
	}
	assert.Greater(t, fileSize, sum, "fileSize should include envelope overhead beyond per-resource bytes")
}

func TestCollectResourceCountsAndSizes_NoResources(t *testing.T) {
	b := &bundle.Bundle{}
	counts, sizes, total := collectResourceCountsAndSizes(t.Context(), b)
	assert.Empty(t, counts)
	assert.Empty(t, sizes)
	assert.Equal(t, int64(0), total)
}

func TestResolveDeployEngine(t *testing.T) {
	cases := []struct {
		name      string
		configEng engine.EngineType
		envEng    string
		want      string
	}{
		{"config wins over env", engine.EngineDirect, "terraform", "direct"},
		{"env used when config unset", engine.EngineNotSet, "direct", "direct"},
		{"default when neither set", engine.EngineNotSet, "", "terraform"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := &bundle.Bundle{}
			b.Config.Bundle.Engine = c.configEng
			ctx := env.Set(t.Context(), engine.EnvVar, c.envEng)
			assert.Equal(t, c.want, resolveDeployEngine(ctx, b))
		})
	}
}

func TestCollectResourcesMetadata_DeterministicAndComparable(t *testing.T) {
	// Same bundle config under both engines must produce identical metadata.
	resources := map[string]map[string]any{
		"jobs": {
			"a": map[string]any{"name": "alpha"},
			"b": map[string]any{"name": "beta"},
			"c": map[string]any{"name": "gamma"},
		},
		"pipelines": {
			"p": map[string]any{"name": "pipe"},
		},
	}
	bDirect := bundleWithResources(t, resources)
	bDirect.Config.Bundle.Engine = engine.EngineDirect
	bTerraform := bundleWithResources(t, resources)
	bTerraform.Config.Bundle.Engine = engine.EngineTerraform

	mdDirect := collectResourcesMetadata(t.Context(), bDirect)
	mdTerraform := collectResourcesMetadata(t.Context(), bTerraform)

	require.NotNil(t, mdDirect)
	require.NotNil(t, mdTerraform)
	assert.Equal(t, "direct", mdDirect.StateEngine)
	assert.Equal(t, "terraform", mdTerraform.StateEngine)
	assert.Equal(t, mdDirect.StateFileSizeBytes, mdTerraform.StateFileSizeBytes)
	assert.Equal(t, mdDirect.Resources, mdTerraform.Resources)
}

func TestCollectResourcesMetadata_ReturnsNilWhenNoResources(t *testing.T) {
	b := &bundle.Bundle{}
	assert.Nil(t, collectResourcesMetadata(t.Context(), b))
}

func TestStatHelpers(t *testing.T) {
	assert.Equal(t, int64(3), statMax([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMean([]int64{1, 2, 3}))
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3}))
	// Lower-middle for even count: sorted [1,2,3,4] -> index (4-1)/2 = 1 -> 2.
	assert.Equal(t, int64(2), statMedian([]int64{1, 2, 3, 4}))
	assert.Equal(t, int64(0), statMax(nil))
	assert.Equal(t, int64(0), statMean(nil))
	assert.Equal(t, int64(0), statMedian(nil))
}

func TestUnionKeys(t *testing.T) {
	got := unionKeys(map[string]int64{"a": 1, "b": 2}, map[string][]int64{"b": nil, "c": nil})
	assert.ElementsMatch(t, []string{"a", "b", "c"}, got)
}
