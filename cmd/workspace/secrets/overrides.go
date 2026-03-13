package secrets

import (
	"strconv"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
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

	columns := []tableview.ColumnDef{
		{Header: "Scope", Extract: func(v any) string {
			return v.(workspace.SecretScope).Name
		}},
		{Header: "Backend Type", Extract: func(v any) string {
			return string(v.(workspace.SecretScope).BackendType)
		}},
	}

	tableview.RegisterConfig(listScopesCmd, tableview.TableConfig{Columns: columns})
}

func listSecretsOverride(listSecretsCommand *cobra.Command, _ *workspace.ListSecretsRequest) {
	listSecretsCommand.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Key"}}	{{header "Last Updated Timestamp"}}`)
	listSecretsCommand.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Key|green}}	{{.LastUpdatedTimestamp}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Key", Extract: func(v any) string {
			return v.(workspace.SecretMetadata).Key
		}},
		{Header: "Last Updated", Extract: func(v any) string {
			ts := v.(workspace.SecretMetadata).LastUpdatedTimestamp
			if ts == 0 {
				return ""
			}
			return strconv.FormatInt(ts, 10)
		}},
	}

	tableview.RegisterConfig(listSecretsCommand, tableview.TableConfig{Columns: columns})
}

func init() {
	cmdOverrides = append(cmdOverrides, cmdOverride)
	listScopesOverrides = append(listScopesOverrides, listScopesOverride)
	listSecretsOverrides = append(listSecretsOverrides, listSecretsOverride)
}
