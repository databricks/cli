// Package tableprint renders static, terminal-width-aware text tables.
// It fits columns to the available width, drops trailing columns,
// truncates wide cells, and works identically with an interactive terminal or a pipe.
package tableprint

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"golang.org/x/term"
)

const (
	maxColumnWidth = 20
	columnSep      = "  "
	ellipsis       = "..."
	separatorChar  = "-"
)

// writes table (with an "N rows" footer).
// if table is too wide trailing columns are dropped with a "(showing M of N columns)" note.
// cells are truncated to maxColumnWidth.
func Render(ctx context.Context, out io.Writer, columns []string, rows [][]string, width int) error {
	if len(columns) == 0 {
		return nil
	}
	n := len(columns)

	colWidth := make([]int, n)
	for i, col := range columns {
		colWidth[i] = min(cmdio.Width(col), maxColumnWidth)
	}
	for _, row := range rows {
		for i := 0; i < n && i < len(row); i++ {
			colWidth[i] = max(colWidth[i], min(cmdio.Width(row[i]), maxColumnWidth))
		}
	}

	shown, cropped := fitColumns(colWidth, width)

	widths := append([]int(nil), colWidth[:shown]...)
	if cropped {
		widths = append(widths, cmdio.Width(ellipsis))
	}

	header := make([]string, 0, len(widths))
	for i := range shown {
		header = append(header, truncateToWidth(columns[i], colWidth[i]))
	}
	if cropped {
		header = append(header, cmdio.Faint(ctx, ellipsis))
	}
	emitRow(out, header, widths)

	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat(separatorChar, w)
	}
	emitRow(out, sep, widths)

	for _, row := range rows {
		cells := make([]string, 0, len(widths))
		for i := range shown {
			cell := ""
			if i < len(row) {
				cell = truncateToWidth(row[i], colWidth[i])
			}
			cells = append(cells, cell)
		}
		if cropped {
			cells = append(cells, cmdio.Faint(ctx, ellipsis))
		}
		emitRow(out, cells, widths)
	}

	fmt.Fprintln(out)
	if cropped {
		fmt.Fprintln(out, cmdio.Faint(ctx, fmt.Sprintf("(showing %d of %d columns)", shown, n)))
	}
	fmt.Fprintf(out, "%d rows\n", len(rows))
	return nil
}

// emitRow writes one table line. Every cell but the last is padded to its
// column width; the last is left unpadded to avoid trailing whitespace.
func emitRow(out io.Writer, cells []string, widths []int) {
	parts := make([]string, len(cells))
	last := len(cells) - 1
	for i, c := range cells {
		if i == last {
			parts[i] = c
		} else {
			parts[i] = cmdio.PadRight(c, widths[i])
		}
	}
	fmt.Fprintln(out, strings.Join(parts, columnSep))
}

// figures out how many columns fit within width and whether any
// were dropped. width <= 0 means unlimited
func fitColumns(colWidth []int, width int) (shown int, cropped bool) {
	n := len(colWidth)
	total := 0
	for i, w := range colWidth {
		if i > 0 {
			total += cmdio.Width(columnSep)
		}
		total += w
	}
	if width <= 0 || total <= width {
		return n, false
	}
	budget := width - cmdio.Width(columnSep) - cmdio.Width(ellipsis)
	used := 0
	for i := range colWidth {
		add := colWidth[i]
		if i > 0 {
			add += cmdio.Width(columnSep)
		}
		if used+add > budget {
			s := max(i, 1)
			return s, s < n
		}
		used += add
	}
	return n, false
}

// shortens s so its display width does not exceed maxWidth,
// appending "..." to mark truncation.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if cmdio.Width(s) <= maxWidth {
		return s
	}
	budget := maxWidth - cmdio.Width(ellipsis)
	if budget <= 0 {
		return cutToWidth(s, maxWidth)
	}
	return cutToWidth(s, budget) + ellipsis
}

// cutToWidth returns the longest prefix of s whose display width is <= w
func cutToWidth(s string, w int) string {
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := cmdio.Width(string(r))
		if used+rw > w {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	return b.String()
}

// DetectWidth returns the terminal width, or 0 (meaning "no cap")
// when out is not an interactive terminal
func DetectWidth(out io.Writer) int {
	f, ok := out.(*os.File)
	if !ok {
		return 0
	}
	if !cmdio.IsOutputTTY(f) {
		return 0
	}
	width, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return width
}
