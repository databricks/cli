package cmdio

import (
	"bytes"
	"context"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/databricks/databricks-sdk-go/listing"
)

// ansiCSIPattern matches ANSI SGR escape sequences so colored cells
// aren't counted toward column widths. github.com/fatih/color emits CSI
// ... m, which is all our templates use.
var ansiCSIPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// renderIteratorPagedTemplate pages an iterator through the template
// renderer, prompting between batches. SPACE advances one page, ENTER
// drains the rest, q/esc/Ctrl+C quit.
func renderIteratorPagedTemplate[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	out io.Writer,
	headerTemplate, tmpl string,
) error {
	return renderIteratorPagedTemplateCore(ctx, iter, os.Stdin, out, headerTemplate, tmpl, pagerPageSize)
}

// templatePager renders accumulated rows, locking column widths from the
// first page so layout stays stable across batches. We do not use
// text/tabwriter because it recomputes widths on every Flush.
type templatePager struct {
	headerT    *template.Template
	rowT       *template.Template
	headerStr  string
	widths     []int
	headerDone bool
}

// flushLines renders the header (on the first call) plus any buffered
// rows, then pads each cell to the widths recorded on the first page so
// columns line up across batches.
func (p *templatePager) flushLines(buf []any) ([]string, error) {
	if p.headerDone && len(buf) == 0 {
		return nil, nil
	}
	var rendered bytes.Buffer
	if !p.headerDone && p.headerStr != "" {
		if err := p.headerT.Execute(&rendered, nil); err != nil {
			return nil, err
		}
		rendered.WriteByte('\n')
	}
	if len(buf) > 0 {
		if err := p.rowT.Execute(&rendered, buf); err != nil {
			return nil, err
		}
	}
	p.headerDone = true

	text := strings.TrimRight(rendered.String(), "\n")
	if text == "" {
		return nil, nil
	}
	rows := strings.Split(text, "\n")
	if p.widths == nil {
		p.widths = computeWidths(rows)
	}
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = padRow(strings.Split(row, "\t"), p.widths)
	}
	return lines, nil
}

func renderIteratorPagedTemplateCore[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	in io.Reader,
	out io.Writer,
	headerTemplate, tmpl string,
	pageSize int,
) error {
	// Header and row templates must be separate *template.Template
	// instances: Parse replaces the receiver's body in place, so sharing
	// one makes the second Parse stomp the first.
	headerT, err := template.New("header").Funcs(renderFuncMap).Parse(headerTemplate)
	if err != nil {
		return err
	}
	rowT, err := template.New("row").Funcs(renderFuncMap).Parse(tmpl)
	if err != nil {
		return err
	}
	pager := &templatePager{
		headerT:   headerT,
		rowT:      rowT,
		headerStr: headerTemplate,
	}
	m := &pagerModel[T]{
		ctx:      ctx,
		iter:     iter,
		pager:    pager,
		pageSize: pageSize,
		limit:    limitFromContext(ctx),
	}
	p := tea.NewProgram(
		m,
		tea.WithInput(in),
		tea.WithOutput(out),
		// Match spinner: let SIGINT reach the process rather than the TUI
		// so Ctrl+C also interrupts a stalled iterator fetch.
		tea.WithoutSignalHandler(),
	)
	if _, err := p.Run(); err != nil {
		return err
	}
	return m.err
}

// visualWidth counts runes ignoring ANSI SGR escape sequences.
func visualWidth(s string) int {
	return utf8.RuneCountInString(ansiCSIPattern.ReplaceAllString(s, ""))
}

func computeWidths(rows []string) []int {
	var widths []int
	for _, row := range rows {
		for i, cell := range strings.Split(row, "\t") {
			if i >= len(widths) {
				widths = append(widths, 0)
			}
			if w := visualWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}
	return widths
}

// padRow joins cells with two-space separators matching tabwriter's
// minpad, padding every cell except the last to widths[i] visual runes.
func padRow(cells []string, widths []int) string {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString("  ")
		}
		b.WriteString(cell)
		if i < len(cells)-1 && i < len(widths) {
			if pad := widths[i] - visualWidth(cell); pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	return b.String()
}
