package mcp

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractColumns(t *testing.T) {
	tests := []struct {
		name     string
		manifest *sql.ResultManifest
		want     []string
	}{
		{
			"with columns",
			&sql.ResultManifest{Schema: &sql.ResultSchema{
				Columns: []sql.ColumnInfo{{Name: "id"}, {Name: "name"}},
			}},
			[]string{"id", "name"},
		},
		{"nil manifest", nil, nil},
		{"nil schema", &sql.ResultManifest{}, nil},
		{
			"empty columns",
			&sql.ResultManifest{Schema: &sql.ResultSchema{}},
			[]string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractColumns(tc.manifest)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRenderJSON(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	var buf bytes.Buffer
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}}

	err := renderJSON(ctx, &buf, columns, rows)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"alice"`)
	assert.Contains(t, output, `"bob"`)
	// Row count goes to stderr, not stdout. Stdout should be valid JSON.
	assert.NotContains(t, output, "Row count")
}

func TestRenderJSONNoRows(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	var buf bytes.Buffer
	columns := []string{"id"}
	var rows [][]string

	err := renderJSON(ctx, &buf, columns, rows)
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
