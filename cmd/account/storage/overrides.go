package storage

import (
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func listOverride(listCmd *cobra.Command) {
	listCmd.Annotations["template"] = cmdio.Heredoc(`
	{{range .}}{{.StorageConfigurationId | green}}	{{.StorageConfigurationName}}	{{.RootBucketInfo.BucketName}}
	{{end}}`)
}

func init() {
	listOverrides = append(listOverrides, listOverride)
}
