package query_history

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	// TODO: figure out the right format
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.UserName}}	{{white "%s" .Status}}	{{.QueryText}}
	{{end}}`)
}
