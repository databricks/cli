// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDomainIcon struct {
	Color string `json:"color,omitempty"`
	Name  string `json:"name,omitempty"`
}

type ResourceDomainProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceDomain struct {
	BusinessOwnerIds  []int                         `json:"business_owner_ids,omitempty"`
	CreateTime        string                        `json:"create_time,omitempty"`
	Description       string                        `json:"description,omitempty"`
	DomainId          string                        `json:"domain_id,omitempty"`
	Draft             bool                          `json:"draft,omitempty"`
	EffectiveDraft    bool                          `json:"effective_draft,omitempty"`
	Icon              *ResourceDomainIcon           `json:"icon,omitempty"`
	Name              string                        `json:"name,omitempty"`
	ParentDomainId    string                        `json:"parent_domain_id,omitempty"`
	ProviderConfig    *ResourceDomainProviderConfig `json:"provider_config,omitempty"`
	Subtitle          string                        `json:"subtitle,omitempty"`
	TagKey            string                        `json:"tag_key"`
	TechnicalOwnerIds []int                         `json:"technical_owner_ids,omitempty"`
	UpdateTime        string                        `json:"update_time,omitempty"`
}
