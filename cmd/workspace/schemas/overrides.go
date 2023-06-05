package schemas

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "Full Name"}}	{{header "Owner"}}	{{header "Comment"}}
	{{range .}}{{.FullName|green}}	{{.Owner|cyan}}	{{.Comment}}
	{{end}}`)
}
