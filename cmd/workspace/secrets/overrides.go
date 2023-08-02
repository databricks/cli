package secrets

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func cmdOverride(cmd *cobra.Command) {
	cmd.AddCommand(newPutSecret())
}

func listScopesOverride(listScopesCmd *cobra.Command) {
	listScopesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "Scope"}}	{{header "Backend Type"}}
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
	listScopesOverrides = append(listScopesOverrides, listScopesOverride)
}
