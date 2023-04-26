package networks

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.NetworkId | green}}	{{.NetworkName}}	{{.WorkspaceId}}	{{.VpcStatus}}
	{{end}}`)
}
