// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations struct {
	AzureStorageAccount    string `json:"azure_storage_account,omitempty"`
	AzureStorageService    string `json:"azure_storage_service,omitempty"`
	BucketName             string `json:"bucket_name,omitempty"`
	Region                 string `json:"region,omitempty"`
	StorageDestinationType string `json:"storage_destination_type,omitempty"`
}

type DataSourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type DataSourceAccountNetworkPolicyEgressNetworkAccess struct {
	AllowedInternetDestinations []DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	PolicyEnforcement           *DataSourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement            `json:"policy_enforcement,omitempty"`
	RestrictionMode             string                                                                         `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyEgress struct {
	NetworkAccess *DataSourceAccountNetworkPolicyEgressNetworkAccess `json:"network_access,omitempty"`
}

type DataSourceAccountNetworkPolicy struct {
	AccountId       string                                `json:"account_id,omitempty"`
	Egress          *DataSourceAccountNetworkPolicyEgress `json:"egress,omitempty"`
	NetworkPolicyId string                                `json:"network_policy_id,omitempty"`
}
