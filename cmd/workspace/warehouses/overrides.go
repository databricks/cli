package warehouses

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Name"}}	{{"Size"}}	{{"State"}}
	{{range .}}{{.Id|green}}	{{.Name}}	{{.ClusterSize}}	{{if eq .State "RUNNING"}}{{"RUNNING"|green}}{{else if eq .State "STOPPED"}}{{"STOPPED"|red}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)
}
