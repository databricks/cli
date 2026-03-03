// Package tableview provides an interactive table browser with scrolling and search.
package tableview

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	horizontalScrollStep = 4
	footerHeight         = 1
	searchFooterHeight   = 2
	// headerLines is the number of non-data lines at the top (header + separator).
	headerLines = 2
)

var (
	searchHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("228")).Foreground(lipgloss.Color("0"))
	cursorStyle          = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229"))
	footerStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	searchStyle          = lipgloss.NewStyle().Foreground(lipgloss.Color("229"))
)

// Run displays tabular data in an interactive browser.
// Writes to w (typically stdout). Blocks until user quits.
func Run(w io.Writer, columns []string, rows [][]string) error {
	lines := renderTableLines(columns, rows)

	m := model{
		lines:  lines,
		cursor: headerLines, // Start on first data row.
	}

	p := tea.NewProgram(m, tea.WithOutput(w))
	_, err := p.Run()
	return err
}

// renderTableLines produces aligned table text as individual lines.
func renderTableLines(columns []string, rows [][]string) []string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	// Header.
	fmt.Fprintln(tw, strings.Join(columns, "\t"))

	// Separator: compute widths from header + data for dash line.
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col)
	}
	for _, row := range rows {
		for i := range columns {
			if i < len(row) {
				widths[i] = max(widths[i], len(row[i]))
			}
		}
	}
	seps := make([]string, len(columns))
	for i, w := range widths {
		seps[i] = strings.Repeat("─", w)
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))

	// Data rows.
	for _, row := range rows {
		vals := make([]string, len(columns))
		for i := range columns {
			if i < len(row) {
				vals[i] = row[i]
			}
		}
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
func highlightSearch(line, query string) string {
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
		b.WriteString(searchHighlightStyle.Render(line[pos+idx : pos+idx+qLen]))
		pos += idx + qLen
	}
	return b.String()
}

// renderContent builds the viewport content with cursor and search highlighting.
// Search highlighting is applied first on clean text, then cursor style wraps the result.
func (m model) renderContent() string {
	result := make([]string, len(m.lines))
	for i, line := range m.lines {
		rendered := highlightSearch(line, m.searchQuery)
		if i == m.cursor {
			rendered = cursorStyle.Render(rendered)
		}
		result[i] = rendered
	}
	return strings.Join(result, "\n")
}

type model struct {
	viewport viewport.Model
	lines    []string
	ready    bool
	cursor   int // line index of the highlighted row

	// Search state.
	searching   bool
	searchInput string
	searchQuery string
	matchLines  []int
	matchIdx    int
}

func (m model) dataRowCount() int {
	return max(len(m.lines)-headerLines, 0)
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
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-fh)
			m.viewport.SetHorizontalStep(horizontalScrollStep)
			m.viewport.SetContent(m.renderContent())
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - fh
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
		m.cursor = headerLines
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
	m.cursor = max(m.cursor, headerLines)
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
	return m.viewport.View() + "\n" + footer
}

func (m model) renderFooter() string {
	if m.searching {
		prompt := searchStyle.Render("/ " + m.searchInput + "█")
		return footerStyle.Render(fmt.Sprintf("%d rows", m.dataRowCount())) + "\n" + prompt
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

	return footerStyle.Render(strings.Join(parts, " | "))
}
