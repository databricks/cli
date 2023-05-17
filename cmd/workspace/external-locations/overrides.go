package external_locations

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Name"}}	{{white "Credential"}}	{{white "URL"}}
	{{range .}}{{.Name|green}}	{{.CredentialName|blue}}	{{.Url}}
	{{end}}`)
}
