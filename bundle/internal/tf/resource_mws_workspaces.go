// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package tf

type ResourceMwsWorkspacesCloudResourceBucketGcp struct {
	ProjectId string `json:"project_id"`
}

type ResourceMwsWorkspacesCloudResourceBucket struct {
	Gcp *ResourceMwsWorkspacesCloudResourceBucketGcp `json:"gcp,omitempty"`
}

type ResourceMwsWorkspacesExternalCustomerInfo struct {
	AuthoritativeUserEmail    string `json:"authoritative_user_email"`
	AuthoritativeUserFullName string `json:"authoritative_user_full_name"`
	CustomerName              string `json:"customer_name"`
}

type ResourceMwsWorkspacesNetworkGcpCommonNetworkConfig struct {
	GkeClusterMasterIpRange string `json:"gke_cluster_master_ip_range"`
	GkeConnectivityType     string `json:"gke_connectivity_type"`
}

type ResourceMwsWorkspacesNetworkGcpManagedNetworkConfig struct {
	GkeClusterPodIpRange     string `json:"gke_cluster_pod_ip_range"`
	GkeClusterServiceIpRange string `json:"gke_cluster_service_ip_range"`
	SubnetCidr               string `json:"subnet_cidr"`
}

type ResourceMwsWorkspacesNetwork struct {
	NetworkId               string                                               `json:"network_id,omitempty"`
	GcpCommonNetworkConfig  *ResourceMwsWorkspacesNetworkGcpCommonNetworkConfig  `json:"gcp_common_network_config,omitempty"`
	GcpManagedNetworkConfig *ResourceMwsWorkspacesNetworkGcpManagedNetworkConfig `json:"gcp_managed_network_config,omitempty"`
}

type ResourceMwsWorkspacesToken struct {
	Comment         string `json:"comment,omitempty"`
	LifetimeSeconds int    `json:"lifetime_seconds,omitempty"`
	TokenId         string `json:"token_id,omitempty"`
	TokenValue      string `json:"token_value,omitempty"`
}

type ResourceMwsWorkspaces struct {
	AccountId                           string                                     `json:"account_id"`
	AwsRegion                           string                                     `json:"aws_region,omitempty"`
	Cloud                               string                                     `json:"cloud,omitempty"`
	CreationTime                        int                                        `json:"creation_time,omitempty"`
	CredentialsId                       string                                     `json:"credentials_id,omitempty"`
	CustomerManagedKeyId                string                                     `json:"customer_managed_key_id,omitempty"`
	DeploymentName                      string                                     `json:"deployment_name,omitempty"`
	Id                                  string                                     `json:"id,omitempty"`
	IsNoPublicIpEnabled                 bool                                       `json:"is_no_public_ip_enabled,omitempty"`
	Location                            string                                     `json:"location,omitempty"`
	ManagedServicesCustomerManagedKeyId string                                     `json:"managed_services_customer_managed_key_id,omitempty"`
	NetworkId                           string                                     `json:"network_id,omitempty"`
	PricingTier                         string                                     `json:"pricing_tier,omitempty"`
	PrivateAccessSettingsId             string                                     `json:"private_access_settings_id,omitempty"`
	StorageConfigurationId              string                                     `json:"storage_configuration_id,omitempty"`
	StorageCustomerManagedKeyId         string                                     `json:"storage_customer_managed_key_id,omitempty"`
	WorkspaceId                         int                                        `json:"workspace_id,omitempty"`
	WorkspaceName                       string                                     `json:"workspace_name"`
	WorkspaceStatus                     string                                     `json:"workspace_status,omitempty"`
	WorkspaceStatusMessage              string                                     `json:"workspace_status_message,omitempty"`
	WorkspaceUrl                        string                                     `json:"workspace_url,omitempty"`
	CloudResourceBucket                 *ResourceMwsWorkspacesCloudResourceBucket  `json:"cloud_resource_bucket,omitempty"`
	ExternalCustomerInfo                *ResourceMwsWorkspacesExternalCustomerInfo `json:"external_customer_info,omitempty"`
	Network                             *ResourceMwsWorkspacesNetwork              `json:"network,omitempty"`
	Token                               *ResourceMwsWorkspacesToken                `json:"token,omitempty"`
}
