package agentstream

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSQLArgs(t *testing.T) {
	tests := []struct {
		name      string
		tool      string
		arguments string
		wantSQL   string
		wantTitle string
	}{
		{"execute_sql", toolExecuteSQL, `{"sql":"SELECT 1","title":"One"}`, "SELECT 1", "One"},
		{"execute_sql_query", toolExecuteSQLQuery, `{"query":"SELECT 2","thought":"count"}`, "SELECT 2", ""},
		{"malformed arguments", toolExecuteSQL, `not json`, "", ""},
		{"unknown tool", "other_tool", `{"sql":"SELECT 3"}`, "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sql, title := parseSQLArgs(tc.tool, tc.arguments)
			assert.Equal(t, tc.wantSQL, sql)
			assert.Equal(t, tc.wantTitle, title)
		})
	}
}

func TestRenderSQL(t *testing.T) {
	var buf bytes.Buffer
	renderSQL(&buf, toolExecuteSQL, `{"sql":"SELECT 1\nFROM t","title":"One"}`)
	out := buf.String()
	assert.Contains(t, out, "SQL executed (One):")
	assert.Contains(t, out, "  SELECT 1\n  FROM t\n")
}

func TestRenderSQL_MalformedArgumentsPrintsNothing(t *testing.T) {
	var buf bytes.Buffer
	renderSQL(&buf, toolExecuteSQL, `not json`)
	assert.Empty(t, buf.String())
}

func TestIsSQLTool(t *testing.T) {
	assert.True(t, isSQLTool(toolExecuteSQL))
	assert.True(t, isSQLTool(toolExecuteSQLQuery))
	assert.False(t, isSQLTool("output_final_response"))
}
