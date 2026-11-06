package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestJoinRoundTripNested(t *testing.T) {
	in := []byte("name: d\nsource: cat.sch.dim\non: source.k = d.k\nusing:\n    - k\njoins:\n    - name: d2\n      source: cat.sch.dim2\n      using:\n        - k2\ncardinality: many_to_one\n")
	var j Join
	require.NoError(t, yaml.Unmarshal(in, &j))
	assert.Equal(t, "d", j.Name)
	require.Len(t, j.Joins, 1)
	assert.Equal(t, "d2", j.Joins[0].Name)
	require.NotNil(t, j.Cardinality)
	assert.Equal(t, "many_to_one", *j.Cardinality)
}

func TestMaterializedViewFieldsAlias(t *testing.T) {
	in := []byte("name: mv\ntype: aggregated\nfields:\n    - month\nmeasures:\n    - total\n")
	var mv MaterializedView
	require.NoError(t, yaml.Unmarshal(in, &mv))
	assert.Equal(t, []string{"month"}, mv.Dimensions)
	assert.Equal(t, "aggregated", mv.MVType)
}

func TestWindowSpecRoundTrip(t *testing.T) {
	in := []byte("order: month\nsemiadditive: last\nrange: trailing 3 period\n")
	var w WindowSpec
	require.NoError(t, yaml.Unmarshal(in, &w))
	out, err := yaml.Marshal(&w)
	require.NoError(t, err)
	var w2 WindowSpec
	require.NoError(t, yaml.Unmarshal(out, &w2))
	assert.Equal(t, w, w2)
}
