package cmdio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/listing"
)

// Heredoc is the equivalent of compute.TrimLeadingWhitespace
// (command-execution API helper from SDK), except it's more
// friendly to non-printable characters.
func Heredoc(tmpl string) string {
	lines := strings.Split(tmpl, "\n")
	leadingWhitespace := 1<<31 - 1
	for _, line := range lines {
		for pos, char := range line {
			if char == ' ' || char == '\t' {
				continue
			}
			// first non-whitespace character
			if pos < leadingWhitespace {
				leadingWhitespace = pos
			}
			// is not needed further
			break
		}
	}
	var sb strings.Builder
	for i := range lines {
		if lines[i] == "" || strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if len(lines[i]) < leadingWhitespace {
			sb.WriteString(lines[i])
		} else {
			sb.WriteString(lines[i][leadingWhitespace:])
		}
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}

// writeFlusher represents a buffered writer that can be flushed. This is useful when
// buffering writing a large number of resources (such as during a list API).
type writeFlusher interface {
	io.Writer
	Flush() error
}

type jsonRenderer interface {
	// Render an object as JSON to the provided writeFlusher.
	renderJson(context.Context, writeFlusher) error
}

type textRenderer interface {
	// Render an object as text to the provided writeFlusher.
	renderText(context.Context, io.Writer) error
}

type templateRenderer interface {
	// Render an object using the provided template and write to the provided tabwriter.Writer.
	renderTemplate(context.Context, *template.Template, *tabwriter.Writer) error
}

type readerRenderer struct {
	reader io.Reader
}

func (r readerRenderer) renderText(_ context.Context, w io.Writer) error {
	_, err := io.Copy(w, r.reader)
	return err
}

type iteratorRenderer[T any] struct {
	t              listing.Iterator[T]
	bufferSize     int
	inputOnlyPaths []string
}

func (ir iteratorRenderer[T]) getBufferSize() int {
	if ir.bufferSize == 0 {
		return 20
	}
	return ir.bufferSize
}

func (ir iteratorRenderer[T]) renderJson(ctx context.Context, w writeFlusher) error {
	// Iterators are always rendered as a list of resources in JSON.
	_, err := w.Write([]byte("[\n  "))
	if err != nil {
		return err
	}
	limit := limitFromContext(ctx)
	for i := 0; ir.t.HasNext(ctx); i++ {
		if limit > 0 && i >= limit {
			break
		}
		if i != 0 {
			_, err = w.Write([]byte(",\n  "))
			if err != nil {
				return err
			}
		}
		n, err := ir.t.Next(ctx)
		if err != nil {
			return err
		}
		masked, err := applyInputOnlyMask(n, ir.inputOnlyPaths)
		if err != nil {
			return err
		}
		res, err := json.MarshalIndent(masked, "  ", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(res)
		if err != nil {
			return err
		}
		if (i+1)%ir.getBufferSize() == 0 {
			err = w.Flush()
			if err != nil {
				return err
			}
		}
	}
	_, err = w.Write([]byte("\n]\n"))
	if err != nil {
		return err
	}
	return w.Flush()
}

func (ir iteratorRenderer[T]) renderTemplate(ctx context.Context, t *template.Template, w *tabwriter.Writer) error {
	buf := make([]any, 0, ir.getBufferSize())
	limit := limitFromContext(ctx)
	for i := 0; ir.t.HasNext(ctx); i++ {
		if limit > 0 && i >= limit {
			break
		}
		n, err := ir.t.Next(ctx)
		if err != nil {
			return err
		}
		buf = append(buf, n)
		if len(buf) == cap(buf) {
			err = t.Execute(w, buf)
			if err != nil {
				return err
			}
			err = w.Flush()
			if err != nil {
				return err
			}
			buf = buf[:0]
		}
	}
	if len(buf) > 0 {
		err := t.Execute(w, buf)
		if err != nil {
			return err
		}
	}
	return w.Flush()
}

type defaultRenderer struct {
	t              any
	inputOnlyPaths []string
}

func (d defaultRenderer) renderJson(ctx context.Context, w writeFlusher) error {
	c := fromContext(ctx)
	v, err := applyInputOnlyMask(d.t, d.inputOnlyPaths)
	if err != nil {
		return err
	}
	pretty, err := marshalJSON(v, c.capabilities.SupportsStdoutColor())
	if err != nil {
		return err
	}
	_, err = w.Write(pretty)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	if err != nil {
		return err
	}
	return w.Flush()
}

func (d defaultRenderer) renderTemplate(_ context.Context, t *template.Template, w *tabwriter.Writer) error {
	return t.Execute(w, d.t)
}

// Returns something implementing one of the following interfaces:
//   - jsonRenderer
//   - textRenderer
//   - templateRenderer
//
// inputOnlyPaths, when non-empty, lists dotted JSON paths that should be
// stripped from the rendered value before it is written to stdout. The
// paths are consulted only by the JSON render path; text/template
// rendering operates on the raw value.
func newRenderer(t any, inputOnlyPaths []string) any {
	if r, ok := t.(io.Reader); ok {
		return readerRenderer{reader: r}
	}
	return defaultRenderer{t: t, inputOnlyPaths: inputOnlyPaths}
}

func newIteratorRenderer[T any](i listing.Iterator[T], inputOnlyPaths []string) iteratorRenderer[T] {
	return iteratorRenderer[T]{t: i, inputOnlyPaths: inputOnlyPaths}
}

type bufferedFlusher struct {
	w io.Writer
	b *bytes.Buffer
}

func (b bufferedFlusher) Write(bs []byte) (int, error) {
	return b.b.Write(bs)
}

func (b bufferedFlusher) Flush() error {
	_, err := b.w.Write(b.b.Bytes())
	if err != nil {
		return err
	}
	b.b.Reset()
	return nil
}

func newBufferedFlusher(w io.Writer) writeFlusher {
	return bufferedFlusher{
		w: w,
		b: &bytes.Buffer{},
	}
}

func renderWithTemplate(ctx context.Context, r any, outputFormat flags.Output, w io.Writer, headerTemplate, template string) error {
	// TODO: add terminal width & white/dark theme detection
	switch outputFormat {
	case flags.OutputJSON:
		if jr, ok := r.(jsonRenderer); ok {
			return jr.renderJson(ctx, newBufferedFlusher(w))
		}
		return errors.New("json output not supported")
	case flags.OutputText:
		if tr, ok := r.(templateRenderer); ok && template != "" {
			return renderUsingTemplate(ctx, tr, w, headerTemplate, template)
		}
		if tr, ok := r.(textRenderer); ok {
			return tr.renderText(ctx, w)
		}
		if jr, ok := r.(jsonRenderer); ok {
			return jr.renderJson(ctx, newBufferedFlusher(w))
		}
		return errors.New("no renderer defined")
	default:
		return fmt.Errorf("invalid output format: %s", outputFormat)
	}
}

type listingInterface interface {
	HasNext(context.Context) bool
}

func Render(ctx context.Context, v any) error {
	return RenderFiltered(ctx, v, nil)
}

// RenderFiltered behaves like Render but strips the given dotted JSON
// paths from the value before it is marshaled. Used by generated CLI
// commands for response types containing INPUT_ONLY fields (which the
// SDK transport struct carries unconditionally) so those fields don't
// leak into user-facing JSON output.
func RenderFiltered(ctx context.Context, v any, inputOnlyPaths []string) error {
	c := fromContext(ctx)
	if _, ok := v.(listingInterface); ok {
		panic("use RenderIterator instead")
	}
	return renderWithTemplate(ctx, newRenderer(v, inputOnlyPaths), c.outputFormat, c.out, c.headerTemplate, c.template)
}

// RenderIterator renders the items produced by i. When the terminal is
// fully interactive (stdin + stdout + stderr all TTYs) and the command
// has a row template, we page through the existing template + tabwriter
// pipeline (same colors, same alignment as the non-paged path; widths are
// locked from the first batch so columns stay aligned across pages).
// Piped output and JSON output keep the existing non-paged behavior.
func RenderIterator[T any](ctx context.Context, i listing.Iterator[T]) error {
	return RenderIteratorFiltered(ctx, i, nil)
}

// RenderIteratorFiltered behaves like RenderIterator but strips the given
// dotted JSON paths from each element before it is marshaled. See
// RenderFiltered for the motivation.
func RenderIteratorFiltered[T any](ctx context.Context, i listing.Iterator[T], inputOnlyPaths []string) error {
	c := fromContext(ctx)
	if c.capabilities.SupportsPager() && c.outputFormat == flags.OutputText && c.template != "" {
		return renderIteratorPagedTemplate(ctx, i, c.in, c.out, c.headerTemplate, c.template)
	}
	return renderWithTemplate(ctx, newIteratorRenderer(i, inputOnlyPaths), c.outputFormat, c.out, c.headerTemplate, c.template)
}

func RenderWithTemplate(ctx context.Context, v any, headerTemplate, template string) error {
	c := fromContext(ctx)
	if _, ok := v.(listingInterface); ok {
		panic("listings must use RenderIterator, not RenderWithTemplate")
	}
	return renderWithTemplate(ctx, newRenderer(v, nil), c.outputFormat, c.out, headerTemplate, template)
}

// staticTemplateFuncs are the ctx-independent helpers shared across every
// renderFuncMap call.
var staticTemplateFuncs = template.FuncMap{
	"replace": strings.ReplaceAll,
	"join":    strings.Join,
	"sub":     func(a, b int) int { return a - b },
	"pretty_date": func(t time.Time) string {
		return t.Format("2006-01-02T15:04:05Z")
	},
	"pretty_UTC_date_from_millis": func(millis int64) string {
		t := time.UnixMilli(millis).UTC()
		return t.Format("2006-01-02T15:04:05Z")
	},
	"b64_encode": func(in string) (string, error) {
		var out bytes.Buffer
		enc := base64.NewEncoder(base64.StdEncoding, &out)
		_, err := enc.Write([]byte(in))
		if err != nil {
			return "", err
		}
		err = enc.Close()
		if err != nil {
			return "", err
		}
		return out.String(), nil
	},
	"b64_decode": func(in string) (string, error) {
		dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(in))
		out, err := io.ReadAll(dec)
		if err != nil {
			return "", err
		}
		return string(out), nil
	},
}

// renderFuncMap returns the template helpers used by cmdio's rendering
// pipeline. Color helpers and the JSON pretty-printer depend on ctx; the
// rest are taken from staticTemplateFuncs.
func renderFuncMap(ctx context.Context) template.FuncMap {
	fm := RenderFuncMap(ctx)
	fm["header"] = fm["blue"]
	fm["bool"] = func(v bool) string {
		if v {
			return Green(ctx, "YES")
		}
		return Red(ctx, "NO")
	}
	fm["pretty_json"] = func(in string) (string, error) {
		var tmp any
		err := json.Unmarshal([]byte(in), &tmp)
		if err != nil {
			return "", err
		}
		b, err := marshalJSON(tmp, colorEnabled(ctx))
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	maps.Copy(fm, staticTemplateFuncs)
	return fm
}

func renderUsingTemplate(ctx context.Context, r templateRenderer, w io.Writer, headerTmpl, tmpl string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	base := template.New("command").Funcs(renderFuncMap(ctx))
	if headerTmpl != "" {
		headerT, err := base.Parse(headerTmpl)
		if err != nil {
			return err
		}
		err = headerT.Execute(tw, nil)
		if err != nil {
			return err
		}
		if _, err := tw.Write([]byte("\n")); err != nil {
			return err
		}
		// Do not flush here. Instead, allow the first 100 resources to determine the initial spacing of the header columns.
	}
	t, err := base.Parse(tmpl)
	if err != nil {
		return err
	}
	err = r.renderTemplate(ctx, t, tw)
	if err != nil {
		return err
	}
	return tw.Flush()
}

const errorTemplate = `{{ "Error" | red }}: {{ .Summary }}
{{- if and .Paths (ne (index .Paths 0).String "") }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

const warningTemplate = `{{ "Warning" | yellow }}: {{ .Summary }}
{{- if and .Paths (ne (index .Paths 0).String "") }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

const recommendationTemplate = `{{ "Recommendation" | blue }}: {{ .Summary }}
{{- if and .Paths (ne (index .Paths 0).String "") }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

func RenderDiagnostics(ctx context.Context, diags diag.Diagnostics) error {
	c := fromContext(ctx)
	return renderDiagnostics(ctx, c.err, diags)
}

func renderDiagnostics(ctx context.Context, out io.Writer, diags diag.Diagnostics) error {
	fm := renderFuncMap(ctx)
	errorT := template.Must(template.New("error").Funcs(fm).Parse(errorTemplate))
	warningT := template.Must(template.New("warning").Funcs(fm).Parse(warningTemplate))
	recommendationT := template.Must(template.New("recommendation").Funcs(fm).Parse(recommendationTemplate))

	// Print errors and warnings.
	for _, d := range diags {
		var t *template.Template
		switch d.Severity {
		case diag.Error:
			t = errorT
		case diag.Warning:
			t = warningT
		case diag.Recommendation:
			t = recommendationT
		}

		// Render the diagnostic with the appropriate template.
		err := t.Execute(out, d)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
	}

	return nil
}
