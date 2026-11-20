package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSingleSourceUnquotedVersionAndFieldsAlias(t *testing.T) {
	in := []byte("version: 1.1\nsource: cat.sch.t\nfields:\n    - name: m\n      expr: MONTH(d)\nmeasures:\n    - name: c\n      expr: COUNT(*)\n")
	var v SingleSourceMetricView
	require.NoError(t, yaml.Unmarshal(in, &v))
	assert.Equal(t, "1.1", v.Version)
	require.Len(t, v.Dimensions, 1)
	assert.Equal(t, "m", *v.Dimensions[0].Name)
}

func TestColumnV11WildcardNameAbsent(t *testing.T) {
	var c ColumnV11
	require.NoError(t, yaml.Unmarshal([]byte("expr: source.*\n"), &c))
	assert.Nil(t, c.Name)
	out, err := yaml.Marshal(&c)
	require.NoError(t, err)
	assert.NotContains(t, string(out), "name:")
}

func TestMultiSourceParsesSourcesAndRelationships(t *testing.T) {
	in := []byte("version: \"1.1\"\nview_type: MULTI_SOURCE\nsources:\n    - name: orders\n      from: cat.sch.orders\n      primary_key:\n        - k\n    - name: li\n      from: cat.sch.li\n      relationships:\n        - ref: orders\n          foreign_key:\n            - k\ndimensions:\n    - name: y\n      expr: YEAR(orders.d)\n")
	var v MultiSourceMetricView
	require.NoError(t, yaml.Unmarshal(in, &v))
	require.Len(t, v.Sources, 2)
	require.Len(t, v.Sources[1].Relationships, 1)
	assert.Equal(t, "orders", v.Sources[1].Relationships[0].RefSource)
}
