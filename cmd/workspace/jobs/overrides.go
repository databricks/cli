package jobs

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{green "%d" .JobId}}	{{.Settings.Name}}
	{{end}}`)

	listRunsCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"Job ID"}}	{{"Run ID"}}	{{"Result State"}}	URL
	{{range .}}{{green "%d" .JobId}}	{{blue "%d" .RunId}}	{{if eq .State.ResultState "SUCCESS"}}{{"SUCCESS"|green}}{{else}}{{red "%s" .State.ResultState}}{{end}}	{{.RunPageUrl}}
	{{end}}`)
}
