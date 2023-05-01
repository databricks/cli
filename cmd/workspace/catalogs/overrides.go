package catalogs

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Name"}}	{{white "Type"}}	{{white "Comment"}}
	{{range .}}{{.Name|green}}	{{blue "%s" .CatalogType}}	{{.Comment}}
	{{end}}`)
}
