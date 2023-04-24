package groups

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listReq.Attributes = "id,displayName"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName}}
	{{end}}`)
}
