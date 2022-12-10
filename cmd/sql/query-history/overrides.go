package query_history

import "github.com/databricks/bricks/lib/ui"

func init() {
	// TODO: figure out the right format
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.UserName}}	{{white "%s" .Status}}	{{.QueryText}}
	{{end}}`)
}
