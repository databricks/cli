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
	// Template is the text-mode fallback for non-interactive/piped output.
	// TableConfig drives the interactive TUI when the terminal supports it.
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{.Settings.Name}}
	{{end}}`)

	columns := []tableview.ColumnDef{
		tableview.Col("Job ID", func(j jobs.BaseJob) string { return strconv.FormatInt(j.JobId, 10) }),
		tableview.Col("Name", func(j jobs.BaseJob) string {
			if j.Settings != nil {
				return j.Settings.Name
			}
			return ""
		}),
	}

	tableview.SetTableConfigOnCmd(listCmd, &tableview.TableConfig{
		Columns: columns,
		Search: &tableview.SearchConfig{
			Placeholder: "Search by exact name...",
			NewIterator: func(ctx context.Context, query string) tableview.RowIterator {
				req := *listReq
				req.Name = query
				req.PageToken = ""
				req.Offset = 0
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
