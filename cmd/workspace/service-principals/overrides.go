package service_principals

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *iam.ListServicePrincipalsRequest) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.ApplicationId}}	{{.DisplayName}}	{{range .Groups}}{{.Display}} {{end}}	{{if .Active}}{{"ACTIVE"|green}}{{else}}DISABLED{{end}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
