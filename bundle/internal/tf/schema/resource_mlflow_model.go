// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceMlflowModelTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ResourceMlflowModel struct {
	CreationTimestamp    int                       `json:"creation_timestamp,omitempty"`
	Description          string                    `json:"description,omitempty"`
	Id                   string                    `json:"id,omitempty"`
	LastUpdatedTimestamp int                       `json:"last_updated_timestamp,omitempty"`
	Name                 string                    `json:"name"`
	RegisteredModelId    string                    `json:"registered_model_id,omitempty"`
	UserId               string                    `json:"user_id,omitempty"`
	Tags                 []ResourceMlflowModelTags `json:"tags,omitempty"`
}
