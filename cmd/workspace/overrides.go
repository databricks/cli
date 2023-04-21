package workspace

import "github.com/databricks/bricks/lib/ui"

func init() {
	listReq.Path = "/"
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Type"}}	{{white "Language"}}	{{white "Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path|white}}
	{{end}}`)
}
