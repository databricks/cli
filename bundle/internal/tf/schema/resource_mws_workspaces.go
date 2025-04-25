// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceMwsWorkspacesCloudResourceContainerGcp struct {
	ProjectId string `json:"project_id"`
}

type ResourceMwsWorkspacesCloudResourceContainer struct {
	Gcp *ResourceMwsWorkspacesCloudResourceContainerGcp `json:"gcp,omitempty"`
}

type ResourceMwsWorkspacesExternalCustomerInfo struct {
	AuthoritativeUserEmail    string `json:"authoritative_user_email"`
	AuthoritativeUserFullName string `json:"authoritative_user_full_name"`
	CustomerName              string `json:"customer_name"`
}

type ResourceMwsWorkspacesGcpManagedNetworkConfig struct {
	GkeClusterPodIpRange     string `json:"gke_cluster_pod_ip_range,omitempty"`
	GkeClusterServiceIpRange string `json:"gke_cluster_service_ip_range,omitempty"`
	SubnetCidr               string `json:"subnet_cidr"`
}

type ResourceMwsWorkspacesGkeConfig struct {
	ConnectivityType string `json:"connectivity_type,omitempty"`
	MasterIpRange    string `json:"master_ip_range,omitempty"`
}

type ResourceMwsWorkspacesToken struct {
	Comment         string `json:"comment,omitempty"`
	LifetimeSeconds int    `json:"lifetime_seconds,omitempty"`
	TokenId         string `json:"token_id,omitempty"`
	TokenValue      string `json:"token_value,omitempty"`
}

type ResourceMwsWorkspaces struct {
	AccountId                           string                                        `json:"account_id"`
	AwsRegion                           string                                        `json:"aws_region,omitempty"`
	Cloud                               string                                        `json:"cloud,omitempty"`
	ComputeMode                         string                                        `json:"compute_mode,omitempty"`
	CreationTime                        int                                           `json:"creation_time,omitempty"`
	CredentialsId                       string                                        `json:"credentials_id,omitempty"`
	CustomTags                          map[string]string                             `json:"custom_tags,omitempty"`
	CustomerManagedKeyId                string                                        `json:"customer_managed_key_id,omitempty"`
	DeploymentName                      string                                        `json:"deployment_name,omitempty"`
	EffectiveComputeMode                string                                        `json:"effective_compute_mode,omitempty"`
	GcpWorkspaceSa                      string                                        `json:"gcp_workspace_sa,omitempty"`
	Id                                  string                                        `json:"id,omitempty"`
	IsNoPublicIpEnabled                 bool                                          `json:"is_no_public_ip_enabled,omitempty"`
	Location                            string                                        `json:"location,omitempty"`
	ManagedServicesCustomerManagedKeyId string                                        `json:"managed_services_customer_managed_key_id,omitempty"`
	NetworkId                           string                                        `json:"network_id,omitempty"`
	PricingTier                         string                                        `json:"pricing_tier,omitempty"`
	PrivateAccessSettingsId             string                                        `json:"private_access_settings_id,omitempty"`
	StorageConfigurationId              string                                        `json:"storage_configuration_id,omitempty"`
	StorageCustomerManagedKeyId         string                                        `json:"storage_customer_managed_key_id,omitempty"`
	WorkspaceId                         int                                           `json:"workspace_id,omitempty"`
	WorkspaceName                       string                                        `json:"workspace_name"`
	WorkspaceStatus                     string                                        `json:"workspace_status,omitempty"`
	WorkspaceStatusMessage              string                                        `json:"workspace_status_message,omitempty"`
	WorkspaceUrl                        string                                        `json:"workspace_url,omitempty"`
	CloudResourceContainer              *ResourceMwsWorkspacesCloudResourceContainer  `json:"cloud_resource_container,omitempty"`
	ExternalCustomerInfo                *ResourceMwsWorkspacesExternalCustomerInfo    `json:"external_customer_info,omitempty"`
	GcpManagedNetworkConfig             *ResourceMwsWorkspacesGcpManagedNetworkConfig `json:"gcp_managed_network_config,omitempty"`
	GkeConfig                           *ResourceMwsWorkspacesGkeConfig               `json:"gke_config,omitempty"`
	Token                               *ResourceMwsWorkspacesToken                   `json:"token,omitempty"`
}
