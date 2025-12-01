// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEntityTagAssignmentsTagAssignments struct {
	EntityName string `json:"entity_name"`
	EntityType string `json:"entity_type"`
	TagKey     string `json:"tag_key"`
	TagValue   string `json:"tag_value,omitempty"`
}

type DataSourceEntityTagAssignments struct {
	EntityName     string                                         `json:"entity_name"`
	EntityType     string                                         `json:"entity_type"`
	MaxResults     int                                            `json:"max_results,omitempty"`
	TagAssignments []DataSourceEntityTagAssignmentsTagAssignments `json:"tag_assignments,omitempty"`
}
