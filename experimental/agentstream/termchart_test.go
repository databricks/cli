package agentstream

import (
	"bytes"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func barViz(rows [][]string) *VizEvent {
	return &VizEvent{
		Spec: &VizSpec{
			Title:      "Total by Franchise",
			WidgetType: widgetBar,
			XField:     "name",
			YFields:    []string{"total"},
		},
		Data: &TableData{
			Columns: []string{"name", "total"},
			Rows:    rows,
		},
	}
}

func TestRenderChart_BarContent(t *testing.T) {
	var buf bytes.Buffer
	ok := RenderChart(&buf, barViz([][]string{{"Alpha", "100"}, {"Beta", "1250"}}), 80, false)
	require.True(t, ok)

	out := buf.String()
	assert.NotContains(t, out, "\033[", "color disabled must not emit ANSI escapes")
	assert.Contains(t, out, "Total by Franchise")
	assert.Contains(t, out, "Alpha")
	assert.Contains(t, out, "Beta")
	assert.Contains(t, out, "█")
	assert.Contains(t, out, "1,250")
}

func TestRenderChart_ColorEmitsANSI(t *testing.T) {
	var buf bytes.Buffer
	require.True(t, RenderChart(&buf, barViz([][]string{{"Alpha", "100"}}), 80, true))
	assert.Contains(t, buf.String(), "\033[")
}

func TestRenderChart_UnparseableRowsDropped(t *testing.T) {
	var buf bytes.Buffer
	ok := RenderChart(&buf, barViz([][]string{{"Alpha", "100"}, {"Beta", "N/A"}}), 80, false)
	require.True(t, ok)

	out := buf.String()
	assert.Contains(t, out, "Alpha")
	assert.NotContains(t, out, "Beta", "a row that cannot be plotted must not show as a zero bar")
}

func TestRenderChart_NegativeValues(t *testing.T) {
	var buf bytes.Buffer
	ok := RenderChart(&buf, barViz([][]string{{"Alpha", "-100"}, {"Beta", "-640"}}), 80, false)
	require.True(t, ok)

	out := buf.String()
	// Bars scale by absolute value; the signed numbers carry direction.
	assert.Contains(t, out, "█")
	assert.Contains(t, out, "-100")
	assert.Contains(t, out, "-640")
}

func TestRenderChart_NothingPlottable(t *testing.T) {
	var buf bytes.Buffer
	assert.False(t, RenderChart(&buf, barViz([][]string{{"Alpha", "N/A"}}), 80, false))
	assert.Empty(t, buf.String())
}

func TestRenderChart_YFieldMissingFromColumns(t *testing.T) {
	viz := barViz([][]string{{"Alpha", "100"}})
	viz.Spec.YFields = []string{"missing"}
	var buf bytes.Buffer
	assert.False(t, RenderChart(&buf, viz, 80, false))
	assert.Empty(t, buf.String())
}

func TestRenderChart_XFieldMissingFromColumns(t *testing.T) {
	// A spec/column mismatch must not leave a title-only chart behind: the
	// caller relies on the false return to print a placeholder instead.
	viz := barViz([][]string{{"Alpha", "100"}})
	viz.Spec.XField = "missing"
	var buf bytes.Buffer
	assert.False(t, RenderChart(&buf, viz, 80, false))
	assert.Empty(t, buf.String())
}

func TestRenderChart_SparseRowsDoesNotPanic(t *testing.T) {
	viz := barViz([][]string{
		{"Alpha", "10"},
		{"Beta"},
	})
	viz.Spec.Title = "Sparse rows"

	var buf bytes.Buffer
	require.NotPanics(t, func() {
		RenderChart(&buf, viz, 80, false)
	})
	assert.Contains(t, buf.String(), "Sparse rows")
}

func TestRenderChart_LongFormatSparseRowsDoesNotPanic(t *testing.T) {
	viz := &VizEvent{
		Spec: &VizSpec{
			Title:      "Long sparse rows",
			WidgetType: widgetLine,
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
		RenderChart(&buf, viz, 80, false)
	})
	assert.Contains(t, buf.String(), "Long sparse rows")
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		in   string
		want float64
		ok   bool
	}{
		{"NaN", 0, false},
		{"+Inf", 0, false},
		{"not a number", 0, false},
		{"", 0, false},
		{"1,234.5", 1234.5, true},
		{"-5", -5, true},
		{"0", 0, true},
	}
	for _, tc := range tests {
		v, ok := parseFloat(tc.in)
		assert.Equal(t, tc.ok, ok, tc.in)
		assert.InDelta(t, tc.want, v, 0, tc.in)
	}
}

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "123", formatNumber(123))
	assert.Equal(t, "1,000", formatNumber(1000))
	assert.Equal(t, "1,234,567", formatNumber(1234567))
	assert.Equal(t, "-1,234,567", formatNumber(-1234567))
	assert.Equal(t, "-1234.6", formatNumber(-1234.56))
}

func TestTruncateRunes(t *testing.T) {
	assert.Equal(t, "short", truncateRunes("short", 30))
	assert.Equal(t, "exactlyten", truncateRunes("exactlyten", 10))
	got := truncateRunes("ααααααααααααα", 10)
	assert.Equal(t, "ααααααα...", got)
	assert.True(t, utf8.ValidString(got))
}

func TestComputeYLabelsUsesMinimum(t *testing.T) {
	assert.Equal(t, []string{"10", "0", "-10"}, computeYLabels(-10, 10, 3))
}
