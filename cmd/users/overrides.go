package users

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listReq.Attributes = "id,userName,groups,active"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.UserName}}	{{range .Groups}}{{.Display}} {{end}}	{{if .Active}}{{"ACTIVE"|green}}{{else}}DISABLED{{end}}
	{{end}}`)
}
