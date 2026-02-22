// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceGlobalInitScriptProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceGlobalInitScript struct {
	ContentBase64  string                                  `json:"content_base64,omitempty"`
	Enabled        bool                                    `json:"enabled,omitempty"`
	Id             string                                  `json:"id,omitempty"`
	Md5            string                                  `json:"md5,omitempty"`
	Name           string                                  `json:"name"`
	Position       int                                     `json:"position,omitempty"`
	Source         string                                  `json:"source,omitempty"`
	ProviderConfig *ResourceGlobalInitScriptProviderConfig `json:"provider_config,omitempty"`
}
