package aircmd

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testListRows is a small fixture covering each status color, a present and an
// absent MLflow link, and a still-running (no end) row.
func testListRows() []listRow {
	return []listRow{
		{RunID: "1", Experiment: "qwen-train", User: "me@example.com", Status: "SUCCESS", StartedAt: new("2026-06-05T17:32:39.000000+00:00"), Duration: "1m 14s", MLflowURL: "https://h/ml/experiments/E/runs/04c41514fbb0/logs", MLflowLabel: "qwen-run-001", Accelerators: "8x H100"},
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

func TestListModelEnterOpensMLflowLogs(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, "BROWSER", "none")
	r, _ := cmdio.NewRenderer(ctx, io.Discard)
	f := &runFetcher{ctx: ctx, w: newTestWorkspaceClient(t, "https://x.test")}
	m := newListModel(r, f, testListRows(), false)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	cmd()

	assert.Contains(t, stderr.String(), "https://h/ml/experiments/E/runs/04c41514fbb0/logs")
	assert.NotContains(t, stderr.String(), "/artifacts/logs/node_0")
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
	for _, want := range []string{"Run ID", "1", "qwen-train", "1m 14s", "qwen-run-001", "8x H100", "Accelerators"} {
		assert.Contains(t, out, want)
	}

	assert.Equal(t, "No runs found.\n", staticListTable(r, nil, false))
}

func TestEmptyListMessage(t *testing.T) {
	assert.Equal(t, "No active runs found. Use --all-status to include runs that are no longer active.\n", emptyListMessage(true))
	assert.Equal(t, "No runs found.\n", emptyListMessage(false))
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

func TestRenderRowStripsControlCharacters(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	styles := newListStyles(r)
	row := listRow{
		RunID: "1", Status: "SUCCESS", Duration: "-", Accelerators: "-",
		Experiment:  "\x1b]0;pwned\x07exp\x1b[2Jeriment",
		User:        "user\r@example.com",
		MLflowURL:   "https://h/ml/experiments/E/runs/rid",
		MLflowLabel: "my\x1b[31m-run",
	}
	cols := computeListCols([]listRow{row})

	out := styles.renderRow(cols, row, false, false)
	assert.Contains(t, out, "experiment")
	assert.Contains(t, out, "user@example.com")
	assert.Contains(t, out, "my-run")
	assert.NotContains(t, out, "\x1b")
	assert.NotContains(t, out, "\r")
	// Stripping happens before width is measured, so the columns stay aligned.
	assert.Equal(t, cols.experiment, lipgloss.Width(stripControl(row.Experiment)))
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

func TestListModelDetailPaneWrapsLongLines(t *testing.T) {
	m := detailLoadingModel(t)
	reason := "Failure reason: " + strings.Repeat("the task exited with a long error ", 4) + "ENDOFREASON"
	require.Greater(t, len(reason), 80)

	next, _ := m.Update(detailMsg{title: "Run Details", body: reason})
	m = next.(listModel)
	assert.Contains(t, m.View(), "ENDOFREASON", "the end of a long line must be visible, not clipped at the pane edge")

	// A narrower terminal re-wraps the same text.
	next, _ = m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})
	m = next.(listModel)
	pane := m.viewport.View()
	assert.Contains(t, pane, "ENDOFREASON")
	for line := range strings.SplitSeq(pane, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 30, line)
	}
}

