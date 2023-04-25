package instance_profiles

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.InstanceProfileArn}}
	{{end}}`)
}
