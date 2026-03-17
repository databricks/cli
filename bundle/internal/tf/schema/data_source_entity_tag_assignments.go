// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEntityTagAssignmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceEntityTagAssignmentsTagAssignmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceEntityTagAssignmentsTagAssignments struct {
	EntityName     string                                                      `json:"entity_name"`
	EntityType     string                                                      `json:"entity_type"`
	ProviderConfig *DataSourceEntityTagAssignmentsTagAssignmentsProviderConfig `json:"provider_config,omitempty"`
	SourceType     string                                                      `json:"source_type,omitempty"`
	TagKey         string                                                      `json:"tag_key"`
	TagValue       string                                                      `json:"tag_value,omitempty"`
	UpdateTime     string                                                      `json:"update_time,omitempty"`
	UpdatedBy      string                                                      `json:"updated_by,omitempty"`
}

type DataSourceEntityTagAssignments struct {
	EntityName     string                                         `json:"entity_name"`
	EntityType     string                                         `json:"entity_type"`
	MaxResults     int                                            `json:"max_results,omitempty"`
	ProviderConfig *DataSourceEntityTagAssignmentsProviderConfig  `json:"provider_config,omitempty"`
	TagAssignments []DataSourceEntityTagAssignmentsTagAssignments `json:"tag_assignments,omitempty"`
}
