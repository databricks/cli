package main

import (
	"context"

	"github.com/databricks/cli/libs/template"
)

func Render(templateName string, params map[string]string) map[string]string {
	ctx := context.Background()
	tmpl := template.GetTemplate(templateName)
	
	// Configure the writer with parameters
	writer, ok := tmpl.Writer.(*template.DefaultWriter)
	if ok && len(params) > 0 {
		writer.SetParams(params)
	}
	
	_ = tmpl.Writer.Materialize(ctx, tmpl.Reader)
	return tmpl.Writer.GetOutput()
}
