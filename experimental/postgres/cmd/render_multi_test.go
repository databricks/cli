package postgrescmd

import (
	"bytes"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTextMulti_TwoResults(t *testing.T) {
	r1 := &unitResult{
		Source:     "argv[1]",
		SQL:        "INSERT INTO t VALUES (1)",
		CommandTag: "INSERT 0 1",
		Elapsed:    5 * time.Millisecond,
	}
	r2 := &unitResult{
		Source:     "argv[2]",
		SQL:        "SELECT id FROM t",
		Fields:     fieldsWithOIDs([]string{"id"}, []uint32{pgtype.Int8OID}),
		Rows:       [][]any{{int64(1)}},
		CommandTag: "SELECT 1",
		Elapsed:    3 * time.Millisecond,
	}

	var buf bytes.Buffer
	require.NoError(t, renderTextMulti(&buf, []*unitResult{r1, r2}))
	out := buf.String()
	assert.Contains(t, out, "INSERT 0 1\n")
	assert.Contains(t, out, "id")
	assert.Contains(t, out, "(1 row)")
	// Blank-line separator between blocks.
	assert.Contains(t, out, "INSERT 0 1\n\n")
}

func TestRenderJSONMulti_TwoResults(t *testing.T) {
	r1 := &unitResult{
		Source:     "argv[1]",
		SQL:        "INSERT INTO t VALUES (1)",
		CommandTag: "INSERT 0 1",
		Elapsed:    5 * time.Millisecond,
	}
	r2 := &unitResult{
		Source:     "argv[2]",
		SQL:        "SELECT id FROM t",
		Fields:     fieldsWithOIDs([]string{"id"}, []uint32{pgtype.Int8OID}),
		Rows:       [][]any{{int64(1)}, {int64(2)}},
		CommandTag: "SELECT 2",
		Elapsed:    3 * time.Millisecond,
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderJSONMulti(&stdout, &stderr, []*unitResult{r1, r2}, -1, ""))

	out := stdout.String()
	assert.Contains(t, out, `"sql":"INSERT INTO t VALUES (1)"`)
	assert.Contains(t, out, `"kind":"command"`)
	assert.Contains(t, out, `"command":"INSERT"`)
	assert.Contains(t, out, `"rows_affected":1`)
	assert.Contains(t, out, `"sql":"SELECT id FROM t"`)
	assert.Contains(t, out, `"kind":"rows"`)
	assert.Contains(t, out, `"rows":`)
	// Outer array framing.
	assert.Greater(t, len(out), 4)
	assert.Equal(t, byte('['), out[0])
	assert.Equal(t, byte('\n'), out[len(out)-1])
}

func TestRenderJSONMulti_WithErrorAtEnd(t *testing.T) {
	r1 := &unitResult{
		Source:     "argv[1]",
		SQL:        "SELECT 1",
		Fields:     fieldsWithOIDs([]string{"?column?"}, []uint32{pgtype.Int8OID}),
		Rows:       [][]any{{int64(1)}},
		CommandTag: "SELECT 1",
		Elapsed:    1 * time.Millisecond,
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderJSONMulti(&stdout, &stderr, []*unitResult{r1}, 1, "argv[2]: ERROR: syntax error (SQLSTATE 42601)"))

	out := stdout.String()
	assert.Contains(t, out, `"kind":"rows"`)
	assert.Contains(t, out, `"kind":"error"`)
	assert.Contains(t, out, `"message":"argv[2]: ERROR: syntax error (SQLSTATE 42601)"`)
}
