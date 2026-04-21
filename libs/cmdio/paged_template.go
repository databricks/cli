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

// pagerColumnSeparator is the inter-column spacing used when emitting paged
// template output. Matches text/tabwriter.NewWriter(..., 2, ' ', ...) in
// the non-paged path, so single-batch output is visually indistinguishable
// from what renderUsingTemplate produces.
const pagerColumnSeparator = "  "

// ansiCSIPattern matches ANSI SGR escape sequences so we can compute the
// on-screen width of cells that contain colored output. We do not attempt
// to cover every ANSI escape — just the SGR color/style ones (CSI ... m)
// emitted by github.com/fatih/color, which is all our templates use today.
var ansiCSIPattern = regexp.MustCompile("\x1b\\[[0-9;]*m")

// renderIteratorPagedTemplate streams an iterator through the existing
// template-based renderer one page at a time, prompting the user between
// pages on stderr. The rendering is intentionally identical to the non-
// paged path — same templates, same colors — only the flush cadence,
// the user-facing prompt, and column-width stability across batches
// differ.
//
// SPACE advances by one page; ENTER drains the remaining iterator (still
// interruptible by q/esc/Ctrl+C); q/esc/Ctrl+C stop immediately.
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

// renderIteratorPagedTemplateCore is the testable core of
// renderIteratorPagedTemplate: it assumes the caller has already set up
// raw stdin (or any key source) and delivered a channel of keystrokes.
// It never touches os.Stdin directly.
//
// Unlike renderUsingTemplate (the non-paged path) we do not rely on
// text/tabwriter to align columns. Tabwriter computes column widths
// per-flush and resets on Flush(), which produces a jarring
// width-shift when a short final batch follows a wider first batch.
// Here we render each page's template output into an intermediate
// buffer, split it into cells by tab, lock visual column widths from
// the first page, and pad every subsequent page to the same widths.
// The output is visually indistinguishable from tabwriter for single-
// batch lists and stays aligned across batches for longer ones.
func renderIteratorPagedTemplateCore[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	out io.Writer,
	prompts io.Writer,
	keys <-chan byte,
	headerTemplate, tmpl string,
	pageSize int,
) error {
	// Use two independent templates so parsing the row template doesn't
	// overwrite the header template's parsed body (they would if they
	// shared the same *template.Template instance — Parse replaces the
	// body in place and returns the receiver).
	headerT, err := template.New("header").Funcs(renderFuncMap).Parse(headerTemplate)
	if err != nil {
		return err
	}
	rowT, err := template.New("row").Funcs(renderFuncMap).Parse(tmpl)
	if err != nil {
		return err
	}

	limit := limitFromContext(ctx)
	drainAll := false
	buf := make([]any, 0, pageSize)
	total := 0

	var lockedWidths []int
	firstBatchDone := false

	flushPage := func() error {
		// Nothing to emit after the first batch is out the door and the
		// buffer is empty — we've already written the header.
		if firstBatchDone && len(buf) == 0 {
			return nil
		}

		var rendered bytes.Buffer
		if !firstBatchDone && headerTemplate != "" {
			if err := headerT.Execute(&rendered, nil); err != nil {
				return err
			}
			rendered.WriteByte('\n')
		}
		if len(buf) > 0 {
			if err := rowT.Execute(&rendered, buf); err != nil {
				return err
			}
			buf = buf[:0]
		}
		firstBatchDone = true

		text := strings.TrimRight(rendered.String(), "\n")
		if text == "" {
			return nil
		}
		rows := strings.Split(text, "\n")

		// Lock column widths from the first batch (header + first page).
		// Every subsequent batch pads to these widths so columns do not
		// shift between pages.
		if lockedWidths == nil {
			for _, row := range rows {
				for i, cell := range strings.Split(row, "\t") {
					if i >= len(lockedWidths) {
						lockedWidths = append(lockedWidths, 0)
					}
					if w := visualWidth(cell); w > lockedWidths[i] {
						lockedWidths[i] = w
					}
				}
			}
		}

		for _, row := range rows {
			line := padRow(strings.Split(row, "\t"), lockedWidths)
			if _, err := io.WriteString(out, line+"\n"); err != nil {
				return err
			}
		}
		return nil
	}

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
		if err := flushPage(); err != nil {
			return err
		}
		if drainAll {
			if pagerShouldQuit(keys) {
				return nil
			}
			continue
		}
		// Show the prompt and wait for a key.
		fmt.Fprint(prompts, pagerPromptText)
		k, ok := pagerNextKey(ctx, keys)
		fmt.Fprint(prompts, pagerClearLine)
		if !ok {
			return nil
		}
		switch k {
		case ' ':
			// Continue to the next page.
		case '\r', '\n':
			drainAll = true
		case 'q', 'Q', pagerKeyEscape, pagerKeyCtrlC:
			return nil
		}
	}
	return flushPage()
}

// visualWidth returns the number of runes a string would occupy on-screen
// if ANSI SGR escape sequences are respected as zero-width (which the
// terminal does). This matches what the user sees and lets us pad colored
// cells to consistent visual widths.
func visualWidth(s string) int {
	return utf8.RuneCountInString(ansiCSIPattern.ReplaceAllString(s, ""))
}

// padRow joins the given cells with pagerColumnSeparator, padding every
// cell except the last to widths[i] visual runes. Cells wider than
// widths[i] are emitted as-is — the extra content pushes subsequent
// columns right for that row only, which is the same behavior tabwriter
// would give and the same behavior the non-paged renderer has today.
func padRow(cells []string, widths []int) string {
	var b strings.Builder
	for i, cell := range cells {
		if i > 0 {
			b.WriteString(pagerColumnSeparator)
		}
		b.WriteString(cell)
		if i < len(cells)-1 && i < len(widths) {
			pad := widths[i] - visualWidth(cell)
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}
		}
	}
	return b.String()
}
