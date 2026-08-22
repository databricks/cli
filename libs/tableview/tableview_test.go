package tableview

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
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

func TestTruncateCell(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"short", "abc", "abc"},
		{"exactly max", strings.Repeat("x", maxCellWidth), strings.Repeat("x", maxCellWidth)},
		{"over max", strings.Repeat("x", maxCellWidth+10), strings.Repeat("x", maxCellWidth-3) + "..."},
		{"many bytes few runes", strings.Repeat("é", 200), strings.Repeat("é", 200)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, truncateCell(tc.in))
		})
	}
}

func TestTruncateCellMultibyteBoundary(t *testing.T) {
	got := truncateCell(strings.Repeat("é", maxCellWidth+10))
	assert.True(t, utf8.ValidString(got))
	assert.True(t, strings.HasSuffix(got, "..."))
}

func TestRenderTableLinesTruncatesLongCell(t *testing.T) {
	long := strings.Repeat("x", 10_000)
	lines := renderTableLines([]string{"blob"}, [][]string{{long}})
	require.GreaterOrEqual(t, len(lines), 3)
	dataRow := lines[2]
	assert.LessOrEqual(t, len(dataRow), maxCellWidth)
	assert.Contains(t, dataRow, "...")
	assert.NotContains(t, dataRow, long)
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
	result := model{}.highlightSearch("hello alice", "")
	assert.Equal(t, "hello alice", result)
}

func TestHighlightSearchWithMatch(t *testing.T) {
	result := model{}.highlightSearch("hello alice", "alice")
	assert.Contains(t, result, "alice")
	assert.Contains(t, result, "hello")
}

func TestHighlightSearchNoMatch(t *testing.T) {
	result := model{}.highlightSearch("hello bob", "alice")
	assert.Equal(t, "hello bob", result)
}

// readyModel constructs a model in the same shape Run produces, plus a viewport
// large enough that the cursor visibility logic does not need to scroll.
func readyModel(columns []string, rows [][]string, viewportHeight int) model {
	all := renderTableLines(columns, rows)
	m := model{
		header: all[:headerLines],
		lines:  all[headerLines:],
	}
	m.viewport = viewport.New(80, viewportHeight)
	m.viewport.SetContent(m.renderContent())
	m.ready = true
	return m
}

func TestViewKeepsHeaderAboveScrollableContent(t *testing.T) {
	columns := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}, {"2", "bob"}, {"3", "carol"}}
	m := readyModel(columns, rows, 2)

	// Scroll the viewport down so the first data row falls below the top
	// of the viewport. Before the sticky-header change this would also push
	// the column header off-screen and never bring it back.
	m.viewport.SetYOffset(1)

	out := m.View()
	headerIdx := strings.Index(out, "id")
	carolIdx := strings.Index(out, "carol")
	require.NotEqual(t, -1, headerIdx, "View output must contain the column header")
	require.NotEqual(t, -1, carolIdx, "View output must contain the visible row after scrolling")
	assert.Less(t, headerIdx, carolIdx, "column header must render above the scrolled rows")
}

func TestModelDataRowCountExcludesHeader(t *testing.T) {
	m := readyModel([]string{"id"}, [][]string{{"1"}, {"2"}, {"3"}}, 5)
	assert.Equal(t, 3, m.dataRowCount())
}

func TestMoveCursorClampsAtZeroAndLast(t *testing.T) {
	m := readyModel([]string{"id"}, [][]string{{"1"}, {"2"}, {"3"}}, 5)
	m.moveCursor(-100)
	assert.Equal(t, 0, m.cursor, "cursor should clamp to first data row, not below")
	m.moveCursor(100)
	assert.Equal(t, 2, m.cursor, "cursor should clamp to last data row")
}
