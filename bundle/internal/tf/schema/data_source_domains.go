// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDomainsDomainsIcon struct {
	Color string `json:"color,omitempty"`
	Name  string `json:"name,omitempty"`
}

type DataSourceDomainsDomainsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceDomainsDomains struct {
	BusinessOwnerIds  []int                                   `json:"business_owner_ids,omitempty"`
	CreateTime        string                                  `json:"create_time,omitempty"`
	Description       string                                  `json:"description,omitempty"`
	DomainId          string                                  `json:"domain_id,omitempty"`
	Draft             bool                                    `json:"draft,omitempty"`
	EffectiveDraft    bool                                    `json:"effective_draft,omitempty"`
	Icon              *DataSourceDomainsDomainsIcon           `json:"icon,omitempty"`
	Name              string                                  `json:"name"`
	ParentDomainId    string                                  `json:"parent_domain_id,omitempty"`
	ProviderConfig    *DataSourceDomainsDomainsProviderConfig `json:"provider_config,omitempty"`
	Subtitle          string                                  `json:"subtitle,omitempty"`
	TagKey            string                                  `json:"tag_key,omitempty"`
	TechnicalOwnerIds []int                                   `json:"technical_owner_ids,omitempty"`
	UpdateTime        string                                  `json:"update_time,omitempty"`
}

type DataSourceDomainsProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type DataSourceDomains struct {
	Domains        []DataSourceDomainsDomains       `json:"domains,omitempty"`
	PageSize       int                              `json:"page_size,omitempty"`
	ParentDomainId string                           `json:"parent_domain_id,omitempty"`
	ProviderConfig *DataSourceDomainsProviderConfig `json:"provider_config,omitempty"`
}
