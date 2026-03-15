package external_locations

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *catalog.ListExternalLocationsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Name"}}	{{header "Credential"}}	{{header "URL"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.CredentialName|cyan}}	{{.Url}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Name", Extract: func(v any) string {
			return v.(catalog.ExternalLocationInfo).Name
		}},
		{Header: "Credential", Extract: func(v any) string {
			return v.(catalog.ExternalLocationInfo).CredentialName
		}},
		{Header: "URL", Extract: func(v any) string {
			return v.(catalog.ExternalLocationInfo).Url
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
