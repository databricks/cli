package encryption_keys

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CustomerManagedKeyId | green}}	{{range .UseCases}}{{.}} {{end}}	{{.AwsKeyInfo.KeyArn}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
