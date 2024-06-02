// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMlflowExperimentTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourceMlflowExperiment struct {
	ArtifactLocation string                           `json:"artifact_location,omitempty"`
	CreationTime     int                              `json:"creation_time,omitempty"`
	ExperimentId     string                           `json:"experiment_id,omitempty"`
	Id               string                           `json:"id,omitempty"`
	LastUpdateTime   int                              `json:"last_update_time,omitempty"`
	LifecycleStage   string                           `json:"lifecycle_stage,omitempty"`
	Name             string                           `json:"name,omitempty"`
	Tags             []DataSourceMlflowExperimentTags `json:"tags,omitempty"`
}
