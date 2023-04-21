package credentials

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.CredentialsId | green}}	{{.CredentialsName}}	{{.AwsCredentials.StsRole.RoleArn}}
	{{end}}`)
}
