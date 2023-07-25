package credentials

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CredentialsId | green}}	{{.CredentialsName}}	{{.AwsCredentials.StsRole.RoleArn}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
