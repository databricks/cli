package jobs

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{.Settings.Name}}
	{{end}}`)

	listRunsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "Job ID"}}	{{header "Run ID"}}	{{header "Result State"}}	URL
	{{range .}}{{green "%d" .JobId}}	{{cyan "%d" .RunId}}	{{if eq .State.ResultState "SUCCESS"}}{{"SUCCESS"|green}}{{else}}{{red "%s" .State.ResultState}}{{end}}	{{.RunPageUrl}}
	{{end}}`)
}
