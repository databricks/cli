package instance_profiles

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.InstanceProfileArn}}
	{{end}}`)
}