func TestListModelDetailPaneKeepsBoxRowsAndURLsWhole(t *testing.T) {
	m := detailLoadingModel(t)
	boxRow := "│  command: python train.py " + strings.Repeat("--flag ", 12) + "│"
	url := "MLflow URL: https://example.test/ml/experiments/1/runs/" + strings.Repeat("a", 64)
	require.Greater(t, len(boxRow), 80)
	require.Greater(t, len(url), 80)

	next, _ := m.Update(detailMsg{title: "Run Details", body: boxRow + "\n" + url})
	m = next.(listModel)
	assert.Equal(t, 2, m.viewport.TotalLineCount(), "box rows and URLs must not wrap")

	// ←/→ pans to the clipped end instead.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m = next.(listModel)
	assert.Positive(t, m.viewport.HorizontalScrollPercent())
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

func TestTruncateFitsDisplayWidth(t *testing.T) {
	// A CJK character takes two cells and a combining mark none, so the cut is
	// by display width and never splits a character.
	const wide = "/Users/张伟/大模型微调实验第一百二十三次评估"
	const accented = "xxxxxxxxxxxxxxxxxxxx\u0301yz"
	tests := []struct {
		name      string
		s         string
		n         int
		wantEnd   string // truncate
		wantStart string // truncateStart
	}{
		{name: "fits", s: "abc", n: 5, wantEnd: "abc", wantStart: "abc"},
		{name: "exactly fits", s: "abcde", n: 5, wantEnd: "abcde", wantStart: "abcde"},
		{name: "one cell over", s: "abcdef", n: 5, wantEnd: "abcd…", wantStart: "…cdef"},
		{name: "wide characters", s: wide, n: 11, wantEnd: "/Users/张…", wantStart: "…十三次评估"},
		{name: "cut inside a wide character", s: wide, n: 12, wantEnd: "/Users/张伟…", wantStart: "…十三次评估"},
		{name: "wide character at width two", s: "评估评估", n: 2, wantEnd: "…", wantStart: "…"},
		{name: "combining mark stays with its letter", s: accented, n: 4, wantEnd: "xxx…", wantStart: "…x\u0301yz"},
		{name: "combining mark is dropped with its letter", s: accented, n: 3, wantEnd: "xx…", wantStart: "…yz"},
		{name: "width of one", s: "abcdef", n: 1, wantEnd: "…", wantStart: "…"},
		{name: "width of zero", s: "abcdef", n: 0, wantEnd: "", wantStart: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantEnd, truncate(tt.s, tt.n))
			assert.Equal(t, tt.wantStart, truncateStart(tt.s, tt.n))
		})
	}
}

func TestPadAndTruncate(t *testing.T) {
	assert.Equal(t, "ab   ", pad("ab", 5, false))
	assert.Equal(t, "   ab", pad("ab", 5, true))
	assert.Equal(t, "abcd…", truncate("abcdefgh", 5))
	assert.Equal(t, "abc", truncate("abc", 5))
}

// wideListRows is a fixture whose Experiment and User values both exceed
// columnCap, with realistic widths in the fixed columns.
func wideListRows() []listRow {
	return []listRow{
		{RunID: "684127395104822", Experiment: "/Users/someone.longname@example.com/quickstart/qwen-finetune-lora", User: "someone.with.a.long.name.for.testing@example.com", Status: "SUCCESS", StartedAt: new("2026-10-23T17:32:39.000000+00:00"), Duration: "1h 14m", MLflowURL: "https://h.test/ml/experiments/E/runs/04c41514fbb0", MLflowLabel: "qwen-run-001", Accelerators: "8x H100"},
		{RunID: "684127395104823", Experiment: "mixtral", User: "me@example.com", Status: "RUNNING", StartedAt: new("2026-10-23T18:43:24.000000+00:00"), Duration: "3m 32s", Progress: "81% · ~48m", MLflowURL: "-", MLflowLabel: "-", Accelerators: "1x A10"},
	}
}

func TestListColsFitWidth(t *testing.T) {
	natural := computeListCols(wideListRows())
	require.Equal(t, columnCap, natural.experiment)
	require.Equal(t, columnCap, natural.user)
	require.Equal(t, 186, natural.tableWidth())

	tests := []struct {
		name           string
		width          int
		wantExperiment int
		wantUser       int
		wantFits       bool
	}{
		{name: "unknown width leaves columns alone", width: 0, wantExperiment: columnCap, wantUser: columnCap},
		// The fixed columns alone need 114 cells, so both flexible columns sit at
		// their floors and the table still overflows.
		{name: "80 columns", width: 80, wantExperiment: experimentMinWidth, wantUser: userMinWidth},
		{name: "120 columns", width: 120, wantExperiment: experimentMinWidth, wantUser: userMinWidth},
		{name: "narrowest width that fits", width: 134, wantExperiment: experimentMinWidth, wantUser: userMinWidth, wantFits: true},
		{name: "160 columns splits the shortfall", width: 160, wantExperiment: 23, wantUser: 23, wantFits: true},
		{name: "200 columns keeps natural widths", width: 200, wantExperiment: columnCap, wantUser: columnCap, wantFits: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := natural.fitWidth(tt.width)
			assert.Equal(t, tt.wantExperiment, got.experiment)
			assert.Equal(t, tt.wantUser, got.user)
			if tt.wantFits {
				assert.LessOrEqual(t, got.tableWidth(), tt.width)
			}

			// Only the flexible columns change.
			got.experiment, got.user = natural.experiment, natural.user
			assert.Equal(t, natural, got)
		})
	}
}

