// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type ResourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations struct {
	AzureStorageAccount    string `json:"azure_storage_account,omitempty"`
	AzureStorageService    string `json:"azure_storage_service,omitempty"`
	BucketName             string `json:"bucket_name,omitempty"`
	Region                 string `json:"region,omitempty"`
	StorageDestinationType string `json:"storage_destination_type,omitempty"`
}

type ResourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type ResourceAccountNetworkPolicyEgressNetworkAccess struct {
	AllowedInternetDestinations []ResourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []ResourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	PolicyEnforcement           *ResourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement            `json:"policy_enforcement,omitempty"`
	RestrictionMode             string                                                                       `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyEgress struct {
	NetworkAccess *ResourceAccountNetworkPolicyEgressNetworkAccess `json:"network_access,omitempty"`
}

type ResourceAccountNetworkPolicy struct {
	AccountId       string                              `json:"account_id,omitempty"`
	Egress          *ResourceAccountNetworkPolicyEgress `json:"egress,omitempty"`
	NetworkPolicyId string                              `json:"network_policy_id,omitempty"`
}
