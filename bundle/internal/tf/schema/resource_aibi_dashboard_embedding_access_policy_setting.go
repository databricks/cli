// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAibiDashboardEmbeddingAccessPolicySettingAibiDashboardEmbeddingAccessPolicy struct {
	AccessPolicyType string `json:"access_policy_type"`
}

type ResourceAibiDashboardEmbeddingAccessPolicySetting struct {
	Etag                               string                                                                               `json:"etag,omitempty"`
	Id                                 string                                                                               `json:"id,omitempty"`
	SettingName                        string                                                                               `json:"setting_name,omitempty"`
	AibiDashboardEmbeddingAccessPolicy *ResourceAibiDashboardEmbeddingAccessPolicySettingAibiDashboardEmbeddingAccessPolicy `json:"aibi_dashboard_embedding_access_policy,omitempty"`
}
