package cmdio

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
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

type renderer struct {
	renderTemplate func(context.Context, *template.Template, io.Writer) error
	renderText     func(context.Context, io.Writer) error
	renderJson     func(context.Context, io.Writer) error
}

type reflectIterator struct {
	hasNext reflect.Value
	next    reflect.Value
}

func newReflectIterator(v any) (reflectIterator, bool) {
	rv := reflect.ValueOf(v)
	rt := rv.Type()
	_, hasHasNext := rt.MethodByName("HasNext")
	_, hasNext := rt.MethodByName("Next")
	if hasNext && hasHasNext {
		return reflectIterator{
			hasNext: rv.MethodByName("HasNext"),
			next:    rv.MethodByName("Next"),
		}, true
	}
	return reflectIterator{}, false
}

func (r reflectIterator) HasNext(ctx context.Context) bool {
	res := r.hasNext.Call([]reflect.Value{reflect.ValueOf(ctx)})
	return res[0].Bool()
}

func (r reflectIterator) Next(ctx context.Context) (any, error) {
	res := r.next.Call([]reflect.Value{reflect.ValueOf(ctx)})
	item := res[0].Interface()
	if res[1].IsNil() {
		return item, nil
	}
	return item, res[1].Interface().(error)
}

func New(it any) *renderer {
	if r, ok := any(it).(io.Reader); ok {
		return &renderer{
			renderJson: func(_ context.Context, w io.Writer) error {
				return fmt.Errorf("json output not supported")
			},
			renderText: func(_ context.Context, w io.Writer) error {
				_, err := io.Copy(w, r)
				return err
			},
		}
	}

	if iterator, ok := newReflectIterator(it); ok {
		return &renderer{
			renderJson: func(ctx context.Context, w io.Writer) error {
				// Iterators are always rendered as a list of resources in JSON.
				_, err := w.Write([]byte("[\n"))
				if err != nil {
					return err
				}
				for iterator.HasNext(ctx) {
					n, err := iterator.Next(ctx)
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
				}
				_, err = w.Write([]byte("]\n"))
				if err != nil {
					return err
				}
				return nil
			},
			renderTemplate: func(ctx context.Context, t *template.Template, w io.Writer) error {
				for iterator.HasNext(ctx) {
					n, err := iterator.Next(ctx)
					if err != nil {
						return err
					}
					err = t.Execute(w, []any{n})
					if err != nil {
						return err
					}
				}
				return nil
			},
		}
	}
	return &renderer{
		renderJson: func(_ context.Context, w io.Writer) error {
			pretty, err := fancyJSON(it)
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
		},
		renderTemplate: func(_ context.Context, t *template.Template, w io.Writer) error {
			return t.Execute(w, it)
		},
	}
}

func (r *renderer) renderWithTemplate(ctx context.Context, template string) error {
	// TODO: add terminal width & white/dark theme detection
	c := fromContext(ctx)
	switch c.outputFormat {
	case flags.OutputJSON:
		return r.renderJson(ctx, c.out)
	case flags.OutputText:
		if r.renderTemplate != nil && template != "" {
			return r.renderUsingTemplate(ctx, c.out, template)
		}
		if r.renderText != nil {
			return r.renderText(ctx, c.out)
		}
		return r.renderJson(ctx, c.out)
	default:
		return fmt.Errorf("invalid output format: %s", c.outputFormat)
	}
}

func Render(ctx context.Context, v any) error {
	c := fromContext(ctx)
	return RenderWithTemplate(ctx, v, c.template)
}

func RenderWithTemplate(ctx context.Context, v any, template string) error {
	return New(v).renderWithTemplate(ctx, template)
}

func RenderJson(ctx context.Context, v any) error {
	c := fromContext(ctx)
	return New(v).renderJson(ctx, c.out)
}

func RenderReader(ctx context.Context, r io.Reader) error {
	return New(r).renderWithTemplate(ctx, "")
}

func (r *renderer) renderUsingTemplate(ctx context.Context, w io.Writer, tmpl string) error {
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
