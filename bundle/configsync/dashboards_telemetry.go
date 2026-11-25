package configsync

import "github.com/databricks/cli/libs/telemetry/protos"

func init() {
	registerResourceIDField("dashboards", func(ids *protos.BundleConfigRemoteSyncResourceIds) *[]string { return &ids.ResourceDashboardIDs })
}
