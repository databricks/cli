// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSources struct {
	AccountNetworkPolicies                 map[string]any `json:"databricks_account_network_policies,omitempty"`
	AccountNetworkPolicy                   map[string]any `json:"databricks_account_network_policy,omitempty"`
	AccountSetting                         map[string]any `json:"databricks_account_setting,omitempty"`
	AlertV2                                map[string]any `json:"databricks_alert_v2,omitempty"`
	AlertsV2                               map[string]any `json:"databricks_alerts_v2,omitempty"`
	BudgetPolicies                         map[string]any `json:"databricks_budget_policies,omitempty"`
	BudgetPolicy                           map[string]any `json:"databricks_budget_policy,omitempty"`
	CleanRoomAsset                         map[string]any `json:"databricks_clean_room_asset,omitempty"`
	CleanRoomAssetRevisionsCleanRoomAsset  map[string]any `json:"databricks_clean_room_asset_revisions_clean_room_asset,omitempty"`
	CleanRoomAssetRevisionsCleanRoomAssets map[string]any `json:"databricks_clean_room_asset_revisions_clean_room_assets,omitempty"`
	CleanRoomAssets                        map[string]any `json:"databricks_clean_room_assets,omitempty"`
	CleanRoomAutoApprovalRule              map[string]any `json:"databricks_clean_room_auto_approval_rule,omitempty"`
	CleanRoomAutoApprovalRules             map[string]any `json:"databricks_clean_room_auto_approval_rules,omitempty"`
	CleanRoomsCleanRoom                    map[string]any `json:"databricks_clean_rooms_clean_room,omitempty"`
	CleanRoomsCleanRooms                   map[string]any `json:"databricks_clean_rooms_clean_rooms,omitempty"`
	DatabaseDatabaseCatalog                map[string]any `json:"databricks_database_database_catalog,omitempty"`
	DatabaseDatabaseCatalogs               map[string]any `json:"databricks_database_database_catalogs,omitempty"`
	DatabaseInstance                       map[string]any `json:"databricks_database_instance,omitempty"`
	DatabaseInstances                      map[string]any `json:"databricks_database_instances,omitempty"`
	ExternalMetadata                       map[string]any `json:"databricks_external_metadata,omitempty"`
	ExternalMetadatas                      map[string]any `json:"databricks_external_metadatas,omitempty"`
	MaterializedFeaturesFeatureTag         map[string]any `json:"databricks_materialized_features_feature_tag,omitempty"`
	MaterializedFeaturesFeatureTags        map[string]any `json:"databricks_materialized_features_feature_tags,omitempty"`
	OnlineStore                            map[string]any `json:"databricks_online_store,omitempty"`
	OnlineStores                           map[string]any `json:"databricks_online_stores,omitempty"`
	QualityMonitorV2                       map[string]any `json:"databricks_quality_monitor_v2,omitempty"`
	QualityMonitorsV2                      map[string]any `json:"databricks_quality_monitors_v2,omitempty"`
	RecipientFederationPolicies            map[string]any `json:"databricks_recipient_federation_policies,omitempty"`
	RecipientFederationPolicy              map[string]any `json:"databricks_recipient_federation_policy,omitempty"`
	RequestForAccess                       map[string]any `json:"databricks_request_for_access,omitempty"`
	TagPolicies                            map[string]any `json:"databricks_tag_policies,omitempty"`
	TagPolicy                              map[string]any `json:"databricks_tag_policy,omitempty"`
	WorkspaceNetworkOption                 map[string]any `json:"databricks_workspace_network_option,omitempty"`
	WorkspaceSetting                       map[string]any `json:"databricks_workspace_setting,omitempty"`
}

func NewDataSources() *DataSources {
	return &DataSources{
		AccountNetworkPolicies:                 make(map[string]any),
		AccountNetworkPolicy:                   make(map[string]any),
		AccountSetting:                         make(map[string]any),
		AlertV2:                                make(map[string]any),
		AlertsV2:                               make(map[string]any),
		BudgetPolicies:                         make(map[string]any),
		BudgetPolicy:                           make(map[string]any),
		CleanRoomAsset:                         make(map[string]any),
		CleanRoomAssetRevisionsCleanRoomAsset:  make(map[string]any),
		CleanRoomAssetRevisionsCleanRoomAssets: make(map[string]any),
		CleanRoomAssets:                        make(map[string]any),
		CleanRoomAutoApprovalRule:              make(map[string]any),
		CleanRoomAutoApprovalRules:             make(map[string]any),
		CleanRoomsCleanRoom:                    make(map[string]any),
		CleanRoomsCleanRooms:                   make(map[string]any),
		DatabaseDatabaseCatalog:                make(map[string]any),
		DatabaseDatabaseCatalogs:               make(map[string]any),
		DatabaseInstance:                       make(map[string]any),
		DatabaseInstances:                      make(map[string]any),
		ExternalMetadata:                       make(map[string]any),
		ExternalMetadatas:                      make(map[string]any),
		MaterializedFeaturesFeatureTag:         make(map[string]any),
		MaterializedFeaturesFeatureTags:        make(map[string]any),
		OnlineStore:                            make(map[string]any),
		OnlineStores:                           make(map[string]any),
		QualityMonitorV2:                       make(map[string]any),
		QualityMonitorsV2:                      make(map[string]any),
		RecipientFederationPolicies:            make(map[string]any),
		RecipientFederationPolicy:              make(map[string]any),
		RequestForAccess:                       make(map[string]any),
		TagPolicies:                            make(map[string]any),
		TagPolicy:                              make(map[string]any),
		WorkspaceNetworkOption:                 make(map[string]any),
		WorkspaceSetting:                       make(map[string]any),
	}
}
