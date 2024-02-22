package secrets

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func cmdOverride(cmd *cobra.Command) {
	cmd.AddCommand(newPutSecret())
}

func listScopesOverride(listScopesCmd *cobra.Command) {
	listScopesCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Scope"}}	{{header "Backend Type"}}`)
	listScopesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)
}

func listSecretsOverride(listSecretsCommand *cobra.Command, _ *workspace.ListSecretsRequest) {
	listSecretsCommand.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Key"}}	{{header "Last Updated Timestamp"}}`)
	listSecretsCommand.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Key|green}}	{{.LastUpdatedTimestamp}}
	{{end}}`)
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
	listScopesOverrides = append(listScopesOverrides, listScopesOverride)
	listSecretsOverrides = append(listSecretsOverrides, listSecretsOverride)
}
