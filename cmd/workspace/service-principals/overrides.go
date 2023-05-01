package service_principals

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.ApplicationId}}	{{.DisplayName}}	{{range .Groups}}{{.Display}} {{end}}	{{if .Active}}{{"ACTIVE"|green}}{{else}}DISABLED{{end}}
	{{end}}`)
}
