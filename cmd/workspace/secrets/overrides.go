package secrets

import (
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

func cmdOverride(cmd *cobra.Command) {
	cmd.AddCommand(newPutSecret())
}

func listScopesOverride(listScopesCmd *cobra.Command) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listScopesCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Scope"}}	{{header "Backend Type"}}`)
	listScopesCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.BackendType}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Scope", func(s workspace.SecretScope) string { return s.Name }),
		tableview.Col("Backend Type", func(s workspace.SecretScope) string { return string(s.BackendType) }),
	}

	tableview.SetTableConfigOnCmd(listScopesCmd, &tableview.TableConfig{Columns: columns})
}

func listSecretsOverride(listSecretsCommand *cobra.Command, _ *workspace.ListSecretsRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listSecretsCommand.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Key"}}	{{header "Last Updated Timestamp"}}`)
	listSecretsCommand.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Key|green}}	{{.LastUpdatedTimestamp}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Key", func(s workspace.SecretMetadata) string { return s.Key }),
		tableview.Col("Last Updated", func(s workspace.SecretMetadata) string {
			if s.LastUpdatedTimestamp == 0 {
				return ""
			}
			return time.UnixMilli(s.LastUpdatedTimestamp).UTC().Format("2006-01-02 15:04:05")
		}),
	}

	tableview.SetTableConfigOnCmd(listSecretsCommand, &tableview.TableConfig{Columns: columns})
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
	listScopesOverrides = append(listScopesOverrides, listScopesOverride)
	listSecretsOverrides = append(listSecretsOverrides, listSecretsOverride)
}
