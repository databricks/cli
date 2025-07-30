// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type Resources struct {
	AccountNetworkPolicy                         map[string]any `json:"databricks_account_network_policy,omitempty"`
	AccountSetting                               map[string]any `json:"databricks_account_setting,omitempty"`
	AibiDashboardEmbeddingAccessPolicySetting    map[string]any `json:"databricks_aibi_dashboard_embedding_access_policy_setting,omitempty"`
	AibiDashboardEmbeddingApprovedDomainsSetting map[string]any `json:"databricks_aibi_dashboard_embedding_approved_domains_setting,omitempty"`
	AlertV2                                      map[string]any `json:"databricks_alert_v2,omitempty"`
	AutomaticClusterUpdateWorkspaceSetting       map[string]any `json:"databricks_automatic_cluster_update_workspace_setting,omitempty"`
	BudgetPolicy                                 map[string]any `json:"databricks_budget_policy,omitempty"`
	CleanRoomAsset                               map[string]any `json:"databricks_clean_room_asset,omitempty"`
	CleanRoomAutoApprovalRule                    map[string]any `json:"databricks_clean_room_auto_approval_rule,omitempty"`
	CleanRoomsCleanRoom                          map[string]any `json:"databricks_clean_rooms_clean_room,omitempty"`
	ComplianceSecurityProfileWorkspaceSetting    map[string]any `json:"databricks_compliance_security_profile_workspace_setting,omitempty"`
	DatabaseDatabaseCatalog                      map[string]any `json:"databricks_database_database_catalog,omitempty"`
	DatabaseInstance                             map[string]any `json:"databricks_database_instance,omitempty"`
	DefaultNamespaceSetting                      map[string]any `json:"databricks_default_namespace_setting,omitempty"`
	DisableLegacyAccessSetting                   map[string]any `json:"databricks_disable_legacy_access_setting,omitempty"`
	DisableLegacyDbfsSetting                     map[string]any `json:"databricks_disable_legacy_dbfs_setting,omitempty"`
	DisableLegacyFeaturesSetting                 map[string]any `json:"databricks_disable_legacy_features_setting,omitempty"`
	EnhancedSecurityMonitoringWorkspaceSetting   map[string]any `json:"databricks_enhanced_security_monitoring_workspace_setting,omitempty"`
	ExternalMetadata                             map[string]any `json:"databricks_external_metadata,omitempty"`
	MaterializedFeaturesFeatureTag               map[string]any `json:"databricks_materialized_features_feature_tag,omitempty"`
	OnlineStore                                  map[string]any `json:"databricks_online_store,omitempty"`
	QualityMonitorV2                             map[string]any `json:"databricks_quality_monitor_v2,omitempty"`
	RecipientFederationPolicy                    map[string]any `json:"databricks_recipient_federation_policy,omitempty"`
	RequestForAccess                             map[string]any `json:"databricks_request_for_access,omitempty"`
	RestrictWorkspaceAdminsSetting               map[string]any `json:"databricks_restrict_workspace_admins_setting,omitempty"`
	TagPolicy                                    map[string]any `json:"databricks_tag_policy,omitempty"`
	WorkspaceNetworkOption                       map[string]any `json:"databricks_workspace_network_option,omitempty"`
	WorkspaceSetting                             map[string]any `json:"databricks_workspace_setting,omitempty"`
}

func NewResources() *Resources {
	return &Resources{
		AccountNetworkPolicy:                         make(map[string]any),
		AccountSetting:                               make(map[string]any),
		AibiDashboardEmbeddingAccessPolicySetting:    make(map[string]any),
		AibiDashboardEmbeddingApprovedDomainsSetting: make(map[string]any),
		AlertV2:                                    make(map[string]any),
		AutomaticClusterUpdateWorkspaceSetting:     make(map[string]any),
		BudgetPolicy:                               make(map[string]any),
		CleanRoomAsset:                             make(map[string]any),
		CleanRoomAutoApprovalRule:                  make(map[string]any),
		CleanRoomsCleanRoom:                        make(map[string]any),
		ComplianceSecurityProfileWorkspaceSetting:  make(map[string]any),
		DatabaseDatabaseCatalog:                    make(map[string]any),
		DatabaseInstance:                           make(map[string]any),
		DefaultNamespaceSetting:                    make(map[string]any),
		DisableLegacyAccessSetting:                 make(map[string]any),
		DisableLegacyDbfsSetting:                   make(map[string]any),
		DisableLegacyFeaturesSetting:               make(map[string]any),
		EnhancedSecurityMonitoringWorkspaceSetting: make(map[string]any),
		ExternalMetadata:                           make(map[string]any),
		MaterializedFeaturesFeatureTag:             make(map[string]any),
		OnlineStore:                                make(map[string]any),
		QualityMonitorV2:                           make(map[string]any),
		RecipientFederationPolicy:                  make(map[string]any),
		RequestForAccess:                           make(map[string]any),
		RestrictWorkspaceAdminsSetting:             make(map[string]any),
		TagPolicy:                                  make(map[string]any),
		WorkspaceNetworkOption:                     make(map[string]any),
		WorkspaceSetting:                           make(map[string]any),
	}
}
