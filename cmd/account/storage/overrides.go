package storage

import "github.com/databricks/bricks/libs/cmdio"

func init() {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.StorageConfigurationId | green}}	{{.StorageConfigurationName}}	{{.RootBucketInfo.BucketName}}
	{{end}}`)
}
