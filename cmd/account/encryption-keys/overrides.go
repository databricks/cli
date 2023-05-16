package encryption_keys

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CustomerManagedKeyId | green}}	{{range .UseCases}}{{.}} {{end}}	{{.AwsKeyInfo.KeyArn}}
	{{end}}`)
}
