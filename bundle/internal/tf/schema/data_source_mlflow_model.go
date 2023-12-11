// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceMlflowModelLatestVersionsTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourceMlflowModelLatestVersions struct {
	CreationTimestamp    int                                       `json:"creation_timestamp,omitempty"`
	CurrentStage         string                                    `json:"current_stage,omitempty"`
	Description          string                                    `json:"description,omitempty"`
	LastUpdatedTimestamp int                                       `json:"last_updated_timestamp,omitempty"`
	Name                 string                                    `json:"name,omitempty"`
	RunId                string                                    `json:"run_id,omitempty"`
	RunLink              string                                    `json:"run_link,omitempty"`
	Source               string                                    `json:"source,omitempty"`
	Status               string                                    `json:"status,omitempty"`
	StatusMessage        string                                    `json:"status_message,omitempty"`
	UserId               string                                    `json:"user_id,omitempty"`
	Version              string                                    `json:"version,omitempty"`
	Tags                 []DataSourceMlflowModelLatestVersionsTags `json:"tags,omitempty"`
}

type DataSourceMlflowModelTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type DataSourceMlflowModel struct {
	Description     string                                `json:"description,omitempty"`
	Id              string                                `json:"id,omitempty"`
	Name            string                                `json:"name"`
	PermissionLevel string                                `json:"permission_level,omitempty"`
	UserId          string                                `json:"user_id,omitempty"`
	LatestVersions  []DataSourceMlflowModelLatestVersions `json:"latest_versions,omitempty"`
	Tags            []DataSourceMlflowModelTags           `json:"tags,omitempty"`
}
