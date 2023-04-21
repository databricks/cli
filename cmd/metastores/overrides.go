package metastores

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Region"}}
	{{range .}}{{.MetastoreId|green}}	{{.Name|white}}	{{.Region}}
	{{end}}`)
}
