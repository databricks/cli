package secrets

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listScopesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Scope"}}	{{white "Backend Type"}}
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)
}
