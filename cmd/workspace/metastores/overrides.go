package metastores

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{"Region"}}
	{{range .}}{{.MetastoreId|green}}	{{.Name|cyan}}	{{.Region}}
	{{end}}`)
}