func TestListColsFitWidthFloorsNeverWiden(t *testing.T) {
	// Columns already narrower than the floors stay as they are.
	natural := computeListCols(testListRows())
	got := natural.fitWidth(40)
	assert.Equal(t, natural.experiment, got.experiment)
	assert.Equal(t, userMinWidth, got.user)
}

func TestListModelFitsTerminalWidth(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	rows := wideListRows()
	f := &runFetcher{ctx: t.Context(), exhausted: true}
	m := newListModel(r, f, rows[:1], false)

	// Before the terminal reports its size the columns keep their natural widths.
	assert.Equal(t, columnCap, m.cols.experiment)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 30})
	m = next.(listModel)
	lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	require.Len(t, lines, 3) // header, one row, hint
	for _, line := range lines[:2] {
		assert.LessOrEqual(t, ansi.StringWidth(line), 160)
	}
	assert.Contains(t, lines[0], "Accelerators")
	assert.Contains(t, lines[1], "8x H100")
	// Experiment keeps its end, User its start.
	assert.Contains(t, lines[1], "  …tart/qwen-finetune-lora  ")
	assert.Contains(t, lines[1], "  someone.with.a.long.nam…  ")

	// A lazily loaded page is laid out against the same width.
	next, _ = m.Update(moreRowsMsg{rows: rows[1:]})
	m = next.(listModel)
	assert.LessOrEqual(t, m.cols.tableWidth(), 160)

	// Widening the terminal restores the natural widths.
	next, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 30})
	m = next.(listModel)
	assert.Equal(t, columnCap, m.cols.experiment)
	assert.Equal(t, columnCap, m.cols.user)
}

func TestStaticListTableIgnoresTerminalWidth(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	lines := strings.Split(staticListTable(r, wideListRows(), false), "\n")
	assert.Equal(t, computeListCols(wideListRows()).tableWidth(), ansi.StringWidth(lines[0]))
	// The one-shot table still cuts a long Experiment at its end.
	assert.Contains(t, lines[1], " /Users/someone.longname@example.com… ")
}

func TestLayoutListCols(t *testing.T) {
	rows := wideListRows()
	natural := computeListCols(rows)
	require.Equal(t, 134, natural.fitWidth(134).tableWidth())

	tests := []struct {
		name           string
		width          int
		wantExperiment int
		wantUser       int
		wantCompact    bool
		wantHidden     []string
		wantOverflow   bool
	}{
		{name: "unknown width keeps natural columns", width: 0, wantExperiment: columnCap, wantUser: columnCap},
		// Even at the floors the full table is 134 cells, so the compact tier
		// applies and its 17 spare cells go back to Experiment and User.
		{name: "133 compacts Started and MLflow", width: 133, wantExperiment: 18, wantUser: 19, wantCompact: true},
		{name: "120 compacts and shows every column", width: 120, wantExperiment: 12, wantUser: 12, wantCompact: true},
		// Hiding Accelerators alone leaves 102 cells, so User goes too and
		// Experiment widens into the freed space.
		{name: "100 hides Accelerators and User", width: 100, wantExperiment: 20, wantUser: 0, wantCompact: true, wantHidden: []string{"User", "Accelerators"}},
		{name: "80 hides MLflow as well", width: 80, wantExperiment: 14, wantUser: 0, wantCompact: true, wantHidden: []string{"MLflow", "User", "Accelerators"}},
		// With every hideable column gone the table is 66 cells, so it still
		// overflows and the floors hold.
		{name: "60 hides Progress and still overflows", width: 60, wantExperiment: experimentMinWidth, wantUser: 0, wantCompact: true, wantHidden: []string{"Progress", "MLflow", "User", "Accelerators"}, wantOverflow: true},
		{name: "160 is unchanged from today", width: 160, wantExperiment: 23, wantUser: 23},
		{name: "200 is unchanged from today", width: 200, wantExperiment: columnCap, wantUser: columnCap},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutListCols(rows, tt.width)
			assert.Equal(t, tt.wantExperiment, got.experiment)
			if got.shown(columnUser) {
				assert.Equal(t, tt.wantUser, got.user)
			} else {
				assert.Equal(t, 0, tt.wantUser, "User hidden but a width was expected")
			}
			assert.Equal(t, tt.wantCompact, got.compactStarted)
			assert.Equal(t, tt.wantHidden, got.hiddenNames())
			switch {
			case tt.wantOverflow:
				assert.Greater(t, got.tableWidth(), tt.width)
			case tt.width > 0:
				assert.LessOrEqual(t, got.tableWidth(), tt.width)
			}
			if tt.wantCompact {
				assert.Equal(t, 11, got.started)
				assert.Equal(t, len("qwen-run-001"), got.mlflow, "MLflow is sized to its widest label")
			} else {
				// A terminal that fits today's table gets today's layout.
				assert.Equal(t, natural.fitWidth(tt.width), got)
				assert.Equal(t, mlflowColWidth, got.mlflow)
			}
		})
	}
}

