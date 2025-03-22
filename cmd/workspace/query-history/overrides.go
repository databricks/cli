package query_history

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *sql.ListQueryHistoryRequest) {
	// TODO: figure out the right format
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .Res}}{{.UserName}}	{{cyan "%s" .Status}}	{{.QueryText}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
