package bundle

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/flags"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var validateFuncMap = template.FuncMap{
	"red":     color.RedString,
	"green":   color.GreenString,
	"blue":    color.BlueString,
	"yellow":  color.YellowString,
	"magenta": color.MagentaString,
	"cyan":    color.CyanString,
	"bold": func(format string, a ...interface{}) string {
		return color.New(color.Bold).Sprintf(format, a...)
	},
	"italic": func(format string, a ...interface{}) string {
		return color.New(color.Italic).Sprintf(format, a...)
	},
}

const errorTemplate = `{{ "Error" | red }}: {{ .Summary }}
  {{ "at " }}{{ .Path.String | green }}
  {{ "in " }}{{ .Location.String | cyan }}

`

const warningTemplate = `{{ "Warning" | yellow }}: {{ .Summary }}
  {{ "at " }}{{ .Path.String | green }}
  {{ "in " }}{{ .Location.String | cyan }}

`

const summaryTemplate = `Name: {{ .Config.Bundle.Name | bold }}
Target: {{ .Config.Bundle.Target | bold }}
Workspace:
  Host: {{ .Config.Workspace.Host | bold }}
  User: {{ .Config.Workspace.CurrentUser.UserName | bold }}
  Path: {{ .Config.Workspace.RootPath | bold }}

{{ .Trailer }}
`

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func renderTextOutput(cmd *cobra.Command, b *bundle.Bundle, diags diag.Diagnostics) error {
	// Print errors and warnings.
	for _, d := range diags {
		var t *template.Template
		switch d.Severity {
		case diag.Error:
			t = template.Must(template.New("error").Funcs(validateFuncMap).Parse(errorTemplate))
		case diag.Warning:
			t = template.Must(template.New("warning").Funcs(validateFuncMap).Parse(warningTemplate))
		}

		// Make file relative to bundle root
		if d.Location.File != "" {
			out, _ := filepath.Rel(b.RootPath, d.Location.File)
			d.Location.File = out
		}

		// Render the diagnostic with the appropriate template.
		err := t.Execute(cmd.OutOrStdout(), d)
		if err != nil {
			return err
		}
	}

	trailer := []string{}
	if errors := len(diags.Filter(diag.Error)); errors > 0 {
		trailer = append(trailer, color.RedString(pluralize(errors, "error", "errors")))
	}
	if warnings := len(diags.Filter(diag.Warning)); warnings > 0 {
		trailer = append(trailer, color.YellowString(pluralize(warnings, "warning", "warnings")))
	}
	if len(trailer) == 0 {
		trailer = append(trailer, color.GreenString("Validation OK!"))
	}

	// Print validation summary.
	t := template.Must(template.New("summary").Funcs(validateFuncMap).Parse(summaryTemplate))
	err := t.Execute(cmd.OutOrStdout(), map[string]any{
		"Config":  b.Config,
		"Trailer": strings.Join(trailer, ", "),
	})
	if err != nil {
		return err
	}

	return diags.Error()
}

func renderJsonOutput(cmd *cobra.Command, b *bundle.Bundle, diags diag.Diagnostics) error {
	buf, err := json.MarshalIndent(b.Config, "", "  ")
	if err != nil {
		return err
	}
	cmd.OutOrStdout().Write(buf)
	return diags.Error()
}

func newValidateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Args:  root.NoArgs,
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		b, diags := utils.ConfigureBundleWithVariables(cmd)
		if err := diags.Error(); err != nil {
			return diags.Error()
		}

		diags = diags.Extend(bundle.Apply(ctx, b, phases.Initialize()))
		if err := diags.Error(); err != nil {
			return err
		}

		switch root.OutputType(cmd) {
		case flags.OutputText:
			return renderTextOutput(cmd, b, diags)
		case flags.OutputJSON:
			return renderJsonOutput(cmd, b, diags)
		default:
			return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
		}
	}

	return cmd
}