// fullLabelListRows is wideListRows with MLflow labels at mlflowColWidth, the
// widest list.go produces and the usual case for a run named by MLflow. With
// these labels compacting saves only the 8 cells of Started.
func fullLabelListRows() []listRow {
	rows := wideListRows()
	for i := range rows {
		if rows[i].MLflowURL != "-" {
			rows[i].MLflowLabel = "…ne-2026-10-23-lr-3e-5"
			if lipgloss.Width(rows[i].MLflowLabel) != mlflowColWidth {
				panic("fixture label is not mlflowColWidth cells wide")
			}
		}
	}
	return rows
}

func TestLayoutListColsFullLabels(t *testing.T) {
	rows := fullLabelListRows()
	tests := []struct {
		name           string
		width          int
		wantExperiment int
		wantUser       int
		wantHidden     []string
	}{
		{name: "133 compacts Started only", width: 133, wantExperiment: 13, wantUser: 14},
		{name: "126 is the narrowest compact fit", width: 126, wantExperiment: experimentMinWidth, wantUser: userMinWidth},
		{name: "125 hides Accelerators", width: 125, wantExperiment: 16, wantUser: 17, wantHidden: []string{"Accelerators"}},
		{name: "120 hides Accelerators", width: 120, wantExperiment: 14, wantUser: 14, wantHidden: []string{"Accelerators"}},
		{name: "100 hides MLflow and User too", width: 100, wantExperiment: 34, wantHidden: []string{"MLflow", "User", "Accelerators"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := layoutListCols(rows, tt.width)
			assert.True(t, got.compactStarted)
			assert.Equal(t, mlflowColWidth, got.mlflow)
			assert.Equal(t, tt.wantExperiment, got.experiment)
			if got.shown(columnUser) {
				assert.Equal(t, tt.wantUser, got.user)
			}
			assert.Equal(t, tt.wantHidden, got.hiddenNames())
			assert.LessOrEqual(t, got.tableWidth(), tt.width)
		})
	}
}

func TestLayoutListColsMovesOneWayAsPagesLoad(t *testing.T) {
	// Loading more rows can only widen the natural columns, so a column that
	// is hidden stays hidden and a compact layout stays compact.
	rows := wideListRows()
	for _, width := range []int{80, 100, 120, 133, 160} {
		first := layoutListCols(rows[1:], width)
		all := layoutListCols(rows, width)
		assert.Equal(t, first.hidden, first.hidden&all.hidden, "width %d: a hidden column reappeared", width)
		if first.compactStarted {
			assert.True(t, all.compactStarted, "width %d: Started widened again", width)
		}
	}
	// The sample does cross a tier: the second page hides User at 100 columns.
	assert.Equal(t, []string{"Accelerators"}, layoutListCols(rows[1:], 100).hiddenNames())
	assert.Equal(t, []string{"User", "Accelerators"}, layoutListCols(rows, 100).hiddenNames())
}

