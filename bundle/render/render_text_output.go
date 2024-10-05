package render

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
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

const summaryHeaderTemplate = `{{- if .Name -}}
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
{{ end -}}`

const resourcesTemplate = `Resources:
{{- range . }}
  {{ .GroupName }}:
  {{- range .Resources }}
    {{ .Key | bold }}:
      Name: {{ .Name }}
      URL:  {{ if .URL }}{{ .URL | cyan }}{{ else }}{{ "(not deployed)" | cyan }}{{ end }}
  {{- end }}
{{- end }}
`

type ResourceGroup struct {
	GroupName string
	Resources []ResourceInfo
}

type ResourceInfo struct {
	Key  string
	Name string
	URL  string
}

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
		return fmt.Sprintf("Found %s\n", strings.Join(parts, " and "))
	} else {
		return color.GreenString("Validation OK!\n")
	}
}

func renderSummaryHeaderTemplate(out io.Writer, b *bundle.Bundle) error {
	if b == nil {
		return renderSummaryHeaderTemplate(out, &bundle.Bundle{})
	}

	var currentUser = &iam.User{}

	if b.Config.Workspace.CurrentUser != nil {
		if b.Config.Workspace.CurrentUser.User != nil {
			currentUser = b.Config.Workspace.CurrentUser.User
		}
	}

	t := template.Must(template.New("summary").Funcs(renderFuncMap).Parse(summaryHeaderTemplate))
	err := t.Execute(out, map[string]any{
		"Name":   b.Config.Bundle.Name,
		"Target": b.Config.Bundle.Target,
		"User":   currentUser.UserName,
		"Path":   b.Config.Workspace.RootPath,
		"Host":   b.Config.Workspace.Host,
	})

	return err
}

func renderDiagnosticsOnly(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics) error {
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

		for i := range d.Locations {
			if b == nil {
				break
			}

			// Make location relative to bundle root
			if d.Locations[i].File != "" {
				out, err := filepath.Rel(b.BundleRootPath, d.Locations[i].File)
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

// RenderDiagnostics renders the diagnostics in a human-readable format.
func RenderDiagnostics(out io.Writer, b *bundle.Bundle, diags diag.Diagnostics, opts RenderOptions) error {
	err := renderDiagnosticsOnly(out, b, diags)
	if err != nil {
		return fmt.Errorf("failed to render diagnostics: %w", err)
	}

	if opts.RenderSummaryTable {
		if b != nil {
			err = renderSummaryHeaderTemplate(out, b)
			if err != nil {
				return fmt.Errorf("failed to render summary: %w", err)
			}
			io.WriteString(out, "\n")
		}
		trailer := buildTrailer(diags)
		io.WriteString(out, trailer)
	}

	return nil
}

func RenderSummary(ctx context.Context, out io.Writer, b *bundle.Bundle) error {
	if err := renderSummaryHeaderTemplate(out, b); err != nil {
		return err
	}

	var resourceGroups []ResourceGroup

	for group, r := range b.Config.Resources.AllResources() {
		resources := make([]ResourceInfo, 0, len(r))
		for key, resource := range r {
			resources = append(resources, ResourceInfo{
				Key:  key,
				Name: resource.GetName(),
				URL:  resource.GetURL(),
			})
		}

		if len(resources) > 0 {
			capitalizedGroup := strings.ToUpper(group[:1]) + group[1:]
			resourceGroups = append(resourceGroups, ResourceGroup{
				GroupName: capitalizedGroup,
				Resources: resources,
			})
		}
	}

	if err := renderResourcesTemplate(out, resourceGroups); err != nil {
		return fmt.Errorf("failed to render resources template: %w", err)
	}

	return nil
}

// Helper function to sort and render resource groups using the template
func renderResourcesTemplate(out io.Writer, resourceGroups []ResourceGroup) error {
	// Sort everything to ensure consistent output
	sort.Slice(resourceGroups, func(i, j int) bool {
		return resourceGroups[i].GroupName < resourceGroups[j].GroupName
	})
	for _, group := range resourceGroups {
		sort.Slice(group.Resources, func(i, j int) bool {
			return group.Resources[i].Key < group.Resources[j].Key
		})
	}

	t := template.Must(template.New("resources").Funcs(renderFuncMap).Parse(resourcesTemplate))

	return t.Execute(out, resourceGroups)
}
