// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceFeatureEngineeringMaterializedFeatureOfflineStoreConfig struct {
	CatalogName     string `json:"catalog_name"`
	SchemaName      string `json:"schema_name"`
	TableNamePrefix string `json:"table_name_prefix"`
}

type ResourceFeatureEngineeringMaterializedFeatureOnlineStoreConfig struct {
	Capacity         string `json:"capacity"`
	CreationTime     string `json:"creation_time,omitempty"`
	Creator          string `json:"creator,omitempty"`
	Name             string `json:"name"`
	ReadReplicaCount int    `json:"read_replica_count,omitempty"`
	State            string `json:"state,omitempty"`
}

type ResourceFeatureEngineeringMaterializedFeature struct {
	FeatureName             string                                                           `json:"feature_name"`
	LastMaterializationTime string                                                           `json:"last_materialization_time,omitempty"`
	MaterializedFeatureId   string                                                           `json:"materialized_feature_id,omitempty"`
	OfflineStoreConfig      *ResourceFeatureEngineeringMaterializedFeatureOfflineStoreConfig `json:"offline_store_config,omitempty"`
	OnlineStoreConfig       *ResourceFeatureEngineeringMaterializedFeatureOnlineStoreConfig  `json:"online_store_config,omitempty"`
	PipelineScheduleState   string                                                           `json:"pipeline_schedule_state,omitempty"`
	TableName               string                                                           `json:"table_name,omitempty"`
}
