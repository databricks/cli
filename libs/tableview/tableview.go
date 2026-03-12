// Package tableview provides an interactive table browser with scrolling and search.
package tableview

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
			scrollViewportToCursor(&m.viewport, m.cursor)
		}
		return m, nil
	case "N":
		if len(m.matchLines) > 0 {
			m.matchIdx = (m.matchIdx - 1 + len(m.matchLines)) % len(m.matchLines)
			m.cursor = m.matchLines[m.matchIdx]
			m.viewport.SetContent(m.renderContent())
			scrollViewportToCursor(&m.viewport, m.cursor)
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
	scrollViewportToCursor(&m.viewport, m.cursor)
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
			scrollViewportToCursor(&m.viewport, m.cursor)
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
