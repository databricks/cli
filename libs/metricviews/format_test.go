package metricviews

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestColumnFormatVariants(t *testing.T) {
	cases := map[string]string{
		"number":     "type: number\ndecimal_places:\n    type: max\n    places: 2\nhide_group_separator: true\n",
		"currency":   "type: currency\ncurrency_code: USD\ndecimal_places:\n    type: exact\n    places: 2\n",
		"percentage": "type: percentage\ndecimal_places:\n    type: all\n",
		"byte":       "type: byte\n",
		"date":       "type: date\ndate_format: year_month_day\nleading_zeros: true\n",
		"date_time":  "type: date_time\ndate_format: year_month_day\ntime_format: locale_hour_minute\n",
	}
	for typ, in := range cases {
		t.Run(typ, func(t *testing.T) {
			var f ColumnFormat
			require.NoError(t, yaml.Unmarshal([]byte(in), &f))
			assert.Equal(t, typ, f.Type)
			out, err := yaml.Marshal(&f)
			require.NoError(t, err)
			var f2 ColumnFormat
			require.NoError(t, yaml.Unmarshal(out, &f2))
			assert.Equal(t, f, f2)
		})
	}
}

func TestValidateRejectsInvalidDecimalPlacesType(t *testing.T) {
	for _, typ := range []string{"MAX", "bogus"} {
		t.Run(typ, func(t *testing.T) {
			in := []byte("version: '1.1'\nsource: main.sales.orders\ndimensions:\n  - name: amount\n    expr: amount\n    format:\n      type: number\n      decimal_places:\n        type: " + typ + "\n        places: 2\n")
			err := validateYAML(t, in)
			require.ErrorContains(t, err, typ)
		})
	}
}

func TestFormatUnmarshalAllowsInvalidEnumValues(t *testing.T) {
	var format ColumnFormat
	require.NoError(t, yaml.Unmarshal([]byte("type: NUMBER\ndecimal_places: {type: MAX}\n"), &format))
	assert.Equal(t, "NUMBER", format.Type)
	require.NotNil(t, format.DecimalPlaces)
	assert.Equal(t, DecimalPlacesType("MAX"), format.DecimalPlaces.Type)
}
