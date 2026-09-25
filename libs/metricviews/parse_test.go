package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
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

func TestValidateRejectsBadVersions(t *testing.T) {
	for _, in := range []string{
		"version: 2.0\nsource: main.default.t\n",
		"version: 9.9\nsource: main.default.t\n",
	} {
		err := Validate([]byte(in))
		assert.Error(t, err)
	}
}

func TestParseAllowsStructurallyInvalidYAML(t *testing.T) {
	for name, in := range map[string]string{
		"missing source":        "version: '1.1'\ndimensions: [{name: id, expr: id}]\n",
		"unknown view type":     "version: '1.1'\nview_type: OTHER\nsource: main.sales.orders\n",
		"unknown version":       "version: '2.0'\nsource: main.sales.orders\n",
		"missing column expr":   "version: '1.1'\nsource: main.sales.orders\ndimensions: [{name: id}]\n",
		"invalid format":        "version: '1.1'\nsource: main.sales.orders\ndimensions: [{name: id, expr: id, format: {type: NUMBER}}]\n",
		"conflicting aliases":   "version: '1.1'\nsource: main.sales.orders\ndimensions: []\nfields: [{name: id, expr: id}]\n",
		"missing discriminator": "version: '1.1'\nsources: [{name: orders, from: main.sales.orders}]\n",
		"duplicate parameter":   "version: '1.1'\nsource: main.sales.orders\nparameters: [{name: first, name: second, data_type: STRING}]\n",
	} {
		t.Run(name, func(t *testing.T) {
			m, err := Parse([]byte(in))
			require.NoError(t, err)
			require.NotNil(t, m)
		})
	}
}

func TestParseRejectsDuplicateYAMLMappingKeys(t *testing.T) {
	_, err := Parse([]byte("version: '1.1'\nsource: first\nsource: second\n"))
	require.ErrorContains(t, err, "mapping key")
}

func TestParseAndValidateChecksInput(t *testing.T) {
	valid := []byte("version: '1.1'\nsource: main.sales.orders\ndimensions: [{name: id, expr: id}]\n")
	m, err := ParseAndValidate(valid)
	require.NoError(t, err)
	require.NotNil(t, m.SingleSource)

	invalid := []byte("version: '1.1'\ndimensions: [{name: id, expr: id}]\n")
	m, err = ParseAndValidate(invalid)
	require.ErrorContains(t, err, "source")
	assert.Nil(t, m)
}

func TestParseAndValidateResolvesDiscriminatorAliases(t *testing.T) {
	in := []byte("schema: &schema '1.1'\nversion: *schema\nkind: &kind MULTI_SOURCE\nview_type: *kind\nsources: [{name: orders, from: main.sales.orders}]\n")
	m, err := ParseAndValidate(in)
	require.NoError(t, err)
	require.NotNil(t, m.MultiSource)
	assert.Equal(t, "1.1", m.Version())
}

func TestParseAndValidateResolvesColumnAliases(t *testing.T) {
	in := []byte("column: &column {name: id, expr: id}\nversion: '1.1'\nsource: main.sales.orders\ndimensions: [*column]\n")
	m, err := ParseAndValidate(in)
	require.NoError(t, err)
	require.Len(t, m.SingleSource.Dimensions, 1)
	assert.Equal(t, "id", *m.SingleSource.Dimensions[0].Name)
}

func TestValidateRejectsUnsupportedViewType(t *testing.T) {
	in := []byte("version: 1.1\nview_type: OTHER\nsource: main.default.t\n")
	err := Validate(in)
	require.Error(t, err)
	assert.ErrorContains(t, err, "view_type")
	assert.ErrorContains(t, err, "OTHER")
}

func TestParseExplicitSingleSourceViewTypeRoundTrip(t *testing.T) {
	in := []byte("version: \"1.1\"\nview_type: SINGLE_SOURCE\nsource: main.sales.orders\ndimensions:\n  - name: id\n    expr: id\n")
	m, err := Parse(in)
	require.NoError(t, err)
	require.NotNil(t, m.SingleSource)

	out, err := yaml.Marshal(m)
	require.NoError(t, err)
	assert.Contains(t, string(out), "view_type: SINGLE_SOURCE\n")

	reparsed, err := Parse(out)
	require.NoError(t, err)
	require.NotNil(t, reparsed.SingleSource)
	assert.Equal(t, m.SingleSource.Source, reparsed.SingleSource.Source)
}

func TestValidateRejectsMultiSourceWithoutViewType(t *testing.T) {
	in := []byte("version: 1.1\nsources:\n  - name: o\n    from: main.default.o\n")
	err := Validate(in)
	require.Error(t, err)
	assert.ErrorContains(t, err, "view_type")
}

func TestMetricViewMarshalRoundTrips(t *testing.T) {
	in := []byte("version: 1.1\nsource: main.default.t\nmeasures:\n    - name: c\n      expr: COUNT(*)\n")
	m, err := Parse(in)
	require.NoError(t, err)
	out, err := m.MarshalYAML()
	require.NoError(t, err)
	assert.NotNil(t, out)
}
