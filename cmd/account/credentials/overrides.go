package credentials

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CredentialsId | green}}	{{.CredentialsName}}	{{.AwsCredentials.StsRole.RoleArn}}
	{{end}}`)
}
