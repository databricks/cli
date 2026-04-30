package postgrescmd

import (
	"bytes"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fields is a small helper to build []pgconn.FieldDescription with just names
// (no OIDs), so renderer tests don't need to know about Postgres OIDs.
func fields(names ...string) []pgconn.FieldDescription {
	out := make([]pgconn.FieldDescription, len(names))
	for i, n := range names {
		out[i] = pgconn.FieldDescription{Name: n}
	}
	return out
}

func TestTextSink_RowsProducing(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)

	require.NoError(t, s.Begin(fields("id", "name")))
	require.NoError(t, s.Row([]any{int64(1), "alice"}))
	require.NoError(t, s.Row([]any{int64(2), "bob"}))
	require.NoError(t, s.End("SELECT 2"))

	assert.Equal(t,
		"id   name\n"+
			"---  ----\n"+
			"1    alice\n"+
			"2    bob\n"+
			"(2 rows)\n",
		buf.String(),
	)
}

func TestTextSink_SingleRow(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)
	require.NoError(t, s.Begin(fields("id")))
	require.NoError(t, s.Row([]any{int64(42)}))
	require.NoError(t, s.End("SELECT 1"))
	assert.Contains(t, buf.String(), "(1 row)\n")
}

func TestTextSink_Empty(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)
	require.NoError(t, s.Begin(fields("id", "name")))
	require.NoError(t, s.End("SELECT 0"))
	assert.Contains(t, buf.String(), "(0 rows)\n")
}

func TestTextSink_CommandOnly(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)
	require.NoError(t, s.Begin(nil))
	require.NoError(t, s.End("INSERT 0 5"))
	assert.Equal(t, "INSERT 0 5\n", buf.String())
}

func TestTextSink_NULLRendersAsNULL(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)
	require.NoError(t, s.Begin(fields("id")))
	require.NoError(t, s.Row([]any{nil}))
	require.NoError(t, s.End("SELECT 1"))
	assert.Contains(t, buf.String(), "NULL")
}

func TestTextSink_OnError_NoOp(t *testing.T) {
	var buf bytes.Buffer
	s := newTextSink(&buf)
	require.NoError(t, s.Begin(fields("id")))
	require.NoError(t, s.Row([]any{int64(1)}))
	s.OnError(assert.AnError)
	// Text sink has no open structure to close. OnError must not panic and
	// must not emit a partial table; the partial result lives in s.rows but
	// is never flushed.
	assert.Empty(t, buf.String())
}
