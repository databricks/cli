// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsCatalogs struct {
	Name string `json:"name"`
}

type DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappingsUriByRegion struct {
	Region string `json:"region"`
	Uri    string `json:"uri"`
}

type DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappings struct {
	Name        string                                                                                 `json:"name"`
	UriByRegion []DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappingsUriByRegion `json:"uri_by_region,omitempty"`
}

type DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssets struct {
	Catalogs                    []DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsCatalogs         `json:"catalogs,omitempty"`
	DataReplicationWorkspaceSet string                                                                      `json:"data_replication_workspace_set"`
	LocationMappings            []DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappings `json:"location_mappings,omitempty"`
}

type DataSourceDisasterRecoveryFailoverGroupWorkspaceSets struct {
	Name                     string   `json:"name"`
	ReplicateWorkspaceAssets bool     `json:"replicate_workspace_assets"`
	StableUrlNames           []string `json:"stable_url_names,omitempty"`
	WorkspaceIds             []string `json:"workspace_ids"`
}

type DataSourceDisasterRecoveryFailoverGroup struct {
	CreateTime             string                                                     `json:"create_time,omitempty"`
	EffectivePrimaryRegion string                                                     `json:"effective_primary_region,omitempty"`
	Etag                   string                                                     `json:"etag,omitempty"`
	InitialPrimaryRegion   string                                                     `json:"initial_primary_region,omitempty"`
	Name                   string                                                     `json:"name"`
	Regions                []string                                                   `json:"regions,omitempty"`
	ReplicationPoint       string                                                     `json:"replication_point,omitempty"`
	State                  string                                                     `json:"state,omitempty"`
	UnityCatalogAssets     *DataSourceDisasterRecoveryFailoverGroupUnityCatalogAssets `json:"unity_catalog_assets,omitempty"`
	UpdateTime             string                                                     `json:"update_time,omitempty"`
	WorkspaceSets          []DataSourceDisasterRecoveryFailoverGroupWorkspaceSets     `json:"workspace_sets,omitempty"`
}
