package external_locations

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Credential"}}	{{header "URL"}}
	{{range .}}{{.Name|green}}	{{.CredentialName|cyan}}	{{.Url}}
	{{end}}`)
}
