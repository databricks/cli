package lakeview

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *dashboards.ListDashboardsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Dashboard ID"}}	{{header "Name"}}	{{header "State"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%s" .DashboardId}}	{{.DisplayName}}	{{blue "%s" .LifecycleState}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Dashboard ID", Extract: func(v any) string {
			return v.(dashboards.Dashboard).DashboardId
		}},
		{Header: "Name", Extract: func(v any) string {
			return v.(dashboards.Dashboard).DisplayName
		}},
		{Header: "State", Extract: func(v any) string {
			return string(v.(dashboards.Dashboard).LifecycleState)
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{Columns: columns})
}

func publishOverride(cmd *cobra.Command, req *dashboards.PublishRequest) {
	originalRunE := cmd.RunE
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// Force send embed_credentials even when false, otherwise the API defaults to true.
		req.ForceSendFields = append(req.ForceSendFields, "EmbedCredentials")
		return originalRunE(cmd, args)
	}
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	publishOverrides = append(publishOverrides, publishOverride)
}
