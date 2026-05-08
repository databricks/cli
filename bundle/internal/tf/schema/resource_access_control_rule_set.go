// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccessControlRuleSetGrantRules struct {
	Principals []string `json:"principals,omitempty"`
	Role       string   `json:"role"`
}

type ResourceAccessControlRuleSetProviderConfig struct {
	WorkspaceId string `json:"workspace_id"`
}

type ResourceAccessControlRuleSet struct {
	Api            string                                      `json:"api,omitempty"`
	Etag           string                                      `json:"etag,omitempty"`
	Id             string                                      `json:"id,omitempty"`
	Name           string                                      `json:"name"`
	GrantRules     []ResourceAccessControlRuleSetGrantRules    `json:"grant_rules,omitempty"`
	ProviderConfig *ResourceAccessControlRuleSetProviderConfig `json:"provider_config,omitempty"`
}
