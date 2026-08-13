package aircmd

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// renderListText renders the table for text output: an inline navigable table in
// a terminal (paging in older runs on demand), otherwise printed once. JSON is
// handled by the caller.
func renderListText(cmd *cobra.Command, f *runFetcher, limit int) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	r, color := cmdio.NewRenderer(ctx, out)

	// Navigate only with a full color TTY and no explicit --limit (which means
	// "just print these N"). Everything else — piped, NO_COLOR, --limit — prints
	// once.
	interactive := color &&
		cmdio.IsPagerSupported(ctx) &&
		!cmd.Flags().Changed("limit")

	if interactive {
		first, err := f.next(listPageRows)
		if err != nil {
			return err
		}
		if len(first) == 0 {
			_, err := io.WriteString(out, "No runs found.\n")
			return err
		}
		_, err = tea.NewProgram(
			newListModel(r, f, first, color),
			tea.WithContext(ctx),
			tea.WithInput(cmd.InOrStdin()),
			tea.WithOutput(out),
		).Run()
		return err
	}

	rows, err := f.next(limit)
	if err != nil {
		return err
	}
	warnIfTruncated(ctx, f)
	_, err = io.WriteString(out, staticListTable(r, rows, color))
	return err
}

// staticListTable renders the whole table once, with no selection — used when
// piped or non-interactive.
func staticListTable(r *lipgloss.Renderer, rows []listRow, links bool) string {
	if len(rows) == 0 {
		return "No runs found.\n"
	}
	styles := newListStyles(r)
	cols := computeListCols(rows)
	var b strings.Builder
	b.WriteString(styles.renderHeader(cols))
	b.WriteByte('\n')
	for _, row := range rows {
		b.WriteString(styles.renderRow(cols, row, false, links))
		b.WriteByte('\n')
	}
	return b.String()
}

// Mode constants for the TUI.
const (
	modeList   = 0
	modeDetail = 1
)

// detailMsg carries fetched run details or error.
type detailMsg struct {
	title string
	body  string
	err   error
}

// listModel is the inline, navigable runs table. It lazily pages older runs from
// the fetcher as the cursor nears the end of the loaded rows. fetcher is nil for
// a fixed, non-paging table (e.g. in tests).
type listModel struct {
	rows    []listRow
	styles  listStyles
	cols    listCols
	links   bool
	fetcher *runFetcher
	loading bool
	loadErr error

	cursor int
	offset int // index of the first visible row
	height int // terminal height, for windowing

	mode          int            // modeList or modeDetail
	viewport      viewport.Model // for detail pane
	detailTitle   string         // title of detail pane
	detailLoading bool           // loading detail
	detailContent string         // rendered detail content
}

func newListModel(r *lipgloss.Renderer, f *runFetcher, rows []listRow, links bool) listModel {
	return listModel{
		rows:     rows,
		styles:   newListStyles(r),
		cols:     computeListCols(rows),
		links:    links,
		fetcher:  f,
		mode:     modeList,
		viewport: viewport.New(0, 0),
	}
}

func (m listModel) Init() tea.Cmd { return nil }

// moreRowsMsg carries a lazily fetched batch of rows, or the error that ended paging.
type moreRowsMsg struct {
	rows []listRow
	err  error
}

// fetchCmd pulls the next batch of rows in the background; sets loading so only
// one runs at a time, and returns the updated model with the fetch command.
func (m listModel) fetchCmd() (listModel, tea.Cmd) {
	m.loading = true
	f := m.fetcher
	return m, func() tea.Msg {
		rows, err := f.next(listPageRows)
		return moreRowsMsg{rows: rows, err: err}
	}
}

// maybeFetch starts a fetch when the cursor nears the end of the loaded rows and
// more runs may still exist.
func (m listModel) maybeFetch() (listModel, tea.Cmd) {
	if m.fetcher == nil || m.loading || m.loadErr != nil || m.fetcher.exhausted {
		return m, nil
	}
	if m.cursor < len(m.rows)-m.visibleCount() {
		return m, nil
	}
	return m.fetchCmd()
}

// listPageRows is the most rows shown per page.
const listPageRows = 20

