package workspace

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listReq.Path = "/"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Type"}}	{{white "Language"}}	{{white "Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path|white}}
	{{end}}`)
}
