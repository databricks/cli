package tableview

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/viewport"
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

// renderTableLines produces aligned table text as individual lines.
func renderTableLines(columns []string, rows [][]string) []string {
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 4, 2, ' ', 0)

	// Header.
	fmt.Fprintln(tw, strings.Join(columns, "\t"))

	// Separator: compute widths from header + data for dash line.
	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range rows {
		for i := range columns {
			if i < len(row) {
				widths[i] = max(widths[i], utf8.RuneCountInString(row[i]))
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
// It works in rune-space so that case-folding length changes (e.g. "ß"→"ss")
// do not misalign the highlighted spans in the original string.
func highlightSearch(line, query string) string {
	if query == "" {
		return line
	}
	lineRunes := []rune(line)
	queryRunes := []rune(strings.ToLower(query))
	lineLower := []rune(strings.ToLower(line))
	qLen := len(queryRunes)

	var b strings.Builder
	pos := 0
	for pos <= len(lineLower)-qLen {
		match := false
		for i := range qLen {
			if lineLower[pos+i] != queryRunes[i] {
				break
			}
			if i == qLen-1 {
				match = true
			}
		}
		if !match {
			b.WriteRune(lineRunes[pos])
			pos++
			continue
		}
		b.WriteString(searchHighlightStyle.Render(string(lineRunes[pos : pos+qLen])))
		pos += qLen
	}
	// Write remaining runes after last possible match position.
	b.WriteString(string(lineRunes[pos:]))
	return b.String()
}

// scrollViewportToCursor ensures the cursor line is visible in the viewport.
func scrollViewportToCursor(vp *viewport.Model, cursor int) {
	top := vp.YOffset
	bottom := top + vp.Height - 1
	if cursor < top {
		vp.SetYOffset(cursor)
	} else if cursor > bottom {
		vp.SetYOffset(cursor - vp.Height + 1)
	}
}

// RenderStaticTable renders a non-interactive table to the writer.
// This is used as fallback when the terminal doesn't support full interactivity.
func RenderStaticTable(w io.Writer, columns []string, rows [][]string) error {
	const maxColumnWidth = 40

	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	// Header
	fmt.Fprintln(tw, strings.Join(columns, "\t"))
	// Separator
	seps := make([]string, len(columns))
	for i, col := range columns {
		width := utf8.RuneCountInString(col)
		for _, row := range rows {
			if i < len(row) {
				width = max(width, min(utf8.RuneCountInString(row[i]), maxColumnWidth))
			}
		}
		seps[i] = strings.Repeat("-", width)
	}
	fmt.Fprintln(tw, strings.Join(seps, "\t"))
	// Data rows (no cell truncation; truncation is a TUI display concern)
	for _, row := range rows {
		vals := make([]string, len(columns))
		for i := range columns {
			if i < len(row) {
				vals[i] = row[i]
			}
		}
		fmt.Fprintln(tw, strings.Join(vals, "\t"))
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\n%d rows\n", len(rows))
	return err
}
