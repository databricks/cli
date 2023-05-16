package repos

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .Id}}	{{.Path}}	{{.Branch|blue}}	{{.Url|cyan}}
	{{end}}`)
}
