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

	"github.com/databricks/cli/libs/flags"
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
	for i := 0; i < len(lines); i++ {
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

type writeFlusher interface {
	io.Writer
	Flush() error
}

type jsonRenderer interface {
	renderJson(context.Context, writeFlusher) error
}

type textRenderer interface {
	renderText(context.Context, writeFlusher) error
}

type templateRenderer interface {
	renderTemplate(context.Context, *template.Template, writeFlusher) error
}

type readerRenderer struct {
	reader io.Reader
}

func (r readerRenderer) renderText(_ context.Context, w writeFlusher) error {
	_, err := io.Copy(w, r.reader)
	if err != nil {
		return err
	}
	return w.Flush()
}

type iteratorRenderer struct {
	t          reflectIterator
	bufferSize int
}

func (ir iteratorRenderer) getBufferSize() int {
	if ir.bufferSize == 0 {
		return 100
	}
	return ir.bufferSize
}

func (ir iteratorRenderer) renderJson(ctx context.Context, w writeFlusher) error {
	// Iterators are always rendered as a list of resources in JSON.
	_, err := w.Write([]byte("[\n  "))
	if err != nil {
		return err
	}
	for i := 0; ir.t.HasNext(ctx); i++ {
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
		_, err = w.Write([]byte(",\n  "))
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
	_, err = w.Write([]byte("]\n"))
	if err != nil {
		return err
	}
	return w.Flush()
}

func (ir iteratorRenderer) renderTemplate(ctx context.Context, t *template.Template, w writeFlusher) error {
	for i := 0; ir.t.HasNext(ctx); i++ {
		n, err := ir.t.Next(ctx)
		if err != nil {
			return err
		}
		err = t.Execute(w, []any{n})
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
	return w.Flush()
}

type defaultRenderer struct {
	it any
}

func (d defaultRenderer) renderJson(_ context.Context, w writeFlusher) error {
	pretty, err := fancyJSON(d.it)
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
	return nil
}

func (d defaultRenderer) renderTemplate(_ context.Context, t *template.Template, w writeFlusher) error {
	return t.Execute(w, d.it)
}

// Returns something implementing one of the following interfaces:
//   - jsonRenderer
//   - textRenderer
//   - templateRenderer
func newRenderer(it any) any {
	if r, ok := any(it).(io.Reader); ok {
		return readerRenderer{reader: r}
	}
	if iterator, ok := newReflectIterator(it); ok {
		return iteratorRenderer{t: iterator}
	}
	return defaultRenderer{it: it}
}

type bufferedFlusher struct {
	w io.Writer
	b bytes.Buffer
}

func (b bufferedFlusher) Write(bs []byte) (int, error) {
	return b.w.Write(bs)
}

func (b bufferedFlusher) Flush() error {
	_, err := b.Write(b.b.Bytes())
	if err != nil {
		return err
	}
	b.b.Reset()
	return nil
}

func newBufferedFlusher(w io.Writer) writeFlusher {
	return bufferedFlusher{w: w}
}

func renderWithTemplate(r any, ctx context.Context, template string) error {
	// TODO: add terminal width & white/dark theme detection
	c := fromContext(ctx)
	switch c.outputFormat {
	case flags.OutputJSON:
		if jr, ok := r.(jsonRenderer); ok {
			return jr.renderJson(ctx, newBufferedFlusher(c.out))
		}
		return errors.New("json output not supported")
	case flags.OutputText:
		if tr, ok := r.(templateRenderer); ok && template != "" {
			return renderUsingTemplate(ctx, tr, c.out, template)
		}
		if tr, ok := r.(textRenderer); ok {
			return tr.renderText(ctx, newBufferedFlusher(c.out))
		}
		if jr, ok := r.(jsonRenderer); ok {
			return jr.renderJson(ctx, newBufferedFlusher(c.out))
		}
		return errors.New("no renderer defined")
	default:
		return fmt.Errorf("invalid output format: %s", c.outputFormat)
	}
}

func Render(ctx context.Context, v any) error {
	c := fromContext(ctx)
	return RenderWithTemplate(ctx, v, c.template)
}

func RenderWithTemplate(ctx context.Context, v any, template string) error {
	return renderWithTemplate(newRenderer(v), ctx, template)
}

func RenderJson(ctx context.Context, v any) error {
	c := fromContext(ctx)
	if jr, ok := newRenderer(v).(jsonRenderer); ok {
		jr.renderJson(ctx, newBufferedFlusher(c.out))
	}
	return errors.New("json output not supported")
}

func RenderReader(ctx context.Context, r io.Reader) error {
	c := fromContext(ctx)
	if jr, ok := newRenderer(r).(textRenderer); ok {
		jr.renderText(ctx, newBufferedFlusher(c.out))
	}
	return errors.New("rendering io.Reader not supported")
}

func renderUsingTemplate(ctx context.Context, r templateRenderer, w io.Writer, tmpl string) error {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	t, err := template.New("command").Funcs(template.FuncMap{
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
	}).Parse(tmpl)
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
