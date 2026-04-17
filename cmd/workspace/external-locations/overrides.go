package external_locations

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, _ *catalog.ListExternalLocationsRequest) {
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Credential"}}	{{header "URL"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.CredentialName|cyan}}	{{.Url}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Name", func(l catalog.ExternalLocationInfo) string { return l.Name }),
		tableview.Col("Credential", func(l catalog.ExternalLocationInfo) string { return l.CredentialName }),
		tableview.Col("URL", func(l catalog.ExternalLocationInfo) string { return l.Url }),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
