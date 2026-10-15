package query_history

import (
	"github.com/databricks/cli/libs/cmdio"
	queryhistory "github.com/databricks/cli/libs/sdktypes/queryhistory/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *queryhistory.ListQueriesRequest) {
	// TODO: figure out the right format
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .Res}}{{.UserName}}	{{cyan "%s" .Status}}	{{.QueryText}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
