package aitools

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func renderStaticTable(t *testing.T, columns []string, rows [][]string) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, tableview.RenderStatic(cmdio.MockDiscard(t.Context()), &buf, columns, rows, tableview.StaticOptions{Width: 0, TruncateCells: false}))
	return buf.String()
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}}

	err := renderJSON(&buf, columns, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"alice"`)
	assert.Contains(t, output, `"bob"`)
	assert.NotContains(t, output, "Row count")
}

func TestRenderJSONNoRows(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id"}
	var rows [][]string

	err := renderJSON(&buf, columns, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "Row count")
}

func TestRenderStaticTable(t *testing.T) {
	output := renderStaticTable(t, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}})
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "bob")
	assert.Contains(t, output, "---")
	assert.Contains(t, output, "2 rows")
}

func TestRenderStaticTableEmpty(t *testing.T) {
	output := renderStaticTable(t, []string{"id", "name"}, nil)
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "0 rows")
}

func TestRenderStaticTableByteParity(t *testing.T) {
	tests := []struct {
		name    string
		columns []string
		rows    [][]string
		want    string
	}{
		{"basic", []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}}, "id  name\n--  -----\n1   alice\n2   bob\n\n2 rows\n"},
		{"single", []string{"c"}, [][]string{{"a"}, {"bbbbbb"}}, "c\n------\na\nbbbbbb\n\n2 rows\n"},
		{"ragged", []string{"a", "b"}, [][]string{{"1"}}, "a  b\n-  -\n1  \n\n1 rows\n"},
		{"empty-rows", []string{"id", "name"}, nil, "id  name\n--  ----\n\n0 rows\n"},
		{"exactly40", []string{"c"}, [][]string{{strings.Repeat("z", 40)}}, "c\n" + strings.Repeat("-", 40) + "\n" + strings.Repeat("z", 40) + "\n\n1 rows\n"},
		{"over40-mid", []string{"a", "b"}, [][]string{{strings.Repeat("x", 45), "B"}}, "a" + strings.Repeat(" ", 44) + "  b\n" + strings.Repeat("-", 40) + strings.Repeat(" ", 5) + "  -\n" + strings.Repeat("x", 45) + "  B\n\n1 rows\n"},
		{"empty-cells", []string{"a", "b"}, [][]string{{"", ""}}, "a  b\n-  -\n   \n\n1 rows\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, renderStaticTable(t, tc.columns, tc.rows))
		})
	}
}

func TestRenderCSVBasic(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id", "name", "city"}
	rows := [][]string{
		{"1", "Alice", "New York"},
		{"2", "Bob", "London"},
	}

	err := renderCSV(&buf, columns, rows)
	require.NoError(t, err)
	assert.Equal(t, "id,name,city\r\n1,Alice,New York\r\n2,Bob,London\r\n", buf.String())
}

func TestRenderCSVSpecialCharacters(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"name", "description"}
	rows := [][]string{
		{"Alice", "has a comma, here"},
		{"Bob", `has "quotes" here`},
		{"Carol", "has a\nnewline"},
	}

	err := renderCSV(&buf, columns, rows)
	require.NoError(t, err)
	assert.Equal(t, "name,description\r\nAlice,\"has a comma, here\"\r\nBob,\"has \"\"quotes\"\" here\"\r\nCarol,\"has a\r\nnewline\"\r\n", buf.String())
}

func TestRenderCSVEmptyResultSet(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id", "name"}
	var rows [][]string

	err := renderCSV(&buf, columns, rows)
	require.NoError(t, err)
	assert.Equal(t, "id,name\r\n", buf.String())
}

func TestRenderCSVShortRows(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"a", "b", "c"}
	rows := [][]string{
		{"1"},
	}

	err := renderCSV(&buf, columns, rows)
	require.NoError(t, err)
	assert.Equal(t, "a,b,c\r\n1,,\r\n", buf.String())
}
