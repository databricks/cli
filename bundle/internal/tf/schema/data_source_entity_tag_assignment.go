// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEntityTagAssignment struct {
	EntityName string `json:"entity_name"`
	EntityType string `json:"entity_type"`
	TagKey     string `json:"tag_key"`
	TagValue   string `json:"tag_value,omitempty"`
}
