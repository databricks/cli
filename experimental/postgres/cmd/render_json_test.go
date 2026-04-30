package postgrescmd

import (
	"bytes"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsWithOIDs(names []string, oids []uint32) []pgconn.FieldDescription {
	out := make([]pgconn.FieldDescription, len(names))
	for i, n := range names {
		out[i] = pgconn.FieldDescription{Name: n, DataTypeOID: oids[i]}
	}
	return out
}

func TestJSONSink_TwoRows(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)

	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"id", "name"}, []uint32{pgtype.Int8OID, pgtype.TextOID})))
	require.NoError(t, s.Row([]any{int64(1), "alice"}))
	require.NoError(t, s.Row([]any{int64(2), "bob"}))
	require.NoError(t, s.End("SELECT 2"))

	assert.Equal(t,
		"[\n"+
			`{"id":1,"name":"alice"}`+",\n"+
			`{"id":2,"name":"bob"}`+
			"\n]\n",
		stdout.String(),
	)
	assert.Empty(t, stderr.String())
}

func TestJSONSink_EmptyRowsProducing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"id"}, []uint32{pgtype.Int8OID})))
	require.NoError(t, s.End("SELECT 0"))
	assert.Equal(t, "[\n]\n", stdout.String())
}

func TestJSONSink_KeysInColumnOrder(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"b", "a"}, []uint32{pgtype.Int8OID, pgtype.Int8OID})))
	require.NoError(t, s.Row([]any{int64(1), int64(2)}))
	require.NoError(t, s.End("SELECT 1"))
	assert.Equal(t, "[\n"+`{"b":1,"a":2}`+"\n]\n", stdout.String())
}

func TestJSONSink_CommandOnly_WithRowCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(nil))
	require.NoError(t, s.End("INSERT 0 5"))
	// Byte-equal: pins the field order so adding a future field (e.g. last_oid)
	// must update the test rather than silently drift.
	assert.Equal(t, `{"command":"INSERT","rows_affected":5}`+"\n", stdout.String())
}

func TestJSONSink_CommandOnly_NoRowCount(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(nil))
	require.NoError(t, s.End("CREATE DATABASE"))
	assert.Equal(t, `{"command":"CREATE"}`+"\n", stdout.String())
}

func TestJSONSink_DuplicateColumns(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"id", "id", "id"}, []uint32{pgtype.Int8OID, pgtype.Int8OID, pgtype.Int8OID})))
	require.NoError(t, s.Row([]any{int64(1), int64(2), int64(3)}))
	require.NoError(t, s.End("SELECT 1"))

	assert.Contains(t, stdout.String(), `"id":1`)
	assert.Contains(t, stdout.String(), `"id__2":2`)
	assert.Contains(t, stdout.String(), `"id__3":3`)
	assert.Contains(t, stderr.String(), "duplicate column names")
}

func TestJSONSink_OnError_AfterRows(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"id"}, []uint32{pgtype.Int8OID})))
	require.NoError(t, s.Row([]any{int64(1)}))
	s.OnError(assert.AnError)
	assert.Equal(t, "[\n"+`{"id":1}`+"\n]\n", stdout.String())
}

func TestJSONSink_OnError_AfterBeginNoRows(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fieldsWithOIDs([]string{"id"}, []uint32{pgtype.Int8OID})))
	s.OnError(assert.AnError)
	assert.Equal(t, "[\n]\n", stdout.String())
}

func TestJSONSink_OnError_BeforeBegin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newJSONSink(&stdout, &stderr)
	s.OnError(assert.AnError)
	assert.Empty(t, stdout.String())
}

func TestCommandTagParse(t *testing.T) {
	tests := []struct {
		tag      string
		verb     string
		rows     int64
		hasCount bool
	}{
		{"INSERT 0 5", "INSERT", 5, true},
		{"UPDATE 3", "UPDATE", 3, true},
		{"DELETE 0", "DELETE", 0, true},
		{"SELECT 100", "SELECT", 100, true},
		{"MERGE 5", "MERGE", 5, true},
		{"COPY 1000", "COPY", 1000, true},
		{"FETCH 7", "FETCH", 7, true},
		{"MOVE 3", "MOVE", 3, true},
		{"CREATE DATABASE", "CREATE", 0, false},
		{"SET", "SET", 0, false},
	}
	for _, tc := range tests {
		assert.Equal(t, tc.verb, commandTagVerb(tc.tag), "verb for %q", tc.tag)
		count, ok := commandTagRowCount(tc.tag)
		assert.Equal(t, tc.hasCount, ok, "hasCount for %q", tc.tag)
		if tc.hasCount {
			assert.Equal(t, tc.rows, count, "rows for %q", tc.tag)
		}
	}
}
