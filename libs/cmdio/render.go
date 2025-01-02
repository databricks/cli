package cmdio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/fatih/color"
	"github.com/nwidger/jsoncolor"
)

// Heredoc is the equivalent of compute.TrimLeadingWhitespace
// (command-execution API helper from SDK), except it's more
// friendly to non-printable characters.
func Heredoc(tmpl string) (trimmed string) {
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
	for i := range lines {
		if lines[i] == "" || strings.TrimSpace(lines[i]) == "" {
			continue
		}
		if len(lines[i]) < leadingWhitespace {
			trimmed += lines[i] + "\n" // or not..
		} else {
			trimmed += lines[i][leadingWhitespace:] + "\n"
		}
	}
	return strings.TrimSpace(trimmed)
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
	t          listing.Iterator[T]
	bufferSize int
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
	for i := 0; ir.t.HasNext(ctx); i++ {
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
		res, err := json.MarshalIndent(n, "  ", "  ")
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
	for i := 0; ir.t.HasNext(ctx); i++ {
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
	t any
}

func (d defaultRenderer) renderJson(_ context.Context, w writeFlusher) error {
	pretty, err := fancyJSON(d.t)
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
func newRenderer(t any) any {
	if r, ok := t.(io.Reader); ok {
		return readerRenderer{reader: r}
	}
	return defaultRenderer{t: t}
}

func newIteratorRenderer[T any](i listing.Iterator[T]) iteratorRenderer[T] {
	return iteratorRenderer[T]{t: i}
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

func renderWithTemplate(r any, ctx context.Context, outputFormat flags.Output, w io.Writer, headerTemplate, template string) error {
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
	c := fromContext(ctx)
	if _, ok := v.(listingInterface); ok {
		panic("use RenderIterator instead")
	}
	return renderWithTemplate(newRenderer(v), ctx, c.outputFormat, c.out, c.headerTemplate, c.template)
}

func RenderIterator[T any](ctx context.Context, i listing.Iterator[T]) error {
	c := fromContext(ctx)
	return renderWithTemplate(newIteratorRenderer(i), ctx, c.outputFormat, c.out, c.headerTemplate, c.template)
}

func RenderWithTemplate(ctx context.Context, v any, headerTemplate, template string) error {
	c := fromContext(ctx)
	if _, ok := v.(listingInterface); ok {
		panic("use RenderIteratorWithTemplate instead")
	}
	return renderWithTemplate(newRenderer(v), ctx, c.outputFormat, c.out, headerTemplate, template)
}

func RenderIteratorWithTemplate[T any](ctx context.Context, i listing.Iterator[T], headerTemplate, template string) error {
	c := fromContext(ctx)
	return renderWithTemplate(newIteratorRenderer(i), ctx, c.outputFormat, c.out, headerTemplate, template)
}

func RenderIteratorJson[T any](ctx context.Context, i listing.Iterator[T]) error {
	c := fromContext(ctx)
	return renderWithTemplate(newIteratorRenderer(i), ctx, c.outputFormat, c.out, c.headerTemplate, c.template)
}

var renderFuncMap = template.FuncMap{
	// we render colored output if stdout is TTY, otherwise we render text.
	// in the future we'll check if we can explicitly check for stderr being
	// a TTY
	"header":  color.BlueString,
	"red":     color.RedString,
	"green":   color.GreenString,
	"blue":    color.BlueString,
	"yellow":  color.YellowString,
	"magenta": color.MagentaString,
	"cyan":    color.CyanString,
	"bold": func(format string, a ...any) string {
		return color.New(color.Bold).Sprintf(format, a...)
	},
	"italic": func(format string, a ...any) string {
		return color.New(color.Italic).Sprintf(format, a...)
	},
	"replace": strings.ReplaceAll,
	"join":    strings.Join,
	"bool": func(v bool) string {
		if v {
			return color.GreenString("YES")
		}
		return color.RedString("NO")
	},
	"pretty_json": func(in string) (string, error) {
		var tmp any
		err := json.Unmarshal([]byte(in), &tmp)
		if err != nil {
			return "", err
		}
		b, err := fancyJSON(tmp)
		if err != nil {
			return "", err
		}
		return string(b), nil
	},
	"pretty_date": func(t time.Time) string {
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

func renderUsingTemplate(ctx context.Context, r templateRenderer, w io.Writer, headerTmpl, tmpl string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	base := template.New("command").Funcs(renderFuncMap)
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

func fancyJSON(v any) ([]byte, error) {
	// create custom formatter
	f := jsoncolor.NewFormatter()

	// set custom colors
	f.StringColor = color.New(color.FgGreen)
	f.TrueColor = color.New(color.FgGreen, color.Bold)
	f.FalseColor = color.New(color.FgRed)
	f.NumberColor = color.New(color.FgCyan)
	f.NullColor = color.New(color.FgMagenta)
	f.ObjectColor = color.New(color.Reset)
	f.CommaColor = color.New(color.Reset)
	f.ColonColor = color.New(color.Reset)

	return jsoncolor.MarshalIndentWithFormatter(v, "", "  ", f)
}

const errorTemplate = `{{ "Error" | red }}: {{ .Summary }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

const warningTemplate = `{{ "Warning" | yellow }}: {{ .Summary }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

const recommendationTemplate = `{{ "Recommendation" | blue }}: {{ .Summary }}
{{- range $index, $element := .Paths }}
  {{ if eq $index 0 }}at {{else}}   {{ end}}{{ $element.String | green }}
{{- end }}
{{- range $index, $element := .Locations }}
  {{ if eq $index 0 }}in {{else}}   {{ end}}{{ $element.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}

`

func RenderDiagnosticsToErrorOut(ctx context.Context, diags diag.Diagnostics) error {
	c := fromContext(ctx)
	return RenderDiagnostics(c.err, diags)
}

func RenderDiagnostics(out io.Writer, diags diag.Diagnostics) error {
	errorT := template.Must(template.New("error").Funcs(renderFuncMap).Parse(errorTemplate))
	warningT := template.Must(template.New("warning").Funcs(renderFuncMap).Parse(warningTemplate))
	recommendationT := template.Must(template.New("recommendation").Funcs(renderFuncMap).Parse(recommendationTemplate))

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
