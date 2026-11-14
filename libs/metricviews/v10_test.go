package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestV10ParsesUnquotedVersion(t *testing.T) {
	in := []byte("version: 0.1\nsource: main.default.t\ndimensions:\n    - name: ID\n      expr: id\nmeasures:\n    - name: total\n      expr: SUM(v)\n")
	var v MetricViewV10
	require.NoError(t, yaml.Unmarshal(in, &v))
	assert.Equal(t, "0.1", v.Version)
	require.Len(t, v.Dimensions, 1)
	assert.Equal(t, "ID", v.Dimensions[0].Name)
}
