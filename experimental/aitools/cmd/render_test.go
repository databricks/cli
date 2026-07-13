package aitools

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	var buf bytes.Buffer
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}}

	err := renderStaticTable(&buf, columns, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "name")
	assert.Contains(t, output, "alice")
	assert.Contains(t, output, "bob")
	assert.Contains(t, output, "---")
	assert.Contains(t, output, "2 rows")
}

func TestRenderStaticTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	columns := []string{"id", "name"}
	var rows [][]string

	err := renderStaticTable(&buf, columns, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "id")
	assert.Contains(t, output, "0 rows")
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
