package configsync

import "github.com/databricks/cli/libs/telemetry/protos"

func init() {
	registerResourceIDField("jobs", func(ids *protos.BundleConfigRemoteSyncResourceIds) *[]string { return &ids.ResourceJobIDs })
}
