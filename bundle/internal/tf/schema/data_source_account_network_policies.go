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

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                       `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                       `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                                      `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                            `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                      `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                                      `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                                     `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                           `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                               `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngress struct {
	PublicAccess *DataSourceAccountNetworkPoliciesItemsIngressPublicAccess `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                             `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                             `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                                            `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                  `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                            `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                                            `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                                           `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                 `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                     `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRun struct {
	PublicAccess *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccess `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPoliciesItems struct {
	AccountId       string                                              `json:"account_id,omitempty"`
	Egress          *DataSourceAccountNetworkPoliciesItemsEgress        `json:"egress,omitempty"`
	Ingress         *DataSourceAccountNetworkPoliciesItemsIngress       `json:"ingress,omitempty"`
	IngressDryRun   *DataSourceAccountNetworkPoliciesItemsIngressDryRun `json:"ingress_dry_run,omitempty"`
	NetworkPolicyId string                                              `json:"network_policy_id"`
}

type DataSourceAccountNetworkPolicies struct {
	Items []DataSourceAccountNetworkPoliciesItems `json:"items,omitempty"`
}
