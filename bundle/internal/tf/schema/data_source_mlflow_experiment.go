// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMlflowExperimentProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceMlflowExperimentTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourceMlflowExperimentTraceLocationUcTraceLocation struct {
	Catalog              string `json:"catalog"`
	EffectiveTablePrefix string `json:"effective_table_prefix,omitempty"`
	Schema               string `json:"schema"`
	TablePrefix          string `json:"table_prefix,omitempty"`
}

type DataSourceMlflowExperimentTraceLocation struct {
	UcTraceLocation *DataSourceMlflowExperimentTraceLocationUcTraceLocation `json:"uc_trace_location,omitempty"`
}

type DataSourceMlflowExperiment struct {
	ArtifactLocation string                                    `json:"artifact_location,omitempty"`
	CreationTime     int                                       `json:"creation_time,omitempty"`
	ExperimentId     string                                    `json:"experiment_id,omitempty"`
	Id               string                                    `json:"id,omitempty"`
	LastUpdateTime   int                                       `json:"last_update_time,omitempty"`
	LifecycleStage   string                                    `json:"lifecycle_stage,omitempty"`
	Name             string                                    `json:"name,omitempty"`
	ProviderConfig   *DataSourceMlflowExperimentProviderConfig `json:"provider_config,omitempty"`
	Tags             []DataSourceMlflowExperimentTags          `json:"tags,omitempty"`
	TraceLocation    *DataSourceMlflowExperimentTraceLocation  `json:"trace_location,omitempty"`
}
