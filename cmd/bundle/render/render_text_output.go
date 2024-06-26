package render

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/fatih/color"
)

var renderFuncMap = template.FuncMap{
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
{{- if .Path.String }}
  {{ "at " }}{{ .Path.String | green }}
{{- end }}
{{- if .Location.File }}
  {{ "in " }}{{ .Location.String | cyan }}
{{- end }}
{{- if .Detail }}

{{ .Detail }}
{{- end }}
`

const warningTemplate = `{{ "Warning" | yellow }}: {{ .Summary }}
{{- if .Path.String }}
  {{ "at " }}{{ .Path.String | green }}
{{- end }}
{{- if .Location.File }}
  {{ "in " }}{{ .Location.String | cyan }}
{{- end }}
`

const summaryTemplate = `{{- if .Name }}
Name: {{ .Name | bold }}
{{- end }}
{{- if .Target }}
Target: {{ .Target | bold }}
{{- end }}
{{- if or .User .Host .Path }}
Workspace:
{{- if .Host }}
  Host: {{ .Host | bold }}
{{- end }}
{{- if .User }}
  User: {{ .User | bold }}
{{- end }}
{{- if .Path }}
  Path: {{ .Path | bold }}
{{- end }}
{{- end }}

{{ .Trailer }}
`

func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, singular)
	}
	return fmt.Sprintf("%d %s", n, plural)
}

func buildTrailer(diags diag.Diagnostics) string {
	parts := []string{}
	if errors := len(diags.Filter(diag.Error)); errors > 0 {
		parts = append(parts, color.RedString(pluralize(errors, "error", "errors")))
	}
	if warnings := len(diags.Filter(diag.Warning)); warnings > 0 {
		parts = append(parts, color.YellowString(pluralize(warnings, "warning", "warnings")))
	}
	if len(parts) > 0 {
		return fmt.Sprintf("Found %s", strings.Join(parts, " and "))
	} else {
		return color.GreenString("Validation OK!")
	}
}

func renderSummaryTemplate(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
	var currentUser = &iam.User{}

	if b.Config.Workspace.CurrentUser != nil {
		if b.Config.Workspace.CurrentUser.User != nil {
			currentUser = b.Config.Workspace.CurrentUser.User
		}
	}

	t := template.Must(template.New("summary").Funcs(renderFuncMap).Parse(summaryTemplate))
	err := t.Execute(out, map[string]any{
		"Name":    b.Config.Bundle.Name,
		"Target":  b.Config.Bundle.Target,
		"User":    currentUser.UserName,
		"Path":    b.Config.Workspace.RootPath,
		"Host":    b.Config.Workspace.Host,
		"Trailer": buildTrailer(diags),
	})

	return err
}

func renderDiagnostics(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
	errorT := template.Must(template.New("error").Funcs(renderFuncMap).Parse(errorTemplate))
	warningT := template.Must(template.New("warning").Funcs(renderFuncMap).Parse(warningTemplate))

	// Print errors and warnings.
	for _, d := range diags {
		var t *template.Template
		switch d.Severity {
		case diag.Error:
			t = errorT
		case diag.Warning:
			t = warningT
		}

		// Make file relative to bundle root
		if d.Location.File != "" {
			out, err := filepath.Rel(b.RootPath, d.Location.File)
			// if we can't relativize the path, just use path as-is
			if err == nil {
				d.Location.File = out
			}
		}

		// Render the diagnostic with the appropriate template.
		err := t.Execute(out, d)
		if err != nil {
			return fmt.Errorf("failed to render template: %w", err)
		}
	}

	return nil
}

// RenderTextOutput renders the diagnostics in a human-readable format.
//
// It prints errors and returns root.AlreadyPrintedErr if there are any errors.
// If there are unexpected errors during rendering, it returns an error different from root.AlreadyPrintedErr.
func RenderTextOutput(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
	err := renderDiagnostics(out, b, diags)
	if err != nil {
		return fmt.Errorf("failed to render diagnostics: %w", err)
	}

	err = renderSummaryTemplate(out, b, diags)
	if err != nil {
		return fmt.Errorf("failed to render summary: %w", err)
	}

	return nil
}
