package account_service_principals

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.Id|green}}	{{.ApplicationId}}	{{.DisplayName}}	{{range .Roles}}{{.Display}} {{end}}	{{if .Active}}{{"ACTIVE"|green}}{{else}}DISABLED{{end}}
	{{end}}`)
}
