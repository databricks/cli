package ui

import (
	"encoding/json"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/fatih/color"
	"github.com/nwidger/jsoncolor"
	"github.com/spf13/cobra"
)

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

// TODO: move to a separate package
func Render(cmd *cobra.Command, v any) error {
	// TODO: add terminal width & white/dark theme detection
	tmpl, ok := cmd.Annotations["template"]
	if ok {
		tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
		t, err := template.New("command").Funcs(template.FuncMap{
			// TODO: consume enums and integers
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
			"pretty_json": func(in string) (string, error) {
				var tmp any
				err := json.Unmarshal([]byte(in), &tmp)
				if err != nil {
					return "", err
				}
				b, err := MarshalJSON(tmp)
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
	// TODO: render in other formats
	pretty, err := MarshalJSON(v)
	if err != nil {
		return err
	}
	cmd.OutOrStdout().Write(pretty)
	return nil
}

func MarshalJSON(v any) ([]byte, error) {
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
	// KeyColor:        color.New(color.FgWhite),
	// StringColor:     color.New(color.FgGreen),
	// BoolColor:       color.New(),
	// NullColor:       color.New(),

	return jsoncolor.MarshalIndentWithFormatter(v, "", "  ", f)
}
