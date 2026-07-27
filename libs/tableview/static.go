// This file is the static, terminal-width-aware rendering path, used for
// non-interactive output; the interactive browser lives in tableview.go.
package tableview

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
	maxColumnWidth = 40
	columnSep      = "  "
	ellipsis       = "..."
	separatorChar  = "-"
)

// StaticOptions configures RenderStatic. Width is the terminal column budget
// (0 means unlimited, so no columns are dropped). TruncateCells caps each cell
// at maxColumnWidth with an ellipsis; when false, cells are emitted in full and
// only the separator dash count is capped.
type StaticOptions struct {
	Width         int
	TruncateCells bool
}

// RenderStatic writes columns and rows as an aligned text table with an
// "N rows" footer. When opts.Width is positive and the table is wider, trailing
// columns are dropped in favor of a "(showing M of N columns)" note.
func RenderStatic(ctx context.Context, out io.Writer, columns []string, rows [][]string, opts StaticOptions) error {
	if len(columns) == 0 && opts.TruncateCells {
		return nil
	}
	n := len(columns)

	colWidth := make([]int, n)
	for i, col := range columns {
		colWidth[i] = cellWidth(col, opts.TruncateCells)
	}
	for _, row := range rows {
		for i := 0; i < n && i < len(row); i++ {
			colWidth[i] = max(colWidth[i], cellWidth(row[i], opts.TruncateCells))
		}
	}

	shown, cropped := fitColumns(colWidth, opts.Width)

	widths := append([]int(nil), colWidth[:shown]...)
	if cropped {
		widths = append(widths, cmdio.Width(ellipsis))
	}

	header := make([]string, 0, len(widths))
	for i := range shown {
		header = append(header, renderCell(columns[i], colWidth[i], opts.TruncateCells))
	}
	if cropped {
		header = append(header, cmdio.Faint(ctx, ellipsis))
	}
	emitRow(out, header, widths)

	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat(separatorChar, min(w, maxColumnWidth))
	}
	emitRow(out, sep, widths)

	for _, row := range rows {
		cells := make([]string, 0, len(widths))
		for i := range shown {
			cell := ""
			if i < len(row) {
				cell = renderCell(row[i], colWidth[i], opts.TruncateCells)
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

// cellWidth returns the column-width contribution of s: capped at maxColumnWidth
// when truncating, otherwise its full display width.
func cellWidth(s string, truncate bool) int {
	if truncate {
		return min(cmdio.Width(s), maxColumnWidth)
	}
	return cmdio.Width(s)
}

// renderCell returns s truncated to width when truncating, otherwise s unchanged.
func renderCell(s string, width int, truncate bool) string {
	if truncate {
		return truncateToWidth(s, width)
	}
	return s
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
