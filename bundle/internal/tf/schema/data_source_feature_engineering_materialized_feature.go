// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringMaterializedFeatureCronScheduleTrigger struct {
	CronExpression string `json:"cron_expression,omitempty"`
}

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

type DataSourceFeatureEngineeringMaterializedFeatureProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeatureStreamingMode struct {
	Mode string `json:"mode,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeatureTableTrigger struct {
}

type DataSourceFeatureEngineeringMaterializedFeature struct {
	CronSchedule            string                                                              `json:"cron_schedule,omitempty"`
	CronScheduleTrigger     *DataSourceFeatureEngineeringMaterializedFeatureCronScheduleTrigger `json:"cron_schedule_trigger,omitempty"`
	FeatureName             string                                                              `json:"feature_name,omitempty"`
	IsOnline                bool                                                                `json:"is_online,omitempty"`
	LastMaterializationTime string                                                              `json:"last_materialization_time,omitempty"`
	MaterializedFeatureId   string                                                              `json:"materialized_feature_id"`
	OfflineStoreConfig      *DataSourceFeatureEngineeringMaterializedFeatureOfflineStoreConfig  `json:"offline_store_config,omitempty"`
	OnlineStoreConfig       *DataSourceFeatureEngineeringMaterializedFeatureOnlineStoreConfig   `json:"online_store_config,omitempty"`
	PipelineScheduleState   string                                                              `json:"pipeline_schedule_state,omitempty"`
	ProviderConfig          *DataSourceFeatureEngineeringMaterializedFeatureProviderConfig      `json:"provider_config,omitempty"`
	StreamingMode           *DataSourceFeatureEngineeringMaterializedFeatureStreamingMode       `json:"streaming_mode,omitempty"`
	TableName               string                                                              `json:"table_name,omitempty"`
	TableTrigger            *DataSourceFeatureEngineeringMaterializedFeatureTableTrigger        `json:"table_trigger,omitempty"`
}
