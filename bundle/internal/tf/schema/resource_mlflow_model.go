// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMlflowModelTags struct {
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

type ResourceMlflowModel struct {
	Description       string                    `json:"description,omitempty"`
	Id                string                    `json:"id,omitempty"`
	Name              string                    `json:"name"`
	RegisteredModelId string                    `json:"registered_model_id,omitempty"`
	Tags              []ResourceMlflowModelTags `json:"tags,omitempty"`
}
