// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsCatalogs struct {
	Name string `json:"name"`
}

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsLocationMappingsUriByRegion struct {
	Region string `json:"region"`
	Uri    string `json:"uri"`
}

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsLocationMappings struct {
	Name        string                                                                                                `json:"name"`
	UriByRegion []DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsLocationMappingsUriByRegion `json:"uri_by_region,omitempty"`
}

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssets struct {
	Catalogs                    []DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsCatalogs         `json:"catalogs,omitempty"`
	DataReplicationWorkspaceSet string                                                                                     `json:"data_replication_workspace_set"`
	LocationMappings            []DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssetsLocationMappings `json:"location_mappings,omitempty"`
}

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsWorkspaceSets struct {
	Name                     string   `json:"name"`
	ReplicateWorkspaceAssets bool     `json:"replicate_workspace_assets"`
	StableUrlNames           []string `json:"stable_url_names,omitempty"`
	WorkspaceIds             []string `json:"workspace_ids"`
}

type DataSourceDisasterRecoveryFailoverGroupsFailoverGroups struct {
	CreateTime             string                                                                    `json:"create_time,omitempty"`
	EffectivePrimaryRegion string                                                                    `json:"effective_primary_region,omitempty"`
	Etag                   string                                                                    `json:"etag,omitempty"`
	InitialPrimaryRegion   string                                                                    `json:"initial_primary_region,omitempty"`
	Name                   string                                                                    `json:"name"`
	Regions                []string                                                                  `json:"regions,omitempty"`
	ReplicationPoint       string                                                                    `json:"replication_point,omitempty"`
	State                  string                                                                    `json:"state,omitempty"`
	UnityCatalogAssets     *DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsUnityCatalogAssets `json:"unity_catalog_assets,omitempty"`
	UpdateTime             string                                                                    `json:"update_time,omitempty"`
	WorkspaceSets          []DataSourceDisasterRecoveryFailoverGroupsFailoverGroupsWorkspaceSets     `json:"workspace_sets,omitempty"`
}

type DataSourceDisasterRecoveryFailoverGroups struct {
	FailoverGroups []DataSourceDisasterRecoveryFailoverGroupsFailoverGroups `json:"failover_groups,omitempty"`
	PageSize       int                                                      `json:"page_size,omitempty"`
	Parent         string                                                   `json:"parent"`
}
