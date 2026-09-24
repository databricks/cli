package metricviews

import (
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

func TestParameterV10CollapsesNull(t *testing.T) {
	var p ParameterV10
	require.NoError(t, yaml.Unmarshal([]byte("name: p\ndata_type: STRING\ndefault: null\n"), &p))
	assert.Nil(t, p.Default)
	require.NoError(t, yaml.Unmarshal([]byte("name: p\ndata_type: STRING\ndefault: EMEA\n"), &p))
	require.NotNil(t, p.Default)
	assert.Equal(t, "EMEA", *p.Default)
}
