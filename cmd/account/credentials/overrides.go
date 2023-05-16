package credentials

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CredentialsId | green}}	{{.CredentialsName}}	{{.AwsCredentials.StsRole.RoleArn}}
	{{end}}`)
}
