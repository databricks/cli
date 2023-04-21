package secrets

import "github.com/databricks/bricks/lib/ui"

func init() {
	listScopesCmd.Annotations["template"] = ui.Heredoc(`
	{{white "Scope"}}	{{white "Backend Type"}}
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)
}
