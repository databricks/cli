package postgrescmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVSink_TwoRows(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newCSVSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fields("id", "name")))
	require.NoError(t, s.Row([]any{int64(1), "alice"}))
	require.NoError(t, s.Row([]any{int64(2), "bob"}))
	require.NoError(t, s.End("SELECT 2"))

	assert.Equal(t, "id,name\n1,alice\n2,bob\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestCSVSink_NULLEmptyField(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newCSVSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fields("id", "note")))
	require.NoError(t, s.Row([]any{int64(1), nil}))
	require.NoError(t, s.End("SELECT 1"))

	assert.Equal(t, "id,note\n1,\n", stdout.String())
}

func TestCSVSink_CommandOnly(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newCSVSink(&stdout, &stderr)
	require.NoError(t, s.Begin(nil))
	require.NoError(t, s.End("CREATE DATABASE"))
	assert.Empty(t, stdout.String())
	assert.Equal(t, "CREATE DATABASE\n", stderr.String())
}

func TestCSVSink_QuotesFieldsWithCommas(t *testing.T) {
	var stdout, stderr bytes.Buffer
	s := newCSVSink(&stdout, &stderr)
	require.NoError(t, s.Begin(fields("note")))
	require.NoError(t, s.Row([]any{"a,b"}))
	require.NoError(t, s.End("SELECT 1"))
	assert.Contains(t, stdout.String(), `"a,b"`)
}
