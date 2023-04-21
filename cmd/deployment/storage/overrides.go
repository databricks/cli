package storage

import "github.com/databricks/bricks/lib/ui"

func init() {
	listCmd.Annotations["template"] = ui.Heredoc(`
	{{range .}}{{.StorageConfigurationId | green}}	{{.StorageConfigurationName}}	{{.RootBucketInfo.BucketName}}
	{{end}}`)
}
