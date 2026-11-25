package dresources

import "github.com/databricks/cli/bundle/config/resources"

func init() {
	registerTestConfig("quality_monitors", &resources.QualityMonitor{
		TableName:        "main.myschema.mytable",
		AssetsDir:        "/Workspace/Users/user@example.com/assets",
		OutputSchemaName: "main.myschema",
	})

	registerKnownMissingInRemoteType("quality_monitors",
		"skip_builtin_dashboard",
		"warehouse_id",
	)

	registerCreateRecoveryOptOut("quality_monitors", "the read response has no create time")
}
