package aircmd

import (
	"io"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testListRows is a small fixture covering each status color, a present and an
// absent MLflow link, and a still-running (no end) row.
func testListRows() []listRow {
	return []listRow{
		{RunID: "1", Experiment: "qwen-train", User: "me@example.com", Status: "SUCCESS", StartedAt: new("2026-06-05T17:32:39.000000+00:00"), Duration: "1m 14s", MLflowURL: "https://h/ml/experiments/E/runs/04c41514fbb0/artifacts/logs/node_0", Accelerators: "8x H100"},
		{RunID: "2", Experiment: "llama-train", User: "me@example.com", Status: "RUNNING", StartedAt: new("2026-06-05T18:43:24.000000+00:00"), Duration: "3m 32s", ETA: "~48m 20s", MLflowURL: "-", Accelerators: "1x A10"},
		{RunID: "3", Experiment: "mixtral", User: "me@example.com", Status: "FAILED", StartedAt: nil, Duration: "-", MLflowURL: "-", Accelerators: "-"},
	}
}

func testListModel(t *testing.T) listModel {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	return newListModel(r, nil, testListRows(), false)
}

func key(t *testing.T, m listModel, s string) listModel {
	t.Helper()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
	return next.(listModel)
}

func TestListModelNavigation(t *testing.T) {
	m := testListModel(t)
	require.Equal(t, 0, m.cursor)

	m = key(t, m, "j")
	assert.Equal(t, 1, m.cursor)
	m = key(t, key(t, m, "k"), "k") // clamp at top
	assert.Equal(t, 0, m.cursor)

	for range len(m.rows) + 2 { // clamp at bottom
		m = key(t, m, "j")
	}
	assert.Equal(t, len(m.rows)-1, m.cursor)
}

func TestListModelWindowScrolls(t *testing.T) {
	m := testListModel(t)
	// Height 5 leaves room for ~2 rows (header + hint reserved).
	next, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 5})
	m = next.(listModel)
	require.Equal(t, 2, m.visibleCount())
	require.Equal(t, 0, m.offset)

	m = key(t, key(t, m, "j"), "j") // move to row index 2, past the window
	assert.Equal(t, 2, m.cursor)
	assert.Equal(t, 1, m.offset, "window scrolled to keep the cursor visible")
}

func TestListModelPageCap(t *testing.T) {
	rows := make([]listRow, 50)
	for i := range rows {
		rows[i] = listRow{RunID: strconv.Itoa(i)}
	}
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	m := newListModel(r, nil, rows, false)

	// A tall terminal still shows at most listPageRows per page.
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 100})
	assert.Equal(t, listPageRows, next.(listModel).visibleCount())
}

func TestListModelPaging(t *testing.T) {
	rows := make([]listRow, 10)
	for i := range rows {
		rows[i] = listRow{RunID: strconv.Itoa(i)}
	}
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	m := newListModel(r, nil, rows, false)

	// Height 7 leaves a 4-row window (header + hint reserved).
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 7})
	m = next.(listModel)
	require.Equal(t, 4, m.visibleCount())

	page := func(k tea.KeyType) {
		n, _ := m.Update(tea.KeyMsg{Type: k})
		m = n.(listModel)
	}
	page(tea.KeyRight)
	assert.Equal(t, 4, m.cursor)
	page(tea.KeyEnd)
	assert.Equal(t, 9, m.cursor)
	page(tea.KeyLeft)
	assert.Equal(t, 5, m.cursor)
	page(tea.KeyHome)
	assert.Equal(t, 0, m.cursor)
}

func TestListModelMoreRows(t *testing.T) {
	m := testListModel(t)
	m.loading = true
	before := len(m.rows)

	next, cmd := m.Update(moreRowsMsg{rows: []listRow{{RunID: "4"}, {RunID: "5"}}})
	m = next.(listModel)

	assert.False(t, m.loading, "loading cleared after a batch arrives")
	assert.NoError(t, m.loadErr)
	require.Len(t, m.rows, before+2)
	assert.Equal(t, "5", m.rows[len(m.rows)-1].RunID, "new rows appended")
	assert.Nil(t, cmd)
}

