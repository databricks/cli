package postgrescmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderText_RowsProducing(t *testing.T) {
	r := &queryResult{
		Columns: []string{"id", "name"},
		Rows: [][]string{
			{"1", "alice"},
			{"2", "bob"},
		},
		CommandTag: "SELECT 2",
	}
	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, r))

	assert.Equal(t,
		"id   name\n"+
			"---  ----\n"+
			"1    alice\n"+
			"2    bob\n"+
			"(2 rows)\n",
		buf.String(),
	)
}

func TestRenderText_SingleRow(t *testing.T) {
	r := &queryResult{
		Columns:    []string{"id"},
		Rows:       [][]string{{"42"}},
		CommandTag: "SELECT 1",
	}
	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, r))
	assert.Contains(t, buf.String(), "(1 row)\n")
}

func TestRenderText_Empty(t *testing.T) {
	r := &queryResult{
		Columns:    []string{"id", "name"},
		CommandTag: "SELECT 0",
	}
	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, r))
	assert.Contains(t, buf.String(), "(0 rows)\n")
}

func TestRenderText_CommandOnly(t *testing.T) {
	r := &queryResult{
		CommandTag: "INSERT 0 5",
	}
	var buf bytes.Buffer
	require.NoError(t, renderText(&buf, r))
	assert.Equal(t, "INSERT 0 5\n", buf.String())
}

func TestQueryResultIsRowsProducing(t *testing.T) {
	assert.False(t, (&queryResult{}).IsRowsProducing())
	assert.False(t, (&queryResult{CommandTag: "INSERT 0 1"}).IsRowsProducing())
	assert.True(t, (&queryResult{Columns: []string{"a"}}).IsRowsProducing())
}
