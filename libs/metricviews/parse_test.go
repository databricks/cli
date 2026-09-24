package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDispatch(t *testing.T) {
	v10 := []byte("version: 1.0\nsource: main.default.t\nmeasures:\n    - name: c\n      expr: COUNT(*)\n")
	m, err := Parse(v10)
	require.NoError(t, err)
	require.NotNil(t, m.V10)
	assert.Nil(t, m.SingleSource)
	assert.Equal(t, "1.0", m.Version())

	single := []byte("version: 1.1\nsource: main.default.t\nmeasures:\n    - name: c\n      expr: COUNT(*)\n")
	m, err = Parse(single)
	require.NoError(t, err)
	require.NotNil(t, m.SingleSource)

	multi := []byte("version: \"1.1\"\nview_type: MULTI_SOURCE\nsources:\n    - name: o\n      from: main.default.o\ndimensions:\n    - name: y\n      expr: YEAR(o.d)\n")
	m, err = Parse(multi)
	require.NoError(t, err)
	require.NotNil(t, m.MultiSource)
}

func TestParseRejectsBadVersions(t *testing.T) {
	for _, in := range []string{
		"version: 2.0\nsource: main.default.t\n",
		"version: 9.9\nsource: main.default.t\n",
	} {
		_, err := Parse([]byte(in))
		assert.Error(t, err)
	}
}

func TestMetricViewMarshalRoundTrips(t *testing.T) {
	in := []byte("version: 1.1\nsource: main.default.t\nmeasures:\n    - name: c\n      expr: COUNT(*)\n")
	m, err := Parse(in)
	require.NoError(t, err)
	out, err := m.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, out)
}