func TestLayoutListColsCapsMLflowAtColumnWidth(t *testing.T) {
	// A label wider than the column (none is built that way) never widens it.
	rows := wideListRows()
	rows[0].MLflowLabel = strings.Repeat("x", mlflowColWidth+5)
	assert.Equal(t, mlflowColWidth, layoutListCols(rows, 120).mlflow)
}

func TestLayoutListColsMLflowFloorIsHeader(t *testing.T) {
	// With no MLflow links the compact column is just wide enough for its header.
	rows := wideListRows()
	for i := range rows {
		rows[i].MLflowURL, rows[i].MLflowLabel = "-", "-"
	}
	got := layoutListCols(rows, 120)
	assert.True(t, got.compactStarted)
	assert.Equal(t, len("MLflow"), got.mlflow)
}

func TestStartedCompact(t *testing.T) {
	assert.Equal(t, "-", startedCompact(listRow{}))
	assert.Equal(t, "06-05 17:32", startedCompact(listRow{StartedAt: new("2026-06-05T17:32:39.000000+00:00")}))
	assert.Equal(t, "2026-06", startedCompact(listRow{StartedAt: new("2026-06")}))
	assert.Equal(t, "Mon Jan  2 15:04:05 2006", startedCompact(listRow{StartedAt: new("Mon Jan  2 15:04:05 2006")}))
}

func TestHiddenNote(t *testing.T) {
	assert.Empty(t, hiddenNote(nil, true))
	assert.Empty(t, hiddenNote(nil, false))
	assert.Equal(t, "  ·  hidden: User, Accelerators", hiddenNote([]string{"User", "Accelerators"}, true))
	assert.Equal(t, "  ·  1 column hidden", hiddenNote([]string{"Accelerators"}, false))
	assert.Equal(t, "  ·  2 columns hidden", hiddenNote([]string{"User", "Accelerators"}, false))
}

func TestRenderHidesColumns(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	styles := newListStyles(r)
	rows := wideListRows()
	cols := layoutListCols(rows, 80)
	require.Equal(t, []string{"MLflow", "User", "Accelerators"}, cols.hiddenNames())

	header := styles.renderHeader(cols)
	assert.Equal(t, cols.tableWidth(), ansi.StringWidth(header))
	for _, name := range cols.hiddenNames() {
		assert.NotContains(t, header, name)
	}
	assert.Contains(t, header, "Progress")

	for _, selected := range []bool{false, true} {
		row := styles.renderRow(cols, rows[0], selected, true)
		assert.LessOrEqual(t, ansi.StringWidth(row), cols.tableWidth())
		assert.Contains(t, row, "10-23 17:32")
		assert.NotContains(t, row, "qwen-run-001")
		assert.NotContains(t, row, "8x H100")
		assert.NotContains(t, row, "runs/04c41514fbb0", "a hidden MLflow column carries no hyperlink")
	}
	assert.True(t, strings.HasPrefix(styles.renderRow(cols, rows[0], true, false), "▸"))
}

func TestListModelNarrowTerminalHidesColumns(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	f := &runFetcher{ctx: t.Context(), exhausted: true}
	m := newListModel(r, f, wideListRows(), false)

	view := func(width int) []string {
		next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		m = next.(listModel)
		lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
		require.Len(t, lines, 4) // header, two rows, hint
		for _, line := range lines {
			assert.LessOrEqual(t, ansi.StringWidth(line), width, line)
		}
		return lines
	}

	lines := view(80)
	assert.NotContains(t, lines[0], "User")
	assert.Contains(t, lines[3], "↑/↓ ←/→ ↵ i L q  ·  row 1/2  ·  hidden: MLflow, User, Accelerators")

	lines = view(100)
	assert.Contains(t, lines[0], "MLflow")
	assert.Contains(t, lines[3], "hidden: User, Accelerators")

	lines = view(120)
	assert.Contains(t, lines[0], "Accelerators")
	assert.Contains(t, lines[1], "10-23 17:32")
	assert.Contains(t, lines[3], "↑/↓ navigate · ←/→ page · ↵ mlflow · i info · L logs · q quit  ·  row 1/2")
	assert.NotContains(t, lines[3], "hidden")

	// Widening past today's threshold restores the full timestamp.
	lines = view(160)
	assert.Contains(t, lines[1], "2026-10-23T17:32:39")
}

