// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedStorageDestinations struct {
	AzureStorageAccount    string `json:"azure_storage_account,omitempty"`
	AzureStorageService    string `json:"azure_storage_service,omitempty"`
	BucketName             string `json:"bucket_name,omitempty"`
	Region                 string `json:"region,omitempty"`
	StorageDestinationType string `json:"storage_destination_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccess struct {
	AllowedInternetDestinations []DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	PolicyEnforcement           *DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessPolicyEnforcement            `json:"policy_enforcement,omitempty"`
	RestrictionMode             string                                                                                `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsEgress struct {
	NetworkAccess *DataSourceAccountNetworkPoliciesItemsEgressNetworkAccess `json:"network_access,omitempty"`
}

type DataSourceAccountNetworkPoliciesItems struct {
	AccountId       string                                       `json:"account_id,omitempty"`
	Egress          *DataSourceAccountNetworkPoliciesItemsEgress `json:"egress,omitempty"`
	NetworkPolicyId string                                       `json:"network_policy_id,omitempty"`
}

type DataSourceAccountNetworkPolicies struct {
	Items []DataSourceAccountNetworkPoliciesItems `json:"items,omitempty"`
}
