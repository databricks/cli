// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringMaterializedFeatureOfflineStoreConfig struct {
	CatalogName     string `json:"catalog_name"`
	SchemaName      string `json:"schema_name"`
	TableNamePrefix string `json:"table_name_prefix"`
}

type DataSourceFeatureEngineeringMaterializedFeatureOnlineStoreConfig struct {
	CatalogName     string `json:"catalog_name"`
	OnlineStoreName string `json:"online_store_name"`
	SchemaName      string `json:"schema_name"`
	TableNamePrefix string `json:"table_name_prefix"`
}

type DataSourceFeatureEngineeringMaterializedFeature struct {
	CronSchedule            string                                                             `json:"cron_schedule,omitempty"`
	FeatureName             string                                                             `json:"feature_name,omitempty"`
	LastMaterializationTime string                                                             `json:"last_materialization_time,omitempty"`
	MaterializedFeatureId   string                                                             `json:"materialized_feature_id"`
	OfflineStoreConfig      *DataSourceFeatureEngineeringMaterializedFeatureOfflineStoreConfig `json:"offline_store_config,omitempty"`
	OnlineStoreConfig       *DataSourceFeatureEngineeringMaterializedFeatureOnlineStoreConfig  `json:"online_store_config,omitempty"`
	PipelineScheduleState   string                                                             `json:"pipeline_schedule_state,omitempty"`
	TableName               string                                                             `json:"table_name,omitempty"`
}
