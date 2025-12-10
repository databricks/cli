// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceEntityTagAssignment struct {
	EntityId   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	TagKey     string `json:"tag_key"`
	TagValue   string `json:"tag_value,omitempty"`
}
