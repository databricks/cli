package networks

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.NetworkId | green}}	{{.NetworkName}}	{{.WorkspaceId}}	{{.VpcStatus}}
	{{end}}`)
}
