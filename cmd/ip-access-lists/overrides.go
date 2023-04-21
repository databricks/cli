package ip_access_lists

import "github.com/databricks/bricks/lib/ui"

func init() {
	// this command still has no Web UI
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.ListId|green}}	{{.Label}}	{{join .IpAddresses ","}}	{{if eq .ListType "ALLOW"}}{{"ALLOW"|green}}{{else}}{{"BLOCK"|red}}{{end}}	{{if .Enabled}}{{"ENABLED"|green}}{{else}}{{"DISABLED"|red}}{{end}}
	{{end}}`)
}
