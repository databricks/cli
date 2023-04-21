package dbfs

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{if .IsDir}}📂{{else}}📄{{end}}	{{.FileSize}}	{{.Path|green}}
	{{end}}`)
}
