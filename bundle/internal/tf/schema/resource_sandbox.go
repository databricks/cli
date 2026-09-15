// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceSandboxProviderConfig struct {
	WorkspaceId string `json:"workspace_id,omitempty"`
}

type ResourceSandboxSpecCompute struct {
	InactivityTimeout string `json:"inactivity_timeout,omitempty"`
}

type ResourceSandboxSpec struct {
	Compute *ResourceSandboxSpecCompute `json:"compute,omitempty"`
}

type ResourceSandboxStatus struct {
	State string `json:"state,omitempty"`
}

type ResourceSandbox struct {
	CreateTime     string                         `json:"create_time,omitempty"`
	DisplayName    string                         `json:"display_name,omitempty"`
	Name           string                         `json:"name,omitempty"`
	ProviderConfig *ResourceSandboxProviderConfig `json:"provider_config,omitempty"`
	SandboxId      string                         `json:"sandbox_id"`
	Spec           *ResourceSandboxSpec           `json:"spec,omitempty"`
	Status         *ResourceSandboxStatus         `json:"status,omitempty"`
	UpdateTime     string                         `json:"update_time,omitempty"`
}