func TestListModelMoreRowsError(t *testing.T) {
	m := testListModel(t)
	m.loading = true
	before := len(m.rows)

	next, cmd := m.Update(moreRowsMsg{err: io.ErrUnexpectedEOF})
	m = next.(listModel)

	assert.False(t, m.loading)
	assert.ErrorIs(t, m.loadErr, io.ErrUnexpectedEOF)
	assert.Len(t, m.rows, before, "rows unchanged on error")
	assert.Nil(t, cmd)
}

func TestListModelMoreRowsEmptyKeepsPaging(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)

	// An empty page while more runs remain re-fetches; once exhausted it stops.
	m := newListModel(r, &runFetcher{}, testListRows(), false)
	m.loading = true
	next, cmd := m.Update(moreRowsMsg{})
	m = next.(listModel)
	assert.NotNil(t, cmd, "empty page with more to scan keeps paging")

	m.fetcher.exhausted = true
	m.loading = true
	_, cmd = m.Update(moreRowsMsg{})
	assert.Nil(t, cmd, "empty page stops once the fetcher is exhausted")
}

func TestListModelQuit(t *testing.T) {
	m := testListModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	require.NotNil(t, cmd)
	assert.Equal(t, tea.QuitMsg{}, cmd())
}

func TestListModelView(t *testing.T) {
	next, _ := testListModel(t).Update(tea.WindowSizeMsg{Width: 200, Height: 24})
	out := next.(listModel).View()

	assert.NotContains(t, out, "\x1b", "Ascii profile + no links should produce no escapes")
	for _, want := range []string{
		"Run ID", "Experiment", "Status", "Started", "Duration", "ETA", "MLflow", "User", "Accelerators",
		"qwen-train", "● SUCCESS", "● RUNNING", "● FAILED",
		"~48m 20s",            // ETA on the running row
		"…/runs/04c41514…",    // shortened MLflow link
		"2026-06-05T17:32:39", // started trimmed to seconds
		"▸",                   // selection gutter on the first row
		"↑/↓ navigate",        // hint line
	} {
		assert.Contains(t, out, want)
	}
}

func TestStaticListTable(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	out := staticListTable(r, testListRows(), false)

	assert.NotContains(t, out, "\x1b")
	assert.NotContains(t, out, "▸", "static table has no selection")
	for _, want := range []string{"Run ID", "1", "qwen-train", "…/runs/04c41514…", "Accelerators"} {
		assert.Contains(t, out, want)
	}

	assert.Equal(t, "No runs found.\n", staticListTable(r, nil, false))
}

func TestStatusColor(t *testing.T) {
	assert.Equal(t, colGreen, statusColor("SUCCESS"))
	assert.Equal(t, colAmber, statusColor("RUNNING"))
	assert.Equal(t, colAmber, statusColor("PENDING"))
	assert.Equal(t, colRed, statusColor("FAILED"))
	assert.Equal(t, colN7, statusColor("CANCELED"))
	assert.Equal(t, colN7, statusColor("UNKNOWN"))
}

func TestStartedDisplay(t *testing.T) {
	assert.Equal(t, "-", startedDisplay(listRow{}))
	assert.Equal(t, "2026-06-05T17:32:39", startedDisplay(listRow{StartedAt: new("2026-06-05T17:32:39.000000+00:00")}))
}

func TestMLflowDisplay(t *testing.T) {
	assert.Equal(t, "…/runs/04c41514…", mlflowDisplay("https://h/ml/experiments/E/runs/04c41514fbb0/artifacts/logs/node_0"))
	assert.Equal(t, "…/runs/run1", mlflowDisplay("https://h/ml/experiments/E/runs/run1/artifacts/logs/node_0"))
	assert.LessOrEqual(t, lipgloss.Width(mlflowDisplay("https://h/no-runs/here")), mlflowColWidth)
}

func TestMLflowRunID(t *testing.T) {
	assert.Equal(t, "abc123", mlflowRunID("https://h/ml/experiments/1/runs/abc123/artifacts"))
	assert.Empty(t, mlflowRunID("https://h/no-runs-here"))
}

func TestPadAndTruncate(t *testing.T) {
	assert.Equal(t, "ab   ", pad("ab", 5, false))
	assert.Equal(t, "   ab", pad("ab", 5, true))
	assert.Equal(t, "abcd…", truncate("abcdefgh", 5))
	assert.Equal(t, "abc", truncate("abc", 5))
}
