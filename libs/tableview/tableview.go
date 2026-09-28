// Package tableview provides an interactive table browser with scrolling and search.
package tableview

import (
	"context"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/databricks/cli/libs/cmdio"
)

const (
	horizontalScrollStep = 4
	footerHeight         = 1
	searchFooterHeight   = 2
	// headerLines is the number of non-data lines at the top (header + separator).
	// These are rendered above the viewport so they stay visible while data scrolls.
	headerLines = 2
	// maxCellWidth bounds each cell so a single very large value cannot inflate a
	// column to the point of degrading rendering and scrolling.
	maxCellWidth = 256
)

// truncateCell shortens s to maxCellWidth, marking truncation with an ellipsis.
func truncateCell(s string) string {
	const ellipsis = "..."
	if len(s) <= maxCellWidth {
		return s
	}
	r := []rune(s)
	if len(r) <= maxCellWidth {
		return s
	}
	return string(r[:maxCellWidth-len(ellipsis)]) + ellipsis
}

// Run displays tabular data in an interactive browser.
// Writes to w (typically stdout). Blocks until user quits.
func Run(ctx context.Context, w io.Writer, columns []string, rows [][]string) error {
	all := renderTableLines(columns, rows)
	header := all[:headerLines]
	dataLines := all[headerLines:]

	r, _ := cmdio.NewRenderer(ctx, w)
	m := model{
		header:               header,
		lines:                dataLines,
		cursor:               0,
		searchHighlightStyle: r.NewStyle().Background(lipgloss.Color("228")).Foreground(lipgloss.Color("0")),
		cursorStyle:          r.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229")),
		footerStyle:          r.NewStyle().Foreground(lipgloss.Color("241")),
		searchStyle:          r.NewStyle().Foreground(lipgloss.Color("229")),
	}

	p := tea.NewProgram(m, tea.WithOutput(w))
	_, err := p.Run()
	return err
}

// renderTableLines produces aligned table text as individual lines.
func renderTableLines(columns []string, rows [][]string) []string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	// Header and data cells are truncated once so widths and emitted values agree.
	header := make([]string, len(columns))
	for i, col := range columns {
		header[i] = truncateCell(col)
	}
	fmt.Fprintln(tw, strings.Join(header, "\t"))

	cells := make([][]string, len(rows))
	widths := make([]int, len(columns))
	for i := range columns {
		widths[i] = len(header[i])
	}
	for r, row := range rows {
		vals := make([]string, len(columns))
		for i := range columns {
			if i < len(row) {
				vals[i] = truncateCell(row[i])
				widths[i] = max(widths[i], len(vals[i]))
			}
		}
		cells[r] = vals
	}

	// Separator dash line, sized to the truncated column widths.
	seps := make([]string, len(columns))
	for i, w := range widths {
		seps[i] = strings.Repeat("─", w)
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	for _, vals := range cells {
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}

	tw.Flush()

	// Split into lines, drop trailing empty.
	lines := strings.Split(buf.String(), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// findMatches returns line indices containing the query (case-insensitive).
func findMatches(lines []string, query string) []int {
	if query == "" {
		return nil
	}
	lower := strings.ToLower(query)
	var matches []int
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lower) {
			matches = append(matches, i)
		}
	}
	return matches
}

// highlightSearch applies search match highlighting to a single line.
func (m model) highlightSearch(line, query string) string {
	if query == "" {
		return line
	}
	lower := strings.ToLower(query)
	qLen := len(query)
	lineLower := strings.ToLower(line)

	var b strings.Builder
	pos := 0
	for {
		idx := strings.Index(lineLower[pos:], lower)
		if idx < 0 {
			b.WriteString(line[pos:])
			break
		}
		b.WriteString(line[pos : pos+idx])
		b.WriteString(m.searchHighlightStyle.Render(line[pos+idx : pos+idx+qLen]))
		pos += idx + qLen
	}
	return b.String()
}

// renderContent builds the viewport content with cursor and search highlighting.
// Search highlighting is applied first on clean text, then cursor style wraps the result.
func (m model) renderContent() string {
	result := make([]string, len(m.lines))
	for i, line := range m.lines {
		rendered := m.highlightSearch(line, m.searchQuery)
		if i == m.cursor {
			rendered = m.cursorStyle.Render(rendered)
		}
		result[i] = rendered
	}
	return strings.Join(result, "\n")
}

type model struct { //nolint:recvcheck // value receivers for tea.Model interface, pointer for cursor mutation
	viewport viewport.Model
	header   []string // sticky header lines (column names + separator)
	lines    []string // data rows only
	ready    bool
	cursor   int // index into lines (data rows)

	// Search state.
	searching   bool
	searchInput string
	searchQuery string
	matchLines  []int // indices into lines
	matchIdx    int

	// Styles, minted from a writer-scoped renderer in Run.
	searchHighlightStyle lipgloss.Style
	cursorStyle          lipgloss.Style
	footerStyle          lipgloss.Style
	searchStyle          lipgloss.Style
}

