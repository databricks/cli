package tableview

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderTableLines(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}}

	lines := renderTableLines(columns, rows)
	require.GreaterOrEqual(t, len(lines), 4)

	assert.Contains(t, lines[0], "id")
	assert.Contains(t, lines[0], "name")
	assert.Contains(t, lines[1], "──")
	assert.Contains(t, lines[2], "alice")
	assert.Contains(t, lines[3], "bob")
}

func TestRenderTableLinesEmpty(t *testing.T) {
	columns := []string{"id", "name"}
	var rows [][]string

	lines := renderTableLines(columns, rows)
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Contains(t, lines[0], "id")
	assert.Contains(t, lines[1], "──")
}

func TestFindMatches(t *testing.T) {
	lines := []string{"header", "---", "alice", "bob", "alice again"}
	matches := findMatches(lines, "alice")
	assert.Equal(t, []int{2, 4}, matches)
}

func TestFindMatchesCaseInsensitive(t *testing.T) {
	lines := []string{"Alice", "BOB", "alice"}
	matches := findMatches(lines, "ALICE")
	assert.Equal(t, []int{0, 2}, matches)
}

func TestFindMatchesNoResults(t *testing.T) {
	lines := []string{"alice", "bob"}
	matches := findMatches(lines, "charlie")
	assert.Nil(t, matches)
}

func TestFindMatchesEmptyQuery(t *testing.T) {
	lines := []string{"alice", "bob"}
	matches := findMatches(lines, "")
	assert.Nil(t, matches)
}

func TestHighlightSearchEmptyQuery(t *testing.T) {
	result := highlightSearch("hello alice", "")
	assert.Equal(t, "hello alice", result)
}

func TestHighlightSearchWithMatch(t *testing.T) {
	result := highlightSearch("hello alice", "alice")
	assert.Contains(t, result, "alice")
	assert.Contains(t, result, "hello")
}

func TestHighlightSearchNoMatch(t *testing.T) {
	result := highlightSearch("hello bob", "alice")
	assert.Equal(t, "hello bob", result)
}
