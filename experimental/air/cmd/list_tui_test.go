package aircmd

import (
	"io"
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testListRows is a small fixture covering each status color, a present and an
// absent MLflow link, and a still-running (no end) row.
func testListRows() []listRow {
	return []listRow{
		{RunID: "1", Experiment: "qwen-train", User: "me@example.com", Status: "SUCCESS", StartedAt: new("2026-06-05T17:32:39.000000+00:00"), Duration: "1m 14s", MLflowURL: "https://h/ml/experiments/E/runs/04c41514fbb0/artifacts/logs/node_0", MLflowLabel: "qwen-run-001", Accelerators: "8x H100"},
		{RunID: "2", Experiment: "llama-train", User: "me@example.com", Status: "RUNNING", StartedAt: new("2026-06-05T18:43:24.000000+00:00"), Duration: "3m 32s", Progress: "81% · ~48m", MLflowURL: "-", MLflowLabel: "-", Accelerators: "1x A10"},
		{RunID: "3", Experiment: "mixtral", User: "me@example.com", Status: "FAILED", StartedAt: nil, Duration: "-", MLflowURL: "-", MLflowLabel: "-", Accelerators: "-"},
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
		"Run ID", "Experiment", "Status", "Started", "Duration", "Progress", "MLflow", "User", "Accelerators",
		"qwen-train", "● SUCCESS", "● RUNNING", "● FAILED",
		"81% · ~48m",          // progress on the running row
		"qwen-run-001",        // MLflow run label
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
	for _, want := range []string{"Run ID", "1", "qwen-train", "qwen-run-001", "Accelerators"} {
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

func TestRenderRowHyperlinks(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	styles := newListStyles(r)
	row := listRow{
		RunID: "1", Experiment: "exp", Status: "SUCCESS", Duration: "-", Accelerators: "-",
		RunURL: "https://h/jobs/runs/1?o=2", ExperimentURL: "https://h/ml/experiments/E?o=2",
		MLflowURL: "https://h/ml/experiments/E/runs/rid", MLflowLabel: "my-run",
	}
	cols := computeListCols([]listRow{row})

	linked := styles.renderRow(cols, row, false, true)
	assert.Contains(t, linked, "\x1b]8;;https://h/jobs/runs/1?o=2", "run id links to the dashboard")
	assert.Contains(t, linked, "\x1b]8;;https://h/ml/experiments/E?o=2", "experiment links to the experiment page")

	plain := styles.renderRow(cols, row, false, false)
	assert.NotContains(t, plain, "\x1b]8;;", "no links when links are disabled")
}

func TestListModelInfoKeyOpensDetail(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	f := &runFetcher{ctx: t.Context(), w: newTestWorkspaceClient(t, "https://x.test")}
	m := newListModel(r, f, testListRows(), false)

	// `i` opens the detail pane in a loading state and dispatches a fetch (not run here).
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(listModel)
	assert.Equal(t, modeDetail, m.mode)
	assert.True(t, m.detailLoading)
	assert.NotNil(t, cmd)
}

// detailLoadingModel returns a sized model in the detail-loading state, as if
// the user just pressed `i` and the fetch is still in flight.
func detailLoadingModel(t *testing.T) listModel {
	t.Helper()
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	f := &runFetcher{ctx: t.Context(), w: newTestWorkspaceClient(t, "https://x.test")}
	m := newListModel(r, f, testListRows(), false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = next.(listModel)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")})
	m = next.(listModel)
	require.Equal(t, modeDetail, m.mode)
	require.True(t, m.detailLoading)
	return m
}

func TestListModelDetailPaneAndBack(t *testing.T) {
	m := detailLoadingModel(t)

	// A resolved detailMsg fills the pane.
	next, _ := m.Update(detailMsg{title: "Run Details", body: "hello from the detail pane"})
	m = next.(listModel)
	require.Equal(t, modeDetail, m.mode)
	assert.False(t, m.detailLoading)
	view := m.View()
	assert.Contains(t, view, "Run Details")
	assert.Contains(t, view, "hello from the detail pane")
	assert.Contains(t, view, "esc back")

	// esc returns to the list.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(listModel)
	assert.Equal(t, modeList, m.mode)
	assert.Contains(t, m.View(), "Run ID")
}

func TestListModelDetailLateMsgDropped(t *testing.T) {
	m := detailLoadingModel(t)

	// User escapes back to the list before the fetch resolves.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(listModel)
	require.Equal(t, modeList, m.mode)

	// The late result must be dropped, not snap the user back into the pane.
	next, _ = m.Update(detailMsg{title: "Run Details", body: "late result"})
	m = next.(listModel)
	assert.Equal(t, modeList, m.mode)
	assert.NotContains(t, m.View(), "late result")
}

func TestPadAndTruncate(t *testing.T) {
	assert.Equal(t, "ab   ", pad("ab", 5, false))
	assert.Equal(t, "   ab", pad("ab", 5, true))
	assert.Equal(t, "abcd…", truncate("abcdefgh", 5))
	assert.Equal(t, "abc", truncate("abc", 5))
}
