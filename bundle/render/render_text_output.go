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

const summaryTemplate = `{{- if .Name -}}
Name: {{ .Name | bold }}
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

{{ end -}}

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
	if recommendations := len(diags.Filter(diag.Recommendation)); recommendations > 0 {
		parts = append(parts, color.BlueString(pluralize(recommendations, "recommendation", "recommendations")))
	}
	switch {
	case len(parts) >= 3:
		first := strings.Join(parts[:len(parts)-1], ", ")
		last := parts[len(parts)-1]
		return fmt.Sprintf("Found %s, and %s", first, last)
	case len(parts) == 2:
		return fmt.Sprintf("Found %s and %s", parts[0], parts[1])
	case len(parts) == 1:
		return fmt.Sprintf("Found %s", parts[0])
	default:
		// No diagnostics to print.
		return color.GreenString("Validation OK!")
	}
}

func renderSummaryTemplate(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
	if b == nil {
		return renderSummaryTemplate(out, &bundle.Bundle{}, diags)
	}

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
	recommendationT := template.Must(template.New("info").Funcs(renderFuncMap).Parse(recommendationTemplate))

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

		for i := range d.Locations {
			if b == nil {
				break
			}

			// Make location relative to bundle root
			if d.Locations[i].File != "" {
				out, err := filepath.Rel(b.RootPath, d.Locations[i].File)
				// if we can't relativize the path, just use path as-is
				if err == nil {
					d.Locations[i].File = out
				}
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

// RenderOptions contains options for rendering diagnostics.
type RenderOptions struct {
	// variable to include leading new line

	RenderSummaryTable bool
}

// RenderTextOutput renders the diagnostics in a human-readable format.
func RenderTextOutput(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics, opts RenderOptions) error {
	err := renderDiagnostics(out, b, diags)
	if err != nil {
		return fmt.Errorf("failed to render diagnostics: %w", err)
	}

	if opts.RenderSummaryTable {
		err = renderSummaryTemplate(out, b, diags)
		if err != nil {
			return fmt.Errorf("failed to render summary: %w", err)
		}
	}

	return nil
}
