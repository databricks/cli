package encryption_keys

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.CustomerManagedKeyId | green}}	{{range .UseCases}}{{.}} {{end}}	{{.AwsKeyInfo.KeyArn}}
	{{end}}`)
}
