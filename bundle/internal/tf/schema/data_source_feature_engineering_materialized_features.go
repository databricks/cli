// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesCronScheduleTrigger struct {
	CronExpression string `json:"cron_expression,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesOfflineStoreConfig struct {
	CatalogName     string `json:"catalog_name"`
	SchemaName      string `json:"schema_name"`
	TableNamePrefix string `json:"table_name_prefix"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesOnlineStoreConfig struct {
	CatalogName     string `json:"catalog_name"`
	OnlineStoreName string `json:"online_store_name"`
	SchemaName      string `json:"schema_name"`
	TableNamePrefix string `json:"table_name_prefix"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesStreamingMode struct {
	Mode string `json:"mode,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesTableTrigger struct {
}

type DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeatures struct {
	CronSchedule            string                                                                                   `json:"cron_schedule,omitempty"`
	CronScheduleTrigger     *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesCronScheduleTrigger `json:"cron_schedule_trigger,omitempty"`
	FeatureName             string                                                                                   `json:"feature_name,omitempty"`
	IsOnline                bool                                                                                     `json:"is_online,omitempty"`
	LastMaterializationTime string                                                                                   `json:"last_materialization_time,omitempty"`
	MaterializedFeatureId   string                                                                                   `json:"materialized_feature_id"`
	OfflineStoreConfig      *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesOfflineStoreConfig  `json:"offline_store_config,omitempty"`
	OnlineStoreConfig       *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesOnlineStoreConfig   `json:"online_store_config,omitempty"`
	PipelineScheduleState   string                                                                                   `json:"pipeline_schedule_state,omitempty"`
	ProviderConfig          *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesProviderConfig      `json:"provider_config,omitempty"`
	StreamingMode           *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesStreamingMode       `json:"streaming_mode,omitempty"`
	TableName               string                                                                                   `json:"table_name,omitempty"`
	TableTrigger            *DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeaturesTableTrigger        `json:"table_trigger,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeaturesProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceFeatureEngineeringMaterializedFeatures struct {
	FeatureName          string                                                                 `json:"feature_name,omitempty"`
	MaterializedFeatures []DataSourceFeatureEngineeringMaterializedFeaturesMaterializedFeatures `json:"materialized_features,omitempty"`
	PageSize             int                                                                    `json:"page_size,omitempty"`
	ProviderConfig       *DataSourceFeatureEngineeringMaterializedFeaturesProviderConfig        `json:"provider_config,omitempty"`
}
