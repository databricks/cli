package jobs

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *jobs.ListJobsRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{.Settings.Name}}
	{{end}}`)
}

func listRunsOverride(listRunsCmd *cobra.Command, listRunsReq *jobs.ListRunsRequest) {
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
