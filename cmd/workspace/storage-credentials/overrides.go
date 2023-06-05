package storage_credentials

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{header "ID"}}	{{header "Name"}}	{{header "Credentials"}}
	{{range .}}{{.Id|green}}	{{.Name|cyan}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .GcpServiceAccountKey}}{{.Email}}{{end}}
	{{end}}`)
}
