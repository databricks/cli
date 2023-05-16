package workspaces

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .WorkspaceId}}	{{.WorkspaceName}}	{{.WorkspaceStatus}}
	{{end}}`)
}
