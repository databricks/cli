package ip_access_lists

import "github.com/databricks/cli/libs/cmdio"

func init() {
	// this command still has no Web UI
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.ListId|green}}	{{.Label}}	{{join .IpAddresses ","}}	{{if eq .ListType "ALLOW"}}{{"ALLOW"|green}}{{else}}{{"BLOCK"|red}}{{end}}	{{if .Enabled}}{{"ENABLED"|green}}{{else}}{{"DISABLED"|red}}{{end}}
	{{end}}`)
}
