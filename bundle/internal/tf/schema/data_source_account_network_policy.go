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

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                               `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                     `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                               `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                               `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                              `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                    `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                        `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngress struct {
	PublicAccess *DataSourceAccountNetworkPolicyIngressPublicAccess `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                      `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                      `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                                     `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                           `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                     `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                                     `json:"all_destinations,omitempty"`
	WorkspaceApi    *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi `json:"workspace_api,omitempty"`
	WorkspaceUi     *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi  `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                                    `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                          `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                              `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngressDryRun struct {
	PublicAccess *DataSourceAccountNetworkPolicyIngressDryRunPublicAccess `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPolicy struct {
	AccountId       string                                       `json:"account_id,omitempty"`
	Egress          *DataSourceAccountNetworkPolicyEgress        `json:"egress,omitempty"`
	Ingress         *DataSourceAccountNetworkPolicyIngress       `json:"ingress,omitempty"`
	IngressDryRun   *DataSourceAccountNetworkPolicyIngressDryRun `json:"ingress_dry_run,omitempty"`
	NetworkPolicyId string                                       `json:"network_policy_id"`
}
