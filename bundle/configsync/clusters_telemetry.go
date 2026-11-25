package configsync

import "github.com/databricks/cli/libs/telemetry/protos"

func init() {
	registerResourceIDField("clusters", func(ids *protos.BundleConfigRemoteSyncResourceIds) *[]string { return &ids.ResourceClusterIDs })
}
