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
	require.NoError(t, renderJSONMulti(&stdout, &stderr, []*unitResult{r1, r2}, nil, ""))

	out := stdout.String()
	// Canonical key order: source, sql, kind, elapsed_ms, payload.
	assert.Contains(t, out, `"source":"argv[1]","sql":"INSERT INTO t VALUES (1)","kind":"command","elapsed_ms":5,"command":"INSERT","rows_affected":1`)
	assert.Contains(t, out, `"source":"argv[2]","sql":"SELECT id FROM t","kind":"rows","elapsed_ms":3,"rows":`)
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
	errored := &unitResult{
		Source:  "argv[2]",
		SQL:     "BROKEN SQL",
		Elapsed: 2 * time.Millisecond,
	}

	var stdout, stderr bytes.Buffer
	require.NoError(t, renderJSONMulti(&stdout, &stderr, []*unitResult{r1}, errored, "ERROR: syntax error (SQLSTATE 42601)"))

	out := stdout.String()
	assert.Contains(t, out, `"kind":"rows"`)
	// Error envelope: same key order, includes elapsed_ms + source + sql.
	assert.Contains(t, out, `"source":"argv[2]","sql":"BROKEN SQL","kind":"error","elapsed_ms":2,"error":{"message":"ERROR: syntax error (SQLSTATE 42601)"}`)
}

func TestRenderJSONMulti_FirstUnitFails(t *testing.T) {
	errored := &unitResult{
		Source:  "argv[1]",
		SQL:     "BROKEN",
		Elapsed: 7 * time.Millisecond,
	}
	var stdout, stderr bytes.Buffer
	require.NoError(t, renderJSONMulti(&stdout, &stderr, nil, errored, "ERROR: bad"))

	out := stdout.String()
	// No leading separator before the single error envelope.
	assert.Contains(t, out, "[\n"+`{"source":"argv[1]","sql":"BROKEN","kind":"error","elapsed_ms":7,"error":{"message":"ERROR: bad"}}`)
	assert.Contains(t, out, "\n]\n")
}
