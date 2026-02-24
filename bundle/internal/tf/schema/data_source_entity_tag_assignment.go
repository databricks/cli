// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceEntityTagAssignmentProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type DataSourceEntityTagAssignment struct {
	EntityName     string                                       `json:"entity_name"`
	EntityType     string                                       `json:"entity_type"`
	ProviderConfig *DataSourceEntityTagAssignmentProviderConfig `json:"provider_config,omitempty"`
	SourceType     string                                       `json:"source_type,omitempty"`
	TagKey         string                                       `json:"tag_key"`
	TagValue       string                                       `json:"tag_value,omitempty"`
	UpdateTime     string                                       `json:"update_time,omitempty"`
	UpdatedBy      string                                       `json:"updated_by,omitempty"`
}
