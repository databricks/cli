package schemas

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "Full Name"}}	{{white "Owner"}}	{{white "Comment"}}
	{{range .}}{{.FullName|green}}	{{.Owner|white}}	{{.Comment}}
	{{end}}`)
}
