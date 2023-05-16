package instance_profiles

import "github.com/databricks/cli/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.InstanceProfileArn}}
	{{end}}`)
}
