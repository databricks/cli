package cmdio

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"
	"time"

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

func renderJson(w io.Writer, v any) error {
	pretty, err := fancyJSON(v)
	if err != nil {
		return err
	}
	_, err = w.Write(pretty)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func renderTemplate(w io.Writer, tmpl string, v any) error {
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
	err = t.Execute(tw, v)
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
