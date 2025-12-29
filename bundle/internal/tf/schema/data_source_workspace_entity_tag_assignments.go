// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceEntityTagAssignmentsTagAssignments struct {
	EntityId   string `json:"entity_id"`
	EntityType string `json:"entity_type"`
	TagKey     string `json:"tag_key"`
	TagValue   string `json:"tag_value,omitempty"`
}

type DataSourceWorkspaceEntityTagAssignments struct {
	EntityId       string                                                  `json:"entity_id"`
	EntityType     string                                                  `json:"entity_type"`
	PageSize       int                                                     `json:"page_size,omitempty"`
	TagAssignments []DataSourceWorkspaceEntityTagAssignmentsTagAssignments `json:"tag_assignments,omitempty"`
}
