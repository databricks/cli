package aircmd

import (
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/muesli/termenv"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// renderListText renders the table for text output: an inline navigable table
// in a terminal, otherwise printed once. JSON is handled by the caller.
func renderListText(cmd *cobra.Command, rows []listRow) error {
	ctx := cmd.Context()
	out := cmd.OutOrStdout()

	color := cmdio.SupportsColor(ctx, out)
	r := lipgloss.NewRenderer(out)
	if !color {
		r.SetColorProfile(termenv.Ascii)
	}

	// Navigate only with a full color TTY, at least one row, and no explicit
	// --limit (which means "just print these N"). Everything else — piped,
	// NO_COLOR, --limit, empty — prints once.
	interactive := len(rows) > 0 &&
		color &&
		cmdio.IsPagerSupported(ctx) &&
		!cmd.Flags().Changed("limit")

	if interactive {
		_, err := tea.NewProgram(
			newListModel(r, rows, color),
			tea.WithContext(ctx),
			tea.WithInput(cmd.InOrStdin()),
			tea.WithOutput(out),
		).Run()
		return err
	}

	_, err := io.WriteString(out, staticListTable(r, rows, color))
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

// listModel is the inline, navigable runs table.
type listModel struct {
	rows   []listRow
	styles listStyles
	cols   listCols
	links  bool

	cursor int
	offset int // index of the first visible row
	height int // terminal height, for windowing
}

func newListModel(r *lipgloss.Renderer, rows []listRow, links bool) listModel {
	return listModel{
		rows:   rows,
		styles: newListStyles(r),
		cols:   computeListCols(rows),
		links:  links,
	}
}

func (m listModel) Init() tea.Cmd { return nil }

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
		m.offset = m.clampedOffset()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
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
		}
		m.offset = m.clampedOffset()
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

// renderHint is the faint one-line key legend, with a scroll position when the
// list is windowed.
func (m listModel) renderHint() string {
	faint := m.styles.r.NewStyle().Foreground(colN7)
	hint := "↑/↓ navigate · ←/→ page · ↵ mlflow · q quit"
	if m.visibleCount() < len(m.rows) {
		hint += fmt.Sprintf("  ·  row %d/%d", m.cursor+1, len(m.rows))
	}
	return faint.Render(hint)
}

// openURL opens a URL in the user's default browser, best-effort.
func openURL(url string) tea.Cmd {
	return func() tea.Msg {
		_ = browser.OpenURL(url)
		return nil
	}
}
