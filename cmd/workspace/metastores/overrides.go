package metastores

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Name"}}	{{"Region"}}
	{{range .}}{{.MetastoreId|green}}	{{.Name}}	{{.Region}}
	{{end}}`)
}
