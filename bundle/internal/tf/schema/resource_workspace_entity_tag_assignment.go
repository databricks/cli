// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceWorkspaceEntityTagAssignmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceWorkspaceEntityTagAssignment struct {
	EntityId       string                                              `json:"entity_id"`
	EntityType     string                                              `json:"entity_type"`
	ProviderConfig *ResourceWorkspaceEntityTagAssignmentProviderConfig `json:"provider_config,omitempty"`
	TagKey         string                                              `json:"tag_key"`
	TagValue       string                                              `json:"tag_value,omitempty"`
}
