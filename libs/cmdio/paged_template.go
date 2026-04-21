package cmdio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"text/template"
	"unicode/utf8"

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
	keys, restore, err := startRawStdinKeyReader(ctx)
	if err != nil {
		return err
	}
	defer restore()
	return renderIteratorPagedTemplateCore(
		ctx,
		iter,
		crlfWriter{w: out},
		crlfWriter{w: os.Stderr},
		keys,
		headerTemplate,
		tmpl,
		pagerPageSize,
	)
}

// templatePager renders accumulated rows to out, locking column widths
// from the first page so layout stays stable across batches. We do not
// use text/tabwriter because it recomputes widths on every Flush.
type templatePager struct {
	out        io.Writer
	headerT    *template.Template
	rowT       *template.Template
	headerStr  string
	widths     []int
	headerDone bool
}

func (p *templatePager) flush(buf []any) error {
	if p.headerDone && len(buf) == 0 {
		return nil
	}
	var rendered bytes.Buffer
	if !p.headerDone && p.headerStr != "" {
		if err := p.headerT.Execute(&rendered, nil); err != nil {
			return err
		}
		rendered.WriteByte('\n')
	}
	if len(buf) > 0 {
		if err := p.rowT.Execute(&rendered, buf); err != nil {
			return err
		}
	}
	p.headerDone = true

	text := strings.TrimRight(rendered.String(), "\n")
	if text == "" {
		return nil
	}
	rows := strings.Split(text, "\n")
	if p.widths == nil {
		p.widths = computeWidths(rows)
	}
	for _, row := range rows {
		if _, err := io.WriteString(p.out, padRow(strings.Split(row, "\t"), p.widths)+"\n"); err != nil {
			return err
		}
	}
	return nil
}

func renderIteratorPagedTemplateCore[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	out io.Writer,
	prompts io.Writer,
	keys <-chan byte,
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
		out:       out,
		headerT:   headerT,
		rowT:      rowT,
		headerStr: headerTemplate,
	}

	limit := limitFromContext(ctx)
	drainAll := false
	buf := make([]any, 0, pageSize)
	total := 0

	for iter.HasNext(ctx) {
		if limit > 0 && total >= limit {
			break
		}
		n, err := iter.Next(ctx)
		if err != nil {
			return err
		}
		buf = append(buf, n)
		total++

		if len(buf) < pageSize {
			continue
		}
		if err := pager.flush(buf); err != nil {
			return err
		}
		buf = buf[:0]
		if drainAll {
			if pagerShouldQuit(keys) {
				return nil
			}
			continue
		}
		fmt.Fprint(prompts, pagerPromptText)
		k, ok := pagerNextKey(ctx, keys)
		fmt.Fprint(prompts, pagerClearLine)
		if !ok {
			return nil
		}
		switch k {
		case ' ':
		case '\r', '\n':
			drainAll = true
		case 'q', 'Q', pagerKeyEscape, pagerKeyCtrlC:
			return nil
		}
	}
	return pager.flush(buf)
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
