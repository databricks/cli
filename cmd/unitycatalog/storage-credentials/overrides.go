package storage_credentials

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{white "ID"}}	{{white "Name"}}	{{white "Credentials"}}
	{{range .}}{{.Id|green}}	{{.Name|white}}	{{if .AwsIamRole}}{{.AwsIamRole.RoleArn}}{{end}}{{if .AzureServicePrincipal}}{{.AzureServicePrincipal.ApplicationId}}{{end}}{{if .GcpServiceAccountKey}}{{.Email}}{{end}}
	{{end}}`)
}