func TestListModelHintNamesHiddenColumnsWithFullLegend(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	f := &runFetcher{ctx: t.Context(), exhausted: true}
	m := newListModel(r, f, fullLabelListRows(), false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m = next.(listModel)
	lines := strings.Split(strings.TrimRight(m.View(), "\n"), "\n")
	require.Len(t, lines, 4)
	assert.NotContains(t, lines[0], "Accelerators")
	assert.Contains(t, lines[0], "MLflow")
	assert.Equal(t, hintLegend+"  ·  row 1/2  ·  hidden: Accelerators", lines[3])
	assert.LessOrEqual(t, ansi.StringWidth(lines[3]), 120)
}

func TestListModelHintFitsNarrowTerminal(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	// Enough rows for a three-digit counter, all with wide values so every
	// hideable column goes at the narrowest widths.
	var rows []listRow
	for len(rows) < 120 {
		rows = append(rows, wideListRows()...)
	}
	for _, width := range []int{40, 46, 70, 75, 80, 100} {
		for _, state := range []string{"idle", "loading", "failed"} {
			t.Run(fmt.Sprintf("%d/%s", width, state), func(t *testing.T) {
				f := &runFetcher{ctx: t.Context(), exhausted: true}
				m := newListModel(r, f, rows, false)
				next, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
				m = next.(listModel)
				switch state {
				case "loading":
					m.loading = true
				case "failed":
					m.loadErr = errors.New("boom")
				}
				m.cursor = len(rows) - 1
				hint := m.renderHint()
				assert.LessOrEqual(t, ansi.StringWidth(hint), width, hint)
				// The row counter and paging state come first, so they survive
				// every step down.
				assert.Contains(t, hint, "row 120/120")
				switch state {
				case "loading":
					assert.Contains(t, hint, "(loading…)")
				case "failed":
					assert.Contains(t, hint, "(load failed)")
				}
				// Only the last-resort truncation, which needs a terminal
				// narrower than the table's own minimum, can cut the note or
				// make the idle and loading hints differ. Otherwise the variant
				// is chosen for the widest counter and the loading state, so it
				// doesn't change while a page loads.
				idle, loading := m, m
				idle.loading, idle.loadErr = false, nil
				loading.loading, loading.loadErr = true, nil
				loadingHint := loading.renderHint()
				if strings.HasSuffix(loadingHint, "…") || strings.HasSuffix(hint, "…") {
					assert.Less(t, width, 48, hint)
					return
				}
				assert.True(t, strings.Contains(hint, "hidden: ") || strings.Contains(hint, "columns hidden"), hint)
				assert.Equal(t, idle.renderHint(), strings.Replace(loadingHint, " (loading…)", "", 1))
			})
		}
	}
}

func TestListModelRelayoutsLoadedPageInNarrowTerminal(t *testing.T) {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	rows := wideListRows()
	f := &runFetcher{ctx: t.Context(), exhausted: true}
	m := newListModel(r, f, rows[1:], false)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = next.(listModel)
	require.Equal(t, []string{"Accelerators"}, m.cols.hiddenNames())

	next, _ = m.Update(moreRowsMsg{rows: rows[:1]})
	m = next.(listModel)
	assert.Equal(t, []string{"User", "Accelerators"}, m.cols.hiddenNames())
	for line := range strings.SplitSeq(strings.TrimRight(m.View(), "\n"), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 100, line)
	}
}

func TestListModelEnterOpensMLflowLogsWhenColumnHidden(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, "BROWSER", "none")
	r, _ := cmdio.NewRenderer(ctx, io.Discard)
	f := &runFetcher{ctx: ctx, w: newTestWorkspaceClient(t, "https://x.test")}
	m := newListModel(r, f, testListRows(), false)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = next.(listModel)
	require.False(t, m.cols.shown(columnMLflow))

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	cmd()
	assert.Contains(t, stderr.String(), "https://h/ml/experiments/E/runs/04c41514fbb0/logs")
}
