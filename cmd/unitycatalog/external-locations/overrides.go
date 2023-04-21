package external_locations

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "Name"}}	{{white "Credential"}}	{{white "URL"}}
	{{range .}}{{.Name|green}}	{{.CredentialName|blue}}	{{.Url}}
	{{end}}`)
}
