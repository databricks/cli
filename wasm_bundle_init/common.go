package main

import (
	"context"

	"github.com/databricks/cli/libs/template"
)

func Render(templateName string, params map[string]string) map[string]string {
	ctx := context.Background()
	tmpl := template.GetTemplate("default-python")
	_ = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	return tmpl.Writer.GetOutput()
}