func (m model) dataRowCount() int {
	return len(m.lines)
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		fh := footerHeight
		if m.searching {
			fh = searchFooterHeight
		}
		// Reserve room for the sticky header above the viewport.
		height := msg.Height - fh - len(m.header)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, height)
			m.viewport.SetHorizontalStep(horizontalScrollStep)
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = height
		}
		return m, nil

	case tea.KeyMsg:
		if m.searching {
			return m.updateSearch(msg)
		}
		return m.updateNormal(msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "/":
		m.searching = true
		m.searchInput = ""
		m.viewport.Height--
		return m, nil
	case "n":
		if len(m.matchLines) > 0 {
			m.matchIdx = (m.matchIdx + 1) % len(m.matchLines)
			m.cursor = m.matchLines[m.matchIdx]
			m.viewport.SetContent(m.renderContent())
			m.scrollToCursor()
		}
		return m, nil
	case "N":
		if len(m.matchLines) > 0 {
			m.matchIdx = (m.matchIdx - 1 + len(m.matchLines)) % len(m.matchLines)
			m.cursor = m.matchLines[m.matchIdx]
			m.viewport.SetContent(m.renderContent())
			m.scrollToCursor()
		}
		return m, nil
	case "up", "k":
		m.moveCursor(-1)
		return m, nil
	case "down", "j":
		m.moveCursor(1)
		return m, nil
	case "pgup", "b":
		m.moveCursor(-m.viewport.Height)
		return m, nil
	case "pgdown", "f", " ":
		m.moveCursor(m.viewport.Height)
		return m, nil
	case "g":
		m.cursor = 0
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoTop()
		return m, nil
	case "G":
		m.cursor = len(m.lines) - 1
		m.viewport.SetContent(m.renderContent())
		m.viewport.GotoBottom()
		return m, nil
	}

	// Let viewport handle horizontal scroll and other keys.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// moveCursor moves the cursor by delta lines, clamped to data rows.
func (m *model) moveCursor(delta int) {
	m.cursor += delta
	m.cursor = max(m.cursor, 0)
	m.cursor = min(m.cursor, len(m.lines)-1)
	m.viewport.SetContent(m.renderContent())
	m.scrollToCursor()
}

// scrollToCursor ensures the cursor line is visible in the viewport.
func (m *model) scrollToCursor() {
	top := m.viewport.YOffset
	bottom := top + m.viewport.Height - 1
	if m.cursor < top {
		m.viewport.SetYOffset(m.cursor)
	} else if m.cursor > bottom {
		m.viewport.SetYOffset(m.cursor - m.viewport.Height + 1)
	}
}

func (m model) updateSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.searching = false
		m.searchQuery = m.searchInput
		m.matchLines = findMatches(m.lines, m.searchQuery)
		m.matchIdx = 0
		m.viewport.Height++
		// Move cursor to first match and re-render.
		if len(m.matchLines) > 0 {
			m.cursor = m.matchLines[0]
		}
		m.viewport.SetContent(m.renderContent())
		if len(m.matchLines) > 0 {
			m.scrollToCursor()
		}
		return m, nil
	case "esc", "ctrl+c":
		m.searching = false
		m.searchInput = ""
		m.viewport.Height++
		return m, nil
	case "backspace":
		if len(m.searchInput) > 0 {
			m.searchInput = m.searchInput[:len(m.searchInput)-1]
		}
		return m, nil
	default:
		// Only accept printable characters.
		if len(msg.String()) == 1 || msg.Type == tea.KeyRunes {
			m.searchInput += msg.String()
		}
		return m, nil
	}
}

func (m model) View() string {
	if !m.ready {
		return "Loading..."
	}

	footer := m.renderFooter()
	header := strings.Join(m.header, "\n")
	return header + "\n" + m.viewport.View() + "\n" + footer
}

func (m model) renderFooter() string {
	if m.searching {
		prompt := m.searchStyle.Render("/ " + m.searchInput + "█")
		return m.footerStyle.Render(fmt.Sprintf("%d rows", m.dataRowCount())) + "\n" + prompt
	}

	parts := []string{fmt.Sprintf("%d rows", m.dataRowCount())}

	if m.searchQuery != "" && len(m.matchLines) > 0 {
		parts = append(parts, fmt.Sprintf("match %d/%d", m.matchIdx+1, len(m.matchLines)))
		parts = append(parts, "n/N next/prev")
	} else if m.searchQuery != "" {
		parts = append(parts, "no matches")
	}

	parts = append(parts, "←→↑↓ scroll", "g/G top/bottom", "/ search", "q quit")

	pct := int(m.viewport.ScrollPercent() * 100)
	parts = append(parts, fmt.Sprintf("%d%%", pct))

	return m.footerStyle.Render(strings.Join(parts, " | "))
}
