package tableview

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderString(t *testing.T, columns []string, rows [][]string, width int) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, RenderStaticWithTruncation(cmdio.MockDiscard(t.Context()), &buf, columns, rows, width))
	return buf.String()
}

func TestRenderEmptyColumns(t *testing.T) {
	assert.Empty(t, renderString(t, nil, nil, 0))
}

func TestRenderNoCap(t *testing.T) {
	out := renderString(t, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}, 0)
	assert.Equal(t, "id  name\n--  -----\n1   alice\n2   bob\n\n2 rows\n", out)
}

func TestRenderFitsWideBudget(t *testing.T) {
	out := renderString(t, []string{"id", "name"}, [][]string{{"1", "alice"}}, 200)
	assert.Equal(t, "id  name\n--  -----\n1   alice\n\n1 rows\n", out)
}

func TestRenderExactFitBoundaryNotCropped(t *testing.T) {
	out := renderString(t, []string{"id", "name"}, [][]string{{"1", "alice"}}, 9)
	assert.NotContains(t, out, ellipsis)
	assert.NotContains(t, out, "showing")
}

func TestRenderCropsColumns(t *testing.T) {
	columns := []string{"a", "b", "c", "d", "e"}
	out := renderString(t, columns, [][]string{{"1", "2", "3", "4", "5"}}, 10)
	assert.Contains(t, out, ellipsis)
	assert.Regexp(t, `\(showing \d of 5 columns\)`, out)
	assert.Contains(t, out, "1 rows")
}

func TestRenderCropAlwaysShowsAtLeastOneColumn(t *testing.T) {
	out := renderString(t, []string{"a", "b", "c"}, [][]string{{"1", "2", "3"}}, 1)
	assert.Contains(t, out, "(showing 1 of 3 columns)")
}

func TestRenderSingleColumnNarrowNotCropped(t *testing.T) {
	out := renderString(t, []string{"name"}, [][]string{{"alice"}}, 4)
	assert.NotContains(t, out, ellipsis)
	assert.NotContains(t, out, "showing")
}

func TestRenderCapsColumnWidth(t *testing.T) {
	long := strings.Repeat("x", maxColumnWidth+10)
	out := renderString(t, []string{"c"}, [][]string{{long}}, 0)
	assert.Contains(t, out, strings.Repeat("x", maxColumnWidth-cmdio.Width(ellipsis))+ellipsis)
	assert.NotContains(t, out, long)
}

func TestRenderCellAtCapNotTruncated(t *testing.T) {
	exact := strings.Repeat("x", maxColumnWidth)
	out := renderString(t, []string{"c"}, [][]string{{exact}}, 0)
	assert.Contains(t, out, exact)
	assert.NotContains(t, out, ellipsis)
}

func TestRenderTruncatesWideGlyphCellSafely(t *testing.T) {
	out := renderString(t, []string{"c"}, [][]string{{strings.Repeat("世", 30)}}, 0)
	assert.True(t, utf8.ValidString(out))
	assert.Contains(t, out, ellipsis)
}

func TestRenderEmptyRows(t *testing.T) {
	out := renderString(t, []string{"id", "name"}, nil, 0)
	assert.Equal(t, "id  name\n--  ----\n\n0 rows\n", out)
}

func TestRenderBlankValue(t *testing.T) {
	out := renderString(t, []string{"a", "b"}, [][]string{{"", "x"}}, 0)
	assert.Equal(t, "a  b\n-  -\n   x\n\n1 rows\n", out)
}

func TestRenderRaggedRow(t *testing.T) {
	out := renderString(t, []string{"a", "b"}, [][]string{{"1"}}, 0)
	assert.Equal(t, "a  b\n-  -\n1  \n\n1 rows\n", out)
}

func TestFitColumns(t *testing.T) {
	tests := []struct {
		name      string
		colWidth  []int
		width     int
		wantShown int
		wantCrop  bool
	}{
		{"no cap", []int{5, 5, 5}, 0, 3, false},
		{"fits exactly", []int{2, 5}, 9, 2, false},
		{"one over", []int{2, 5}, 8, 1, true},
		{"tiny width shows one", []int{3, 3, 3}, 1, 1, true},
		{"single column never cropped", []int{5}, 4, 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shown, cropped := fitColumns(tc.colWidth, tc.width)
			assert.Equal(t, tc.wantShown, shown)
			assert.Equal(t, tc.wantCrop, cropped)
		})
	}
}

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		maxWidth int
		want     string
	}{
		{"under", "abc", 10, "abc"},
		{"exact", "abcde", 5, "abcde"},
		{"over", "abcdefgh", 5, "ab..."},
		{"width below ellipsis hard cuts", "abcdef", 2, "ab"},
		{"zero width", "abc", 0, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, truncateToWidth(tc.in, tc.maxWidth))
		})
	}
}

func TestDetectWidthNonFile(t *testing.T) {
	assert.Equal(t, 0, DetectWidth(&bytes.Buffer{}))
}
