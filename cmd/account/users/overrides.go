package users

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	// Request only the columns the template shows unless --attributes is given.
	_ = listCmd.Flags().Lookup("attributes").Value.Set("id,userName,groups,active")
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.UserName}}	{{range .Groups}}{{.Display}} {{end}}	{{if .Active}}{{"ACTIVE"|green}}{{else}}DISABLED{{end}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
