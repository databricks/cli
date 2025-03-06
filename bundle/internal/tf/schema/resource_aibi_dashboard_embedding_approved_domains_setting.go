// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAibiDashboardEmbeddingApprovedDomainsSettingAibiDashboardEmbeddingApprovedDomains struct {
	ApprovedDomains []string `json:"approved_domains"`
}

type ResourceAibiDashboardEmbeddingApprovedDomainsSetting struct {
	Etag                                  string                                                                                     `json:"etag,omitempty"`
	Id                                    string                                                                                     `json:"id,omitempty"`
	SettingName                           string                                                                                     `json:"setting_name,omitempty"`
	AibiDashboardEmbeddingApprovedDomains *ResourceAibiDashboardEmbeddingApprovedDomainsSettingAibiDashboardEmbeddingApprovedDomains `json:"aibi_dashboard_embedding_approved_domains,omitempty"`
}
