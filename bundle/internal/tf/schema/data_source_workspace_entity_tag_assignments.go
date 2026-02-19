// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceEntityTagAssignmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWorkspaceEntityTagAssignmentsTagAssignmentsProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWorkspaceEntityTagAssignmentsTagAssignments struct {
	EntityId       string                                                               `json:"entity_id"`
	EntityType     string                                                               `json:"entity_type"`
	ProviderConfig *DataSourceWorkspaceEntityTagAssignmentsTagAssignmentsProviderConfig `json:"provider_config,omitempty"`
	TagKey         string                                                               `json:"tag_key"`
	TagValue       string                                                               `json:"tag_value,omitempty"`
}

type DataSourceWorkspaceEntityTagAssignments struct {
	EntityId       string                                                  `json:"entity_id"`
	EntityType     string                                                  `json:"entity_type"`
	PageSize       int                                                     `json:"page_size,omitempty"`
	ProviderConfig *DataSourceWorkspaceEntityTagAssignmentsProviderConfig  `json:"provider_config,omitempty"`
	TagAssignments []DataSourceWorkspaceEntityTagAssignmentsTagAssignments `json:"tag_assignments,omitempty"`
}
