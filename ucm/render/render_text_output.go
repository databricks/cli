package render

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/ucm"
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

func renderSummaryHeaderTemplate(ctx context.Context, out io.Writer, u *ucm.Ucm) error {
	if u == nil {
		return nil
	}

	currentUser := &iam.User{}

	if u.CurrentUser != nil {
		if u.CurrentUser.User != nil {
			currentUser = u.CurrentUser.User
		}
	}

	t := template.Must(template.New("summary").Funcs(renderFuncMap).Parse(summaryHeaderTemplate))
	err := t.Execute(out, map[string]any{
		"Name":   u.Config.Ucm.Name,
		"Target": u.Config.Ucm.Target,
		"User":   currentUser.UserName,
		"Path":   u.Config.Workspace.RootPath,
		"Host":   u.Config.Workspace.Host,
	})

	return err
}

// RenderDiagnostics renders the diagnostics in a human-readable format.
func RenderDiagnosticsSummary(ctx context.Context, out io.Writer, u *ucm.Ucm) error {
	if u != nil {
		err := renderSummaryHeaderTemplate(ctx, out, u)
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

func RenderSummary(ctx context.Context, out io.Writer, u *ucm.Ucm) error {
	if u == nil {
		return nil
	}
	if err := renderSummaryHeaderTemplate(ctx, out, u); err != nil {
		return err
	}

	resourceGroups := resourceGroupsForUcm(u)

	if err := renderResourcesTemplate(out, resourceGroups); err != nil {
		return fmt.Errorf("failed to render resources template: %w", err)
	}

	return nil
}

// Helper function to sort and render resource groups using the template
func renderResourcesTemplate(out io.Writer, resourceGroups []ResourceGroup) error {
	// Sort everything to ensure consistent output
	slices.SortFunc(resourceGroups, func(a, b ResourceGroup) int {
		return cmp.Compare(a.GroupName, b.GroupName)
	})
	for _, group := range resourceGroups {
		slices.SortFunc(group.Resources, func(a, b ResourceInfo) int {
			return cmp.Compare(a.Key, b.Key)
		})
	}

	t := template.Must(template.New("resources").Funcs(renderFuncMap).Parse(resourcesTemplate))

	return t.Execute(out, resourceGroups)
}
