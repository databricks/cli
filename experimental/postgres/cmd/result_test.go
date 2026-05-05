package postgrescmd

import (
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBufferSink_BeginClonesFields(t *testing.T) {
	r := &unitResult{}
	s := &bufferSink{result: r}

	// pgx hands Begin a slice whose backing array gets reused for the next
	// query on the same connection. Mutating the caller's slice after Begin
	// must not change what the buffered result holds.
	fields := []pgconn.FieldDescription{
		{Name: "first_col", DataTypeOID: 23},
	}
	require.NoError(t, s.Begin(fields))

	fields[0] = pgconn.FieldDescription{Name: "second_col", DataTypeOID: 25}

	require.Len(t, r.Fields, 1)
	assert.Equal(t, "first_col", r.Fields[0].Name)
	assert.Equal(t, uint32(23), r.Fields[0].DataTypeOID)
}

func TestBufferSink_RowAndEnd(t *testing.T) {
	r := &unitResult{}
	s := &bufferSink{result: r}

	require.NoError(t, s.Begin([]pgconn.FieldDescription{{Name: "a"}}))
	require.NoError(t, s.Row([]any{int64(1)}))
	require.NoError(t, s.Row([]any{int64(2)}))
	require.NoError(t, s.End("SELECT 2"))

	assert.Equal(t, [][]any{{int64(1)}, {int64(2)}}, r.Rows)
	assert.Equal(t, "SELECT 2", r.CommandTag)
}
