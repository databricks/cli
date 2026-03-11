// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceWorkspaceEntityTagAssignmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceWorkspaceEntityTagAssignment struct {
	EntityId       string                                                `json:"entity_id"`
	EntityType     string                                                `json:"entity_type"`
	ProviderConfig *DataSourceWorkspaceEntityTagAssignmentProviderConfig `json:"provider_config,omitempty"`
	TagKey         string                                                `json:"tag_key"`
	TagValue       string                                                `json:"tag_value,omitempty"`
}
