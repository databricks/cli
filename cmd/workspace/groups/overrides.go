package groups

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *iam.ListGroupsRequest) {
	listReq.Attributes = "id,displayName"
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.DisplayName}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
