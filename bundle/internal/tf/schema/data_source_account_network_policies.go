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

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessBlockedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsEgressNetworkAccess struct {
	AllowedInternetDestinations []DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedInternetDestinations `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations  []DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessAllowedStorageDestinations  `json:"allowed_storage_destinations,omitempty"`
	BlockedInternetDestinations []DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessBlockedInternetDestinations `json:"blocked_internet_destinations,omitempty"`
	PolicyEnforcement           *DataSourceAccountNetworkPoliciesItemsEgressNetworkAccessPolicyEnforcement            `json:"policy_enforcement,omitempty"`
	RestrictionMode             string                                                                                `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsEgress struct {
	NetworkAccess *DataSourceAccountNetworkPoliciesItemsEgressNetworkAccess `json:"network_access,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                               `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                       `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                                `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                    `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                              `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                               `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                   `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                       `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                        `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                                `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                                `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                                `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                             `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                       `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                               `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                               `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                               `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                               `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                            `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPrivateAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                       `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                               `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                              `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	CrossWorkspaceAccess *DataSourceAccountNetworkPoliciesItemsIngressCrossWorkspaceAccess `json:"cross_workspace_access,omitempty"`
	PrivateAccess        *DataSourceAccountNetworkPoliciesItemsIngressPrivateAccess        `json:"private_access,omitempty"`
	PublicAccess         *DataSourceAccountNetworkPoliciesItemsIngressPublicAccess         `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                                     `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                             `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                                      `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                          `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                                    `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                            `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                                     `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                         `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                             `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                              `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                                      `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                                      `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                                      `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                   `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                             `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                     `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                                     `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                                     `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                                     `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                  `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccess struct {
	AllowRules      []DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                      `json:"restriction_mode"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                             `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                     `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                    `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	CrossWorkspaceAccess *DataSourceAccountNetworkPoliciesItemsIngressDryRunCrossWorkspaceAccess `json:"cross_workspace_access,omitempty"`
	PrivateAccess        *DataSourceAccountNetworkPoliciesItemsIngressDryRunPrivateAccess        `json:"private_access,omitempty"`
	PublicAccess         *DataSourceAccountNetworkPoliciesItemsIngressDryRunPublicAccess         `json:"public_access,omitempty"`
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
