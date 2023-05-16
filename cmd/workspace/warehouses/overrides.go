package warehouses

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Size"}}	{{white "State"}}
	{{range .}}{{.Id|green}}	{{.Name|white}}	{{.ClusterSize|white}}	{{if eq .State "RUNNING"}}{{"RUNNING"|green}}{{else if eq .State "STOPPED"}}{{"STOPPED"|red}}{{else}}{{blue "%s" .State}}{{end}}
	{{end}}`)
}
