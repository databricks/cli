package workspace

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listReq.Path = "/"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Type"}}	{{header "Language"}}	{{header "Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path|cyan}}
	{{end}}`)
}
