// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceRecipientsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceRecipients struct {
	ProviderConfig *DataSourceRecipientsProviderConfig `json:"provider_config,omitempty"`
	Recipients     []string                            `json:"recipients,omitempty"`
}
