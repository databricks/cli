package agentstream

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderChart_SparseRowsDoesNotPanic(t *testing.T) {
	viz := &VizEvent{
		Spec: &VizSpec{
			Title:      "Sparse rows",
			WidgetType: "bar",
			XField:     "name",
			YFields:    []string{"total"},
		},
		Data: &TableData{
			Columns: []string{"name", "total"},
			Rows: [][]string{
				{"Alpha", "10"},
				{"Beta"},
			},
		},
	}

	var buf bytes.Buffer
	require.NotPanics(t, func() {
		RenderChart(&buf, viz, 80)
	})
	assert.Contains(t, buf.String(), "Sparse rows")
}

func TestRenderChart_LongFormatSparseRowsDoesNotPanic(t *testing.T) {
	viz := &VizEvent{
		Spec: &VizSpec{
			Title:      "Long sparse rows",
			WidgetType: "line",
			XField:     "month",
			YFields:    []string{"total"},
			ColorField: "region",
		},
		Data: &TableData{
			Columns: []string{"month", "region", "total"},
			Rows: [][]string{
				{"Jan", "US", "10"},
				{"Feb"},
				{"Mar", "EU", "-5"},
			},
		},
	}

	var buf bytes.Buffer
	require.NotPanics(t, func() {
		RenderChart(&buf, viz, 80)
	})
	assert.Contains(t, buf.String(), "Long sparse rows")
}

func TestParseFloatRejectsNonFiniteValues(t *testing.T) {
	assert.Equal(t, 0.0, parseFloat("NaN"))
	assert.Equal(t, 0.0, parseFloat("+Inf"))
	assert.Equal(t, 0.0, parseFloat("not a number"))
	assert.Equal(t, 1234.5, parseFloat("1,234.5"))
}

func TestComputeYLabelsUsesMinimum(t *testing.T) {
	assert.Equal(t, []string{"10", "0", "-10"}, computeYLabels(-10, 10, 3))
}
