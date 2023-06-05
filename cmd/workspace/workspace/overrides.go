package workspace

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listReq.Path = "/"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Type"}}	{{"Language"}}	{{"Path"}}
	{{range .}}{{green "%d" .ObjectId}}	{{blue "%s" .ObjectType}}	{{cyan "%s" .Language}}	{{.Path}}
	{{end}}`)
}