// visibleCount is how many rows a page shows: at most listPageRows, and never
// more than fits below the header and hint.
func (m listModel) visibleCount() int {
	n := min(listPageRows, len(m.rows))
	if m.height > 0 {
		n = min(n, m.height-3)
	}
	return max(1, n)
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = max(msg.Height-3, 1)
		m.offset = m.clampedOffset()
		return m.maybeFetch()

	case moreRowsMsg:
		m.loading = false
		if msg.err != nil {
			m.loadErr = msg.err
			return m, nil
		}
		m.rows = append(m.rows, msg.rows...)
		m.cols = computeListCols(m.rows)
		m.offset = m.clampedOffset()
		// A page with no matches but more to scan: keep paging so the cursor isn't
		// stuck at the end of the loaded rows.
		if len(msg.rows) == 0 && !m.fetcher.exhausted {
			return m.fetchCmd()
		}
		return m, nil

	case detailMsg:
		m.detailLoading = false
		if msg.err != nil {
			m.detailContent = fmt.Sprintf("Error: %v", msg.err)
		} else {
			m.detailContent = msg.body
		}
		m.detailTitle = msg.title
		m.viewport.SetContent(m.detailContent)
		m.viewport.GotoTop()
		m.mode = modeDetail
		return m, nil

	case tea.KeyMsg:
		// Detail pane key handling.
		if m.mode == modeDetail {
			switch msg.String() {
			case "esc", "q":
				m.mode = modeList
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				// Delegate scrolling keys to the viewport.
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}
		}

		// List pane key handling.
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.rows)-1 {
				m.cursor++
			}
		case "right":
			m.cursor = min(m.cursor+m.visibleCount(), len(m.rows)-1)
		case "left":
			m.cursor = max(m.cursor-m.visibleCount(), 0)
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.rows) - 1
		case "enter":
			// Open the selected run's MLflow page in the browser.
			if len(m.rows) > 0 {
				if url := m.rows[m.cursor].MLflowURL; url != "" && url != "-" {
					return m, openURL(url)
				}
			}
		case "i":
			// Open the run details pane; the fetch fills it in.
			if m.fetcher != nil && len(m.rows) > 0 {
				m.mode = modeDetail
				m.detailLoading = true
				m.detailTitle = "Run details"
				runID, _ := parseRunID(m.rows[m.cursor].RunID)
				return m, m.fetchRunDetail(runID)
			}
		case "L", "l":
			// Open the logs snapshot pane; the fetch fills it in.
			if m.fetcher != nil && len(m.rows) > 0 {
				m.mode = modeDetail
				m.detailLoading = true
				m.detailTitle = "Logs snapshot"
				runID, _ := parseRunID(m.rows[m.cursor].RunID)
				return m, m.fetchRunLogs(runID)
			}
		}
		m.offset = m.clampedOffset()
		return m.maybeFetch()
	}
	return m, nil
}

// clampedOffset returns the scroll offset that keeps the cursor visible.
func (m listModel) clampedOffset() int {
	visible := m.visibleCount()
	offset := min(m.offset, m.cursor)
	if m.cursor >= offset+visible {
		offset = m.cursor - visible + 1
	}
	return max(offset, 0)
}

func (m listModel) View() string {
	if m.mode == modeDetail {
		faint := m.styles.r.NewStyle().Foreground(colN7)
		lines := []string{m.detailTitle}
		if m.detailLoading {
			lines = append(lines, "Loading…")
		} else {
			lines = append(lines, m.viewport.View())
		}
		footer := faint.Render("↑/↓ scroll · esc back · q quit")
		lines = append(lines, footer)
		return strings.Join(lines, "\n")
	}

	if len(m.rows) == 0 {
		return m.styles.r.NewStyle().Foreground(colN9).Render("No runs found.") + "\n"
	}

	visible := m.visibleCount()
	lines := []string{m.styles.renderHeader(m.cols)}
	for i := m.offset; i < m.offset+visible && i < len(m.rows); i++ {
		lines = append(lines, m.styles.renderRow(m.cols, m.rows[i], i == m.cursor, m.links))
	}
	lines = append(lines, m.renderHint())
	return strings.Join(lines, "\n") + "\n"
}

// renderHint is the faint one-line key legend, with the cursor position and the
// paging state (loading / load failed).
func (m listModel) renderHint() string {
	faint := m.styles.r.NewStyle().Foreground(colN7)
	hint := fmt.Sprintf("↑/↓ navigate · ←/→ page · ↵ mlflow · i info · L logs · q quit  ·  row %d/%d", m.cursor+1, len(m.rows))
	switch {
	case m.loadErr != nil:
		hint += " (load failed)"
	case m.loading:
		hint += " (loading…)"
	}
	return faint.Render(hint)
}

// fetchRunDetail returns a tea.Cmd that fetches run details in the background.
func (m listModel) fetchRunDetail(runID int64) tea.Cmd {
	ctx := m.fetcher.ctx
	w := m.fetcher.w
	return func() tea.Msg {
		body, err := runDetailText(ctx, w, runID)
		if err != nil {
			return detailMsg{title: "Run Details", err: err}
		}
		return detailMsg{title: "Run Details", body: body}
	}
}

// fetchRunLogs returns a tea.Cmd that fetches a log snapshot in the background.
func (m listModel) fetchRunLogs(runID int64) tea.Cmd {
	ctx := m.fetcher.ctx
	w := m.fetcher.w
	return func() tea.Msg {
		body, err := runLogsSnapshot(ctx, w, runID)
		if err != nil {
			return detailMsg{title: "Logs Snapshot", err: err}
		}
		return detailMsg{title: "Logs Snapshot", body: body}
	}
}

// openURL opens a URL in the user's default browser, best-effort.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		_ = browser.OpenURL(url)
		return nil
	}
}

// parseRunID parses a run id string to int64. Rows carry a formatted int64, so
// this only fails on an unexpectedly malformed value.
func parseRunID(runIDStr string) (int64, error) {
	return strconv.ParseInt(runIDStr, 10, 64)
}
