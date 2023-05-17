package storage_credentials

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Credentials"}}
	{{range .}}{{.Id|green}}	{{.Name|white}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .GcpServiceAccountKey}}{{.Email}}{{end}}
	{{end}}`)
}
