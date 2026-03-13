package jobs

import (
	"context"
	"strconv"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/tableview"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *jobs.ListJobsRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{.Settings.Name}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		{Header: "Job ID", Extract: func(v any) string {
			return strconv.FormatInt(v.(jobs.BaseJob).JobId, 10)
		}},
		{Header: "Name", Extract: func(v any) string {
			if v.(jobs.BaseJob).Settings != nil {
				return v.(jobs.BaseJob).Settings.Name
			}
			return ""
		}},
	}

	tableview.RegisterConfig(listCmd, tableview.TableConfig{
		Columns: columns,
		Search: &tableview.SearchConfig{
			Placeholder: "Search by exact name...",
			NewIterator: func(ctx context.Context, query string) tableview.RowIterator {
				req := *listReq
				req.Name = query
				w := cmdctx.WorkspaceClient(ctx)
				return tableview.WrapIterator(w.Jobs.List(ctx, req), columns)
			},
		},
	})
}

func listRunsOverride(listRunsCmd *cobra.Command, _ *jobs.ListRunsRequest) {
	listRunsCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "Job ID"}}	{{header "Run ID"}}	{{header "Result State"}}	URL`)
	listRunsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{cyan "%d" .RunId}}	{{if eq .State.ResultState "SUCCESS"}}{{"SUCCESS"|green}}{{else}}{{red "%s" .State.ResultState}}{{end}}	{{.RunPageUrl}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
	listRunsOverrides = append(listRunsOverrides, listRunsOverride)
}
