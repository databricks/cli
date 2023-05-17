package groups

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listReq.Attributes = "id,displayName"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName}}
	{{end}}`)
}
