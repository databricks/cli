// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedDatabricksDestinations struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

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

type DataSourceAccountNetworkPolicyEgressNetworkAccessBlockedInternetDestinations struct {
	Destination             string `json:"destination,omitempty"`
	InternetDestinationType string `json:"internet_destination_type,omitempty"`
}

type DataSourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement struct {
	DryRunModeProductFilter []string `json:"dry_run_mode_product_filter,omitempty"`
	EnforcementMode         string   `json:"enforcement_mode,omitempty"`
}

type DataSourceAccountNetworkPolicyEgressNetworkAccess struct {
	AllowedDatabricksDestinations []DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedDatabricksDestinations `json:"allowed_databricks_destinations,omitempty"`
	AllowedInternetDestinations   []DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedInternetDestinations   `json:"allowed_internet_destinations,omitempty"`
	AllowedStorageDestinations    []DataSourceAccountNetworkPolicyEgressNetworkAccessAllowedStorageDestinations    `json:"allowed_storage_destinations,omitempty"`
	BlockedInternetDestinations   []DataSourceAccountNetworkPolicyEgressNetworkAccessBlockedInternetDestinations   `json:"blocked_internet_destinations,omitempty"`
	PolicyEnforcement             *DataSourceAccountNetworkPolicyEgressNetworkAccessPolicyEnforcement              `json:"policy_enforcement,omitempty"`
	RestrictionMode               string                                                                           `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyEgress struct {
	NetworkAccess *DataSourceAccountNetworkPolicyEgressNetworkAccess `json:"network_access,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                        `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                         `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                             `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                       `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                               `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                        `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                            `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                 `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                         `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                         `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                         `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                         `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                      `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                        `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                        `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                        `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                        `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                     `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPrivateAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                         `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                        `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                       `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	CrossWorkspaceAccess *DataSourceAccountNetworkPolicyIngressCrossWorkspaceAccess `json:"cross_workspace_access,omitempty"`
	PrivateAccess        *DataSourceAccountNetworkPolicyIngressPrivateAccess        `json:"private_access,omitempty"`
	PublicAccess         *DataSourceAccountNetworkPolicyIngressPublicAccess         `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                              `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                      `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                               `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                   `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                             `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                                     `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces struct {
	WorkspaceIds []int `json:"workspace_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesOrigin struct {
	AllSourceWorkspaces bool                                                                                              `json:"all_source_workspaces,omitempty"`
	SelectedWorkspaces  *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesOriginSelectedWorkspaces `json:"selected_workspaces,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                                  `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                                      `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                       `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                               `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOrigin struct {
	AllPrivateAccess          bool                                                                               `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                               `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                               `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                            `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                      `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                              `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOriginEndpoints struct {
	EndpointIds []string `json:"endpoint_ids,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOrigin struct {
	AllPrivateAccess          bool                                                                              `json:"all_private_access,omitempty"`
	AllRegisteredEndpoints    bool                                                                              `json:"all_registered_endpoints,omitempty"`
	AzureWorkspacePrivateLink bool                                                                              `json:"azure_workspace_private_link,omitempty"`
	Endpoints                 *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOriginEndpoints `json:"endpoints,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRules struct {
	Authentication *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesAuthentication `json:"authentication,omitempty"`
	Destination    *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesDestination    `json:"destination,omitempty"`
	Label          string                                                                           `json:"label,omitempty"`
	Origin         *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRulesOrigin         `json:"origin,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPrivateAccess struct {
	AllowRules      []DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessAllowRules `json:"allow_rules,omitempty"`
	DenyRules       []DataSourceAccountNetworkPolicyIngressDryRunPrivateAccessDenyRules  `json:"deny_rules,omitempty"`
	RestrictionMode string                                                               `json:"restriction_mode"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities struct {
	PrincipalId   int    `json:"principal_id,omitempty"`
	PrincipalType string `json:"principal_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthentication struct {
	Identities   []DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesAuthenticationIdentities `json:"identities,omitempty"`
	IdentityType string                                                                                      `json:"identity_type,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                              `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessAllowRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi struct {
	ScopeQualifier string   `json:"scope_qualifier,omitempty"`
	Scopes         []string `json:"scopes,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi struct {
	AllDestinations bool `json:"all_destinations,omitempty"`
}

type DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestination struct {
	AccountApi           *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountApi           `json:"account_api,omitempty"`
	AccountDatabricksOne *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountDatabricksOne `json:"account_databricks_one,omitempty"`
	AccountUi            *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAccountUi            `json:"account_ui,omitempty"`
	AllDestinations      bool                                                                                             `json:"all_destinations,omitempty"`
	AppsRuntime          *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationAppsRuntime          `json:"apps_runtime,omitempty"`
	LakebaseRuntime      *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationLakebaseRuntime      `json:"lakebase_runtime,omitempty"`
	WorkspaceApi         *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceApi         `json:"workspace_api,omitempty"`
	WorkspaceUi          *DataSourceAccountNetworkPolicyIngressDryRunPublicAccessDenyRulesDestinationWorkspaceUi          `json:"workspace_ui,omitempty"`
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
	CrossWorkspaceAccess *DataSourceAccountNetworkPolicyIngressDryRunCrossWorkspaceAccess `json:"cross_workspace_access,omitempty"`
	PrivateAccess        *DataSourceAccountNetworkPolicyIngressDryRunPrivateAccess        `json:"private_access,omitempty"`
	PublicAccess         *DataSourceAccountNetworkPolicyIngressDryRunPublicAccess         `json:"public_access,omitempty"`
}

type DataSourceAccountNetworkPolicy struct {
	AccountId       string                                       `json:"account_id,omitempty"`
	Egress          *DataSourceAccountNetworkPolicyEgress        `json:"egress,omitempty"`
	Ingress         *DataSourceAccountNetworkPolicyIngress       `json:"ingress,omitempty"`
	IngressDryRun   *DataSourceAccountNetworkPolicyIngressDryRun `json:"ingress_dry_run,omitempty"`
	NetworkPolicyId string                                       `json:"network_policy_id"`
}
