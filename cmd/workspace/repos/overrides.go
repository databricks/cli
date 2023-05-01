package repos

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .Id}}	{{.Path}}	{{.Branch|blue}}	{{.Url|cyan}}
	{{end}}`)
}
