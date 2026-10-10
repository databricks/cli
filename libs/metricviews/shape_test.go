package metricviews

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestValidateRejectsMissingRequiredKeys(t *testing.T) {
	cases := map[string]string{
		"v1.0 source":              "version: '1.0'\ndimensions: [{name: id, expr: id}]\n",
		"v1.0 column name":         "version: '1.0'\nsource: main.sales.orders\ndimensions: [{expr: id}]\n",
		"v1.0 column expr":         "version: '1.0'\nsource: main.sales.orders\ndimensions: [{name: id}]\n",
		"v1.1 source":              "version: '1.1'\ndimensions: [{name: id, expr: id}]\n",
		"v1.1 empty source":        "version: '1.1'\nsource: ''\ndimensions: [{name: id, expr: id}]\n",
		"v1.1 whitespace source":   "version: '1.1'\nsource: '  '\ndimensions: [{name: id, expr: id}]\n",
		"v1.1 null source":         "version: '1.1'\nsource: null\ndimensions: [{name: id, expr: id}]\n",
		"v1.1 column expr":         "version: '1.1'\nsource: main.sales.orders\ndimensions: [{name: id}]\n",
		"join name":                "version: '1.1'\nsource: main.sales.orders\njoins: [{source: main.sales.items}]\n",
		"join source":              "version: '1.1'\nsource: main.sales.orders\njoins: [{name: items}]\n",
		"v1.0 parameter name":      "version: '1.0'\nsource: main.sales.orders\nparameters: [{data_type: STRING}]\n",
		"v1.0 parameter type":      "version: '1.0'\nsource: main.sales.orders\nparameters: [{name: p}]\n",
		"v1.1 parameter name":      "version: '1.1'\nsource: main.sales.orders\nparameters: [{data_type: STRING}]\n",
		"v1.1 parameter type":      "version: '1.1'\nsource: main.sales.orders\nparameters: [{name: p}]\n",
		"multi-source sources":     "version: '1.1'\nview_type: MULTI_SOURCE\ndimensions: [{name: id, expr: id}]\n",
		"source node name":         "version: '1.1'\nview_type: MULTI_SOURCE\nsources: [{from: main.sales.orders}]\n",
		"source node from":         "version: '1.1'\nview_type: MULTI_SOURCE\nsources: [{name: orders}]\n",
		"relationship ref":         "version: '1.1'\nview_type: MULTI_SOURCE\nsources: [{name: orders, from: main.sales.orders, relationships: [{on: orders.id = items.id}]}]\n",
		"window order":             "version: '1.1'\nsource: main.sales.orders\nmeasures: [{name: total, expr: SUM(amount), window: [{semiadditive: last, range: current}]}]\n",
		"window semiadditive":      "version: '1.1'\nsource: main.sales.orders\nmeasures: [{name: total, expr: SUM(amount), window: [{order: id, range: current}]}]\n",
		"window range":             "version: '1.1'\nsource: main.sales.orders\nmeasures: [{name: total, expr: SUM(amount), window: [{order: id, semiadditive: last}]}]\n",
		"materialization mode":     "version: '1.1'\nsource: main.sales.orders\nmaterialization: {schedule: EVERY 1 HOUR, materialized_views: []}\n",
		"materialization schedule": "version: '1.1'\nsource: main.sales.orders\nmaterialization: {mode: fresh, materialized_views: []}\n",
		"materialization views":    "version: '1.1'\nsource: main.sales.orders\nmaterialization: {schedule: EVERY 1 HOUR, mode: fresh}\n",
		"materialized view name":   "version: '1.1'\nsource: main.sales.orders\nmaterialization: {schedule: EVERY 1 HOUR, mode: fresh, materialized_views: [{type: aggregated}]}\n",
		"materialized view type":   "version: '1.1'\nsource: main.sales.orders\nmaterialization: {schedule: EVERY 1 HOUR, mode: fresh, materialized_views: [{name: mv}]}\n",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateInput(t, []byte(in))
			require.Error(t, err)
		})
	}
}

func TestValidateAllowsV11WildcardWithoutName(t *testing.T) {
	err := validateInput(t, []byte("version: '1.1'\nsource: main.sales.orders\ndimensions: [{expr: 'source.*'}]\n"))
	require.NoError(t, err)
}

func TestValidateRejectsConflictingDimensionKeys(t *testing.T) {
	for _, in := range []string{
		"version: '1.0'\nsource: main.sales.orders\ndimensions: []\nfields: [{name: id, expr: id}]\n",
		"version: '1.1'\nsource: main.sales.orders\ndimensions: [{name: id, expr: id}]\nfields: []\n",
		"version: '1.1'\nview_type: MULTI_SOURCE\nsources: [{name: orders, from: main.sales.orders}]\ndimensions: []\nfields: [{name: id, expr: id}]\n",
		"version: '1.1'\nsource: main.sales.orders\nmaterialization:\n  schedule: EVERY 1 HOUR\n  mode: fresh\n  materialized_views:\n    - name: mv\n      type: aggregated\n      dimensions: []\n      fields: [id]\n",
	} {
		err := validateInput(t, []byte(in))
		require.ErrorContains(t, err, "dimensions")
		require.ErrorContains(t, err, "fields")
	}
}

func TestValidateRejectsInvalidFormatVariants(t *testing.T) {
	for _, in := range []string{
		"type: NUMBER\n",
		"type: other\n",
		"type: currency\n",
		"type: currency\ncurrency_code: null\n",
		"type: date\n",
		"type: date_time\ndate_format: year_month_day\n",
		"type: date_time\ntime_format: locale_hour_minute\n",
		"type: number\ncurrency_code: USD\n",
		"type: number\nunknown_option: value\n",
		"type: number\ndecimal_places: {places: 2}\n",
	} {
		spec := "version: '1.1'\nsource: main.sales.orders\ndimensions:\n  - name: amount\n    expr: amount\n    format:\n      " + strings.ReplaceAll(strings.TrimSuffix(in, "\n"), "\n", "\n      ") + "\n"
		err := validateInput(t, []byte(spec))
		require.Error(t, err, "input: %s", in)
	}
}

func TestValidateRejectsIncompleteFormat(t *testing.T) {
	in := []byte("version: '1.1'\nsource: main.sales.orders\ndimensions:\n  - name: amount\n    expr: amount\n    format: {type: currency}\n")
	err := validateInput(t, in)
	require.ErrorContains(t, err, "currency_code")
}

func TestParseV10PreservesColumnMetadata(t *testing.T) {
	in := []byte("version: '1.0'\nsource: main.sales.orders\ndimensions:\n  - name: order_date\n    expr: order_date\n    comment: Order date\n    display_name: Order Date\n    synonyms: [Purchase Date]\n    format: {type: date, date_format: year_month_day}\n")
	m, err := Parse(in)
	require.NoError(t, err)
	require.Len(t, m.V10.Dimensions, 1)

	out, err := yaml.Marshal(m)
	require.NoError(t, err)
	assert.Contains(t, string(out), "display_name: Order Date")
	assert.Contains(t, string(out), "comment: Order date")
	assert.Contains(t, string(out), "synonyms:")
	assert.Contains(t, string(out), "date_format: year_month_day")

	reparsed, err := Parse(out)
	require.NoError(t, err)
	assert.Equal(t, m.V10.Dimensions, reparsed.V10.Dimensions)
}
