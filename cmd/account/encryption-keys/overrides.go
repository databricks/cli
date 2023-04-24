package encryption_keys

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.CustomerManagedKeyId | green}}	{{range .UseCases}}{{.}} {{end}}	{{.AwsKeyInfo.KeyArn}}
	{{end}}`)
}
