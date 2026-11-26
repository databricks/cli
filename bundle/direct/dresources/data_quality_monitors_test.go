package dresources

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/dataquality"
)

func init() {
	registerTestConfig("data_quality_monitors", &resources.DataQualityMonitor{
		ObjectType: "table",
		ObjectId:   "11111111-2222-3333-4444-555555555555",
		DataProfilingConfig: &dataquality.DataProfilingConfig{
			AssetsDir:      "/Workspace/Users/user@example.com/assets",
			OutputSchemaId: "66666666-7777-8888-9999-000000000000",
			Snapshot:       &dataquality.SnapshotConfig{},
		},
	})

	registerCreateRecoveryOptOut("data_quality_monitors", "the read response has no create time")
}
