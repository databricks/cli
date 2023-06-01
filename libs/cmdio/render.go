package cmdio

import (
	"encoding/json"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"

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
		"black":   color.BlackString,
		"white":   color.WhiteString,
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
	f.FieldColor = color.New(color.FgWhite, color.Bold)
	f.FieldQuoteColor = color.New(color.FgWhite)

	return jsoncolor.MarshalIndentWithFormatter(v, "", "  ", f)
}
