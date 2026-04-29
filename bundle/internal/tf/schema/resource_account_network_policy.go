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

type ResourceAccountNetworkPolicyEgressNetworkAccessBlockedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type ResourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type ResourceAccountNetworkPolicyEgressNetworkAccess struct {
	AllowedInternetDestinations []ResourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []ResourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	BlockedInternetDestinations []ResourceAccountNetworkPolicyEgressNetworkAccessBlockedInternetDestinations `json:"blocked_internet_destinations,omitempty"`
	PolicyEnforcement           *ResourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement            `json:"policy_enforcement,omitempty"`
	RestrictionMode             string                                                                       `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyEgress struct {
	NetworkAccess *ResourceAccountNetworkPolicyEgressNetworkAccess `json:"network_access,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                              `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                 `json:"all_destinations,omitempty"`
	AppsRuntime     *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime     `json:"apps_runtime,omitempty"`
	LakebaseRuntime *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime `json:"lakebase_runtime,omitempty"`
	WorkspaceApi    *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi    `json:"workspace_api,omitempty"`
	WorkspaceUi     *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi     `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                             `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                   `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                             `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                                `json:"all_destinations,omitempty"`
	AppsRuntime     *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime     `json:"apps_runtime,omitempty"`
	LakebaseRuntime *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime `json:"lakebase_runtime,omitempty"`
	WorkspaceApi    *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi    `json:"workspace_api,omitempty"`
	WorkspaceUi     *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi     `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                            `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                  `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccess struct {
	AllowRules      []ResourceAccountNetworkPolicyIngressPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []ResourceAccountNetworkPolicyIngressPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                      `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyIngress struct {
	PublicAccess *ResourceAccountNetworkPolicyIngressPublicAccess `json:"public_access,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                    `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination struct {
	AllDestinations bool                                                                                       `json:"all_destinations,omitempty"`
	AppsRuntime     *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime     `json:"apps_runtime,omitempty"`
	LakebaseRuntime *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime `json:"lakebase_runtime,omitempty"`
	WorkspaceApi    *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi    `json:"workspace_api,omitempty"`
	WorkspaceUi     *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi     `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOrigin struct {
	AllIpRanges      bool                                                                                   `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                         `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                   `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	Scopes []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination struct {
	AllDestinations bool                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime     *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime     `json:"apps_runtime,omitempty"`
	LakebaseRuntime *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime `json:"lakebase_runtime,omitempty"`
	WorkspaceApi    *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi    `json:"workspace_api,omitempty"`
	WorkspaceUi     *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi     `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges struct {
	IpRanges []string `json:"ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOrigin struct {
	AllIpRanges      bool                                                                                  `json:"all_ip_ranges,omitempty"`
	ExcludedIpRanges *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginExcludedIpRanges `json:"excluded_ip_ranges,omitempty"`
	IncludedIpRanges *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOriginIncludedIpRanges `json:"included_ip_ranges,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                        `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccess struct {
	AllowRules      []ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                            `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyIngressDryRun struct {
	PublicAccess *ResourceAccountNetworkPolicyIngressDryRunPublicAccess `json:"public_access,omitempty"`
}

type ResourceAccountNetworkPolicy struct {
	AccountId       string                                     `json:"account_id,omitempty"`
	Egress          *ResourceAccountNetworkPolicyEgress        `json:"egress,omitempty"`
	Ingress         *ResourceAccountNetworkPolicyIngress       `json:"ingress,omitempty"`
	IngressDryRun   *ResourceAccountNetworkPolicyIngressDryRun `json:"ingress_dry_run,omitempty"`
	NetworkPolicyId string                                     `json:"network_policy_id,omitempty"`
}
