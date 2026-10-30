package metricviews

import (
	"os"
	"testing"

	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestNormalizeLowercasesEnums(t *testing.T) {
	in := []byte(`version: "1.1"
source: main.sales.orders
joins:
  - name: li
    source: main.sales.lineitem
    on: source.o_orderkey = li.l_orderkey
    cardinality: MANY_TO_ONE
dimensions:
  - name: d
    expr: o_orderpriority
    format:
      type: number
      decimal_places:
        type: max
        places: 2
measures:
  - name: m
    expr: SUM(o_totalprice)
    window:
      - order: d
        semiadditive: LAST
        range: CURRENT
materialization:
  schedule: EVERY 1 HOUR
  mode: FRESH
  materialized_views:
    - name: mv
      type: AGGREGATED
      dimensions: [d]
      measures: [m]
`)
	m, err := Parse(in)
	require.NoError(t, err)
	m.Normalize()

	ss := m.SingleSource
	require.NotNil(t, ss)
	assert.Equal(t, "many_to_one", *ss.Joins[0].Cardinality)
	assert.Equal(t, "number", ss.Dimensions[0].Format.Type)
	assert.Equal(t, DecimalPlacesType("max"), ss.Dimensions[0].Format.DecimalPlaces.Type)
	assert.Equal(t, "last", ss.Measures[0].Window[0].Semiadditive)
	assert.Equal(t, "current", ss.Measures[0].Window[0].Range)
	assert.Equal(t, "fresh", ss.Materialization.Mode)
	assert.Equal(t, "aggregated", ss.Materialization.MaterializedViews[0].MVType)
}

func TestNormalizeTrimsTrailingWhitespaceOnly(t *testing.T) {
	in := []byte(`version: "1.1"
source: main.sales.orders
filter: "o_totalprice > 1  "
measures:
  - name: m
    expr: "  sum( o_totalprice )  "
`)
	m, err := Parse(in)
	require.NoError(t, err)
	m.Normalize()

	ss := m.SingleSource
	require.NotNil(t, ss)
	// Leading and internal whitespace preserved; trailing trimmed.
	assert.Equal(t, "  sum( o_totalprice )", ss.Measures[0].Expr)
	assert.Equal(t, "o_totalprice > 1", *ss.Filter)
}

func TestNormalizeLeavesVerbatimFields(t *testing.T) {
	// version is not canonicalized; data_type case preserved; date_format casing preserved.
	in := []byte(`version: "1.0"
source: main.sales.orders
measures:
  - name: m
    expr: SUM(o_totalprice)
`)
	m, err := Parse(in)
	require.NoError(t, err)
	m.Normalize()
	assert.Equal(t, "1.0", m.V10.Version)

	in2 := []byte(`version: "1.1"
source: main.sales.orders
parameters:
  - name: p
    data_type: STRING
dimensions:
  - name: d
    expr: o_orderdate
    format:
      type: date
      date_format: Year_Month_Day
`)
	m2, err := Parse(in2)
	require.NoError(t, err)
	m2.Normalize()
	ss := m2.SingleSource
	require.NotNil(t, ss)
	assert.Equal(t, "STRING", ss.Parameters[0].DataType)
	assert.Equal(t, "Year_Month_Day", *ss.Dimensions[0].Format.DateFormat)
	assert.Equal(t, "date", ss.Dimensions[0].Format.Type)
}

func TestNormalizeEliminatesCasingAndWhitespaceDrift(t *testing.T) {
	upper := []byte(`version: "1.1"
source: main.sales.orders
measures:
  - name: m
    expr: "SUM(o_totalprice) "
    window:
      - order: d
        semiadditive: LAST
        range: CURRENT
`)
	lower := []byte(`version: "1.1"
source: main.sales.orders
measures:
  - name: m
    expr: "SUM(o_totalprice)"
    window:
      - order: d
        semiadditive: last
        range: current
`)
	a, err := Parse(upper)
	require.NoError(t, err)
	b, err := Parse(lower)
	require.NoError(t, err)
	assert.False(t, structdiff.IsEqual(a, b), "before normalize the views differ")

	a.Normalize()
	b.Normalize()
	assert.True(t, structdiff.IsEqual(a, b), "after normalize the views are equal")
}

func TestNormalizeIsIdempotent(t *testing.T) {
	for _, p := range fixturePaths(t) {
		data, err := os.ReadFile(p)
		require.NoError(t, err)
		m, err := Parse(data)
		require.NoError(t, err)

		m.Normalize()
		once, err := yaml.Marshal(m)
		require.NoError(t, err)
		m.Normalize()
		twice, err := yaml.Marshal(m)
		require.NoError(t, err)
		assert.Equal(t, string(once), string(twice))
	}
}
