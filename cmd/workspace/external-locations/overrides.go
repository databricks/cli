package external_locations

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Name"}}	{{white "Credential"}}	{{white "URL"}}
	{{range .}}{{.Name|green}}	{{.CredentialName|blue}}	{{.Url}}
	{{end}}`)
}
