package pipelines

import (
	"bytes"
	"context"
	"encoding/json"
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

type tableRow struct {
	Cells []string
}

type tableData struct {
	Rows   []tableRow
	Footer []string
}

// Tab-separated cells let cmdio's tabwriter align columns; the footer follows a blank line.
const rowTemplate = "{{range .Rows}}{{range $i, $c := .Cells}}{{if $i}}\t{{end}}{{$c}}{{end}}\n{{end}}\n" +
	"{{range .Footer}}{{.}}\n{{end}}"

// marshals a result row as a JSON object with keys in column order.
type orderedRow struct {
	columns []string
	values  []string
}

func (r orderedRow) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, col := range r.columns {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(col)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		var val []byte
		if i < len(r.values) {
			val, err = json.Marshal(r.values[i])
		} else {
			val = []byte("null")
		}
		if err != nil {
			return nil, err
		}
		buf.Write(val)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// writes rows as an indented JSON array of column-ordered objects.
func renderJSON(w io.Writer, columns []string, rows [][]string) error {
	objects := make([]orderedRow, len(rows))
	for i, row := range rows {
		objects[i] = orderedRow{columns: columns, values: row}
	}
	output, err := json.MarshalIndent(objects, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	fmt.Fprintf(w, "%s\n", output)
	return nil
}

// renderTable writes a width-aware text table: cells cap at maxColumnWidth, and a positive width that the table exceeds drops trailing columns behind a "..." marker.
func renderTable(ctx context.Context, columns []string, rows [][]string, width int) error {
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

	header := make([]string, 0, shown+1)
	sep := make([]string, 0, shown+1)
	for i := range shown {
		header = append(header, truncateToWidth(columns[i], colWidth[i]))
		sep = append(sep, strings.Repeat(separatorChar, colWidth[i]))
	}
	if cropped {
		header = append(header, cmdio.Faint(ctx, ellipsis))
		sep = append(sep, strings.Repeat(separatorChar, cmdio.Width(ellipsis)))
	}

	data := tableData{Rows: make([]tableRow, 0, len(rows)+2)}
	data.Rows = append(data.Rows, tableRow{header}, tableRow{sep})
	for _, row := range rows {
		cells := make([]string, 0, shown+1)
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
		data.Rows = append(data.Rows, tableRow{cells})
	}

	if cropped {
		data.Footer = append(data.Footer, cmdio.Faint(ctx, fmt.Sprintf("(showing %d of %d columns)", shown, n)))
	}
	data.Footer = append(data.Footer, rowCountLabel(len(rows)))

	return cmdio.RenderWithTemplate(ctx, data, "", rowTemplate)
}

func rowCountLabel(n int) string {
	if n == 1 {
		return "1 row"
	}
	return fmt.Sprintf("%d rows", n)
}

// fitColumns returns how many leading columns fit within width (<= 0 means unlimited) and whether any were dropped; at least one column is always shown.
func fitColumns(colWidth []int, width int) (shown int, cropped bool) {
	n := len(colWidth)
	total := 0
	for i, w := range colWidth {
		if i > 0 {
			total += len(columnSep)
		}
		total += w
	}
	if width <= 0 || total <= width {
		return n, false
	}
	budget := width - len(columnSep) - len(ellipsis)
	used := 0
	for i := range colWidth {
		add := colWidth[i]
		if i > 0 {
			add += len(columnSep)
		}
		if used+add > budget {
			s := max(i, 1)
			return s, s < n
		}
		used += add
	}
	return n, false
}

// truncateToWidth shortens s to at most maxWidth display cells, marking truncation with "...".
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if cmdio.Width(s) <= maxWidth {
		return s
	}
	budget := maxWidth - len(ellipsis)
	if budget <= 0 {
		return cutToWidth(s, maxWidth)
	}
	return cutToWidth(s, budget) + ellipsis
}

// cutToWidth returns the longest prefix of s whose display width is <= w.
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

// detectWidth returns the terminal width, or 0 (meaning "no cap") when out is not an interactive terminal.
func detectWidth(out io.Writer) int {
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
