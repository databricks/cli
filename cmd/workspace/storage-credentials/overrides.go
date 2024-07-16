package storage_credentials

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *catalog.ListStorageCredentialsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Credentials"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .DatabricksGcpServiceAccount}}{{.DatabricksGcpServiceAccount.Email}}{{end}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
