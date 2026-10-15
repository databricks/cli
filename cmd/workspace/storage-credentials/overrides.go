package storage_credentials

import (
	"github.com/databricks/cli/libs/cmdio"
	uccredentials "github.com/databricks/cli/libs/sdktypes/uc/credentials/v1"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command, listReq *uccredentials.ListStorageCredentialsRequest) {
	listCmd.Annotations["headerTemplate"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Credentials"}}`)
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .DatabricksGcpServiceAccount}}{{.DatabricksGcpServiceAccount.Email}}{{end}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
