package shares

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "Name"}}	{{white "Owner"}}
	{{range .}}{{.Name|green}}	{{.Owner|white}}
	{{end}}`)

	// TODO: adapt & improve https://docs.databricks.com/data-sharing/create-share.html#language-CLI
}
