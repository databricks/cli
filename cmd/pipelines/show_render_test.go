package pipelines

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderTableString(t *testing.T, columns []string, rows [][]string, width int) string {
	t.Helper()
	var buf bytes.Buffer
	io := cmdio.NewIO(t.Context(), flags.OutputText, nil, &buf, &buf, "", "")
	ctx := cmdio.InContext(t.Context(), io)
	require.NoError(t, renderTable(ctx, columns, rows, width))
	return buf.String()
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"id":"1","name":"alice"},{"id":"2","name":"bob"}]`, buf.String())
}

func TestRenderJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"id"}, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, buf.String())
}

func TestRenderJSONPreservesColumnOrder(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"name", "id"}, [][]string{{"alice", "1"}})
	require.NoError(t, err)
	assert.Equal(t, "[\n  {\n    \"name\": \"alice\",\n    \"id\": \"1\"\n  }\n]\n", buf.String())
}

func TestRenderJSONEscapesValues(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"c"}, [][]string{{`a"b	c`}})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"c":"a\"b\tc"}]`, buf.String())
}

func TestRenderTableNoCap(t *testing.T) {
	out := renderTableString(t, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}, 0)
	assert.Equal(t, "id  name\n--  -----\n1   alice\n2   bob\n\n2 rows\n", out)
}

func TestRenderTableEmptyRows(t *testing.T) {
	out := renderTableString(t, []string{"id", "name"}, nil, 0)
	assert.Equal(t, "id  name\n--  ----\n\n0 rows\n", out)
}

func TestRenderTableRaggedRow(t *testing.T) {
	out := renderTableString(t, []string{"a", "b"}, [][]string{{"1"}}, 0)
	assert.Equal(t, "a  b\n-  -\n1  \n\n1 row\n", out)
}

func TestRenderTableBlankValue(t *testing.T) {
	out := renderTableString(t, []string{"a", "b"}, [][]string{{"", "x"}}, 0)
	assert.Equal(t, "a  b\n-  -\n   x\n\n1 row\n", out)
}

func TestRenderTableCapsColumnWidth(t *testing.T) {
	long := strings.Repeat("x", maxColumnWidth+10)
	out := renderTableString(t, []string{"c"}, [][]string{{long}}, 0)
	assert.Contains(t, out, strings.Repeat("x", maxColumnWidth-len(ellipsis))+ellipsis)
	assert.NotContains(t, out, long)
}

func TestRenderTableCellAtCapNotTruncated(t *testing.T) {
	exact := strings.Repeat("x", maxColumnWidth)
	out := renderTableString(t, []string{"c"}, [][]string{{exact}}, 0)
	assert.Contains(t, out, exact)
	assert.NotContains(t, out, ellipsis)
}

func TestRenderTableCropsColumns(t *testing.T) {
	out := renderTableString(t, []string{"a", "b", "c", "d", "e"}, [][]string{{"1", "2", "3", "4", "5"}}, 10)
	assert.Equal(t, "a  b  ...\n-  -  ---\n1  2  ...\n\n(showing 2 of 5 columns)\n1 row\n", out)
}

func TestRenderTableCropsColumnsMultipleRows(t *testing.T) {
	out := renderTableString(t, []string{"a", "b", "c"}, [][]string{{"1", "2", "3"}, {"4", "5", "6"}}, 4)
	assert.Equal(t, "a  ...\n-  ---\n1  ...\n4  ...\n\n(showing 1 of 3 columns)\n2 rows\n", out)
}

func TestRenderTableSingleColumnNarrowNotCropped(t *testing.T) {
	out := renderTableString(t, []string{"name"}, [][]string{{"alice"}}, 4)
	assert.NotContains(t, out, ellipsis)
	assert.NotContains(t, out, "showing")
}

func TestRenderTableTruncatesWideGlyphCellSafely(t *testing.T) {
	out := renderTableString(t, []string{"c"}, [][]string{{strings.Repeat("世", 30)}}, 0)
	assert.True(t, utf8.ValidString(out))
	assert.Contains(t, out, ellipsis)
}

func TestRowCountLabel(t *testing.T) {
	assert.Equal(t, "0 rows", rowCountLabel(0))
	assert.Equal(t, "1 row", rowCountLabel(1))
	assert.Equal(t, "2 rows", rowCountLabel(2))
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
	assert.Equal(t, 0, detectWidth(&bytes.Buffer{}))
}
