package storage_credentials

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{"ID"}}	{{"Name"}}	{{"Credentials"}}
	{{range .}}{{.Id|green}}	{{.Name}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .GcpServiceAccountKey}}{{.Email}}{{end}}
	{{end}}`)
}
