package tableview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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

func TestSearchSpaceCharacterInput(t *testing.T) {
	m := model{
		searching:   true,
		searchInput: "my",
	}

	result, _ := m.updateSearch(tea.KeyMsg{Type: tea.KeySpace})
	rm := result.(model)

	assert.Equal(t, "my ", rm.searchInput)
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

func TestComputeColumnWidths(t *testing.T) {
	tests := []struct {
		name      string
		headers   []string
		rows      [][]string
		maxWidths []int
		want      []int
	}{
		{
			name:    "header wider than data",
			headers: []string{"username", "id"},
			rows:    [][]string{{"al", "1"}, {"bo", "2"}},
			want:    []int{8, 2},
		},
		{
			name:    "data wider than header",
			headers: []string{"id", "name"},
			rows:    [][]string{{"1", "alexander"}, {"2", "bob"}},
			want:    []int{2, 9},
		},
		{
			name:      "cap limits wide data",
			headers:   []string{"a", "b"},
			rows:      [][]string{{"short", "this-is-a-very-long-value"}},
			maxWidths: []int{10, 10},
			want:      []int{5, 10},
		},
		{
			name:    "nil maxWidths applies no caps",
			headers: []string{"x"},
			rows:    [][]string{{"a-really-long-string-with-no-cap"}},
			want:    []int{32},
		},
		{
			name:    "empty rows returns header widths",
			headers: []string{"name", "age"},
			rows:    nil,
			want:    []int{4, 3},
		},
		{
			name:    "uneven row lengths",
			headers: []string{"a", "b", "c"},
			rows:    [][]string{{"longvalue"}, {"x", "y"}},
			want:    []int{9, 1, 1},
		},
		{
			name:      "zero cap value means no cap for that column",
			headers:   []string{"a", "b"},
			rows:      [][]string{{"longvalue", "longvalue"}},
			maxWidths: []int{0, 5},
			want:      []int{9, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeColumnWidths(tt.headers, tt.rows, tt.maxWidths)
			assert.Equal(t, tt.want, got)
		})
	}
}
