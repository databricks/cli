package configsync

import "github.com/databricks/cli/libs/telemetry/protos"

func init() {
	registerResourceIDField("pipelines", func(ids *protos.BundleConfigRemoteSyncResourceIds) *[]string { return &ids.ResourcePipelineIDs })
}
