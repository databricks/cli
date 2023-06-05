package query_history

import "github.com/databricks/cli/libs/cmdio"

func init() {
	// TODO: figure out the right format
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.UserName}}	{{.Status}}	{{.QueryText}}
	{{end}}`)
}
