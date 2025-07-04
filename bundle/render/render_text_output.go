package render

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/template"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/logdiag"
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
	"bold": func(format string, a ...any) string {
		return color.New(color.Bold).Sprintf(format, a...)
	},
	"italic": func(format string, a ...any) string {
		return color.New(color.Italic).Sprintf(format, a...)
	},
}

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

func buildTrailer(ctx context.Context) string {
	info := logdiag.Copy(ctx)
	var parts []string
	if info.Errors > 0 {
		parts = append(parts, color.RedString(pluralize(info.Errors, "error", "errors")))
	}
	if info.Warnings > 0 {
		parts = append(parts, color.YellowString(pluralize(info.Warnings, "warning", "warnings")))
	}
	if info.Recommendations > 0 {
		parts = append(parts, color.BlueString(pluralize(info.Recommendations, "recommendation", "recommendations")))
	}
	switch {
	case len(parts) >= 3:
		first := strings.Join(parts[:len(parts)-1], ", ")
		last := parts[len(parts)-1]
		return fmt.Sprintf("Found %s, and %s\n", first, last)
	case len(parts) == 2:
		return fmt.Sprintf("Found %s and %s\n", parts[0], parts[1])
	case len(parts) == 1:
		return fmt.Sprintf("Found %s\n", parts[0])
	default:
		// No diagnostics to print.
		return color.GreenString("Validation OK!\n")
	}
}

func renderSummaryHeaderTemplate(ctx context.Context, out io.Writer, b *bundle.Bundle) error {
	if b == nil {
		return nil
	}

	currentUser := &iam.User{}

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

// RenderDiagnostics renders the diagnostics in a human-readable format.
func RenderDiagnosticsSummary(ctx context.Context, out io.Writer, b *bundle.Bundle) error {
	if b != nil {
		err := renderSummaryHeaderTemplate(ctx, out, b)
		if err != nil {
			return fmt.Errorf("failed to render summary: %w", err)
		}
		_, err = io.WriteString(out, "\n")
		if err != nil {
			return err
		}
	}
	trailer := buildTrailer(ctx)
	_, err := io.WriteString(out, trailer)
	if err != nil {
		return err
	}

	return nil
}

func RenderSummary(ctx context.Context, out io.Writer, b *bundle.Bundle) error {
	if b == nil {
		return nil
	}
	if err := renderSummaryHeaderTemplate(ctx, out, b); err != nil {
		return err
	}

	var resourceGroups []ResourceGroup

	for _, group := range b.Config.Resources.AllResources() {
		resources := make([]ResourceInfo, 0, len(group.Resources))
		for key, resource := range group.Resources {
			resources = append(resources, ResourceInfo{
				Key:  key,
				Name: resource.GetName(),
				URL:  resource.GetURL(),
			})
		}

		if len(resources) > 0 {
			resourceGroups = append(resourceGroups, ResourceGroup{
				GroupName: group.Description.PluralTitle,
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
