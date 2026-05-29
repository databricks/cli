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

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                               `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                       `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                       `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                       `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                       `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessAllowRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                    `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                              `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                      `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                      `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                      `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccessDenyRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                   `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPrivateAccess struct {
	AllowRules      []ResourceAccountNetworkPolicyIngressPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []ResourceAccountNetworkPolicyIngressPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                       `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                              `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                     `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	PrivateAccess *ResourceAccountNetworkPolicyIngressPrivateAccess `json:"private_access,omitempty"`
	PublicAccess  *ResourceAccountNetworkPolicyIngressPublicAccess  `json:"public_access,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                     `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                             `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                             `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                             `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                             `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                          `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                    `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                            `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                            `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                            `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                            `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRules struct {
	Authentication *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                         `json:"label,omitempty"`
	Origin         *ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPrivateAccess struct {
	AllowRules      []ResourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []ResourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                             `json:"restriction_mode"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                    `json:"identity_type,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                            `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination struct {
	AccountApi           *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                           `json:"all_destinations,omitempty"`
	AppsRuntime          *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *ResourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	PrivateAccess *ResourceAccountNetworkPolicyIngressDryRunPrivateAccess `json:"private_access,omitempty"`
	PublicAccess  *ResourceAccountNetworkPolicyIngressDryRunPublicAccess  `json:"public_access,omitempty"`
}

type ResourceAccountNetworkPolicy struct {
	AccountId       string                                     `json:"account_id,omitempty"`
	Egress          *ResourceAccountNetworkPolicyEgress        `json:"egress,omitempty"`
	Ingress         *ResourceAccountNetworkPolicyIngress       `json:"ingress,omitempty"`
	IngressDryRun   *ResourceAccountNetworkPolicyIngressDryRun `json:"ingress_dry_run,omitempty"`
	NetworkPolicyId string                                     `json:"network_policy_id,omitempty"`
}
