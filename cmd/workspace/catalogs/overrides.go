package catalogs

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"Name"}}	{{"Type"}}	{{"Comment"}}
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)
}
