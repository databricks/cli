package main

import (
	"github.com/databricks/cli/libs/template"
)

func Render(templateName string, params map[string]any) map[string]string {
	tmpl := template.GetTemplate(templateName)
	return tmpl.Render(params)
}
