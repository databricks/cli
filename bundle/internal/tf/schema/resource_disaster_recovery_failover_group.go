// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsCatalogs struct {
	Name string `json:"name"`
}

type ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappingsUriByRegion struct {
	Region string `json:"region"`
	Uri    string `json:"uri"`
}

type ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappings struct {
	Name        string                                                                               `json:"name"`
	UriByRegion []ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappingsUriByRegion `json:"uri_by_region,omitempty"`
}

type ResourceDisasterRecoveryFailoverGroupUnityCatalogAssets struct {
	Catalogs                    []ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsCatalogs         `json:"catalogs,omitempty"`
	DataReplicationWorkspaceSet string                                                                    `json:"data_replication_workspace_set"`
	LocationMappings            []ResourceDisasterRecoveryFailoverGroupUnityCatalogAssetsLocationMappings `json:"location_mappings,omitempty"`
}

type ResourceDisasterRecoveryFailoverGroupWorkspaceSets struct {
	Name                     string   `json:"name"`
	ReplicateWorkspaceAssets bool     `json:"replicate_workspace_assets"`
	StableUrlNames           []string `json:"stable_url_names,omitempty"`
	WorkspaceIds             []string `json:"workspace_ids"`
}

type ResourceDisasterRecoveryFailoverGroup struct {
	CreateTime             string                                                   `json:"create_time,omitempty"`
	EffectivePrimaryRegion string                                                   `json:"effective_primary_region,omitempty"`
	Etag                   string                                                   `json:"etag,omitempty"`
	FailoverGroupId        string                                                   `json:"failover_group_id"`
	InitialPrimaryRegion   string                                                   `json:"initial_primary_region"`
	Name                   string                                                   `json:"name,omitempty"`
	Parent                 string                                                   `json:"parent"`
	Regions                []string                                                 `json:"regions"`
	ReplicationPoint       string                                                   `json:"replication_point,omitempty"`
	State                  string                                                   `json:"state,omitempty"`
	UnityCatalogAssets     *ResourceDisasterRecoveryFailoverGroupUnityCatalogAssets `json:"unity_catalog_assets,omitempty"`
	UpdateTime             string                                                   `json:"update_time,omitempty"`
	WorkspaceSets          []ResourceDisasterRecoveryFailoverGroupWorkspaceSets     `json:"workspace_sets,omitempty"`
}
