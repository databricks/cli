package metricviews

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestParameterV11TriState(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		present bool
		isNull  bool
		expr    string
	}{
		{"absent", "name: p\ndata_type: INT\n", false, false, ""},
		{"null", "name: p\ndata_type: STRING\ndefault: null\n", true, true, ""},
		{"tilde", "name: p\ndata_type: STRING\ndefault: ~\n", true, true, ""},
		{"expr", "name: p\ndata_type: STRING\ndefault: EMEA\n", true, false, "EMEA"},
		{"quoted null", "name: p\ndata_type: STRING\ndefault: \"null\"\n", true, false, "null"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p ParameterV11
			require.NoError(t, yaml.Unmarshal([]byte(tc.in), &p))
			assert.Equal(t, tc.present, p.Default.Present)
			assert.Equal(t, tc.isNull, p.Default.IsNull)
			assert.Equal(t, tc.expr, p.Default.Expr)

			out, err := yaml.Marshal(&p)
			require.NoError(t, err)
			var p2 ParameterV11
			require.NoError(t, yaml.Unmarshal(out, &p2))
			assert.Equal(t, p, p2)
		})
	}
}

func TestParameterV11ResolvesAliases(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantName string
		wantType string
		wantExpr string
	}{
		{"name", "data_type: &type STRING\nname: *type\ndefault: value\n", "STRING", "STRING", "value"},
		{"data_type", "name: &n p\ndata_type: *n\ndefault: value\n", "p", "p", "value"},
		{"default", "name: &n p\ndata_type: STRING\ndefault: *n\n", "p", "STRING", "p"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p ParameterV11
			require.NoError(t, yaml.Unmarshal([]byte(tc.in), &p))
			assert.Equal(t, tc.wantName, p.Name)
			assert.Equal(t, tc.wantType, p.DataType)
			assert.Equal(t, ParamDefault{Present: true, Expr: tc.wantExpr}, p.Default)
		})
	}
}

func TestValidateRejectsDuplicateParameterFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"name", "name: first\nname: second\ndata_type: STRING\n"},
		{"data_type", "name: p\ndata_type: STRING\ndata_type: INT\n"},
		{"default", "name: p\ndata_type: STRING\ndefault: null\ndefault: value\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := "version: '1.1'\nsource: main.sales.orders\nparameters:\n  - " + strings.ReplaceAll(strings.TrimSuffix(tc.in, "\n"), "\n", "\n    ") + "\n"
			err := validateYAML(t, []byte(in))
			require.ErrorContains(t, err, tc.name)
		})
	}
}

func TestParameterV11UnmarshalAllowsDuplicateFields(t *testing.T) {
	var p ParameterV11
	require.NoError(t, yaml.Unmarshal([]byte("name: first\nname: second\ndata_type: STRING\n"), &p))
	assert.Equal(t, "second", p.Name)
	require.NoError(t, yaml.Unmarshal([]byte("name: p\ndata_type: STRING\ndefault: null\ndefault: value\n"), &p))
	assert.Equal(t, ParamDefault{Present: true, Expr: "value"}, p.Default)
}

func TestParameterV10CollapsesNull(t *testing.T) {
	var p ParameterV10
	require.NoError(t, yaml.Unmarshal([]byte("name: p\ndata_type: STRING\ndefault: null\n"), &p))
	assert.Nil(t, p.Default)
	require.NoError(t, yaml.Unmarshal([]byte("name: p\ndata_type: STRING\ndefault: EMEA\n"), &p))
	require.NotNil(t, p.Default)
	assert.Equal(t, "EMEA", *p.Default)
}
