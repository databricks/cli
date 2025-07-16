// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSources struct {
	AccountNetworkPolicies          map[string]any `json:"databricks_account_network_policies,omitempty"`
	AccountNetworkPolicy            map[string]any `json:"databricks_account_network_policy,omitempty"`
	AlertV2                         map[string]any `json:"databricks_alert_v2,omitempty"`
	AlertsV2                        map[string]any `json:"databricks_alerts_v2,omitempty"`
	App                             map[string]any `json:"databricks_app,omitempty"`
	Apps                            map[string]any `json:"databricks_apps,omitempty"`
	AwsAssumeRolePolicy             map[string]any `json:"databricks_aws_assume_role_policy,omitempty"`
	AwsBucketPolicy                 map[string]any `json:"databricks_aws_bucket_policy,omitempty"`
	AwsCrossaccountPolicy           map[string]any `json:"databricks_aws_crossaccount_policy,omitempty"`
	AwsUnityCatalogAssumeRolePolicy map[string]any `json:"databricks_aws_unity_catalog_assume_role_policy,omitempty"`
	AwsUnityCatalogPolicy           map[string]any `json:"databricks_aws_unity_catalog_policy,omitempty"`
	BudgetPolicies                  map[string]any `json:"databricks_budget_policies,omitempty"`
	BudgetPolicy                    map[string]any `json:"databricks_budget_policy,omitempty"`
	Catalog                         map[string]any `json:"databricks_catalog,omitempty"`
	Catalogs                        map[string]any `json:"databricks_catalogs,omitempty"`
	CleanRoomsCleanRoom             map[string]any `json:"databricks_clean_rooms_clean_room,omitempty"`
	CleanRoomsCleanRooms            map[string]any `json:"databricks_clean_rooms_clean_rooms,omitempty"`
	Cluster                         map[string]any `json:"databricks_cluster,omitempty"`
	ClusterPolicy                   map[string]any `json:"databricks_cluster_policy,omitempty"`
	Clusters                        map[string]any `json:"databricks_clusters,omitempty"`
	CurrentConfig                   map[string]any `json:"databricks_current_config,omitempty"`
	CurrentMetastore                map[string]any `json:"databricks_current_metastore,omitempty"`
	CurrentUser                     map[string]any `json:"databricks_current_user,omitempty"`
	Dashboards                      map[string]any `json:"databricks_dashboards,omitempty"`
	DatabaseInstance                map[string]any `json:"databricks_database_instance,omitempty"`
	DatabaseInstances               map[string]any `json:"databricks_database_instances,omitempty"`
	DbfsFile                        map[string]any `json:"databricks_dbfs_file,omitempty"`
	DbfsFilePaths                   map[string]any `json:"databricks_dbfs_file_paths,omitempty"`
	Directory                       map[string]any `json:"databricks_directory,omitempty"`
	ExternalLocation                map[string]any `json:"databricks_external_location,omitempty"`
	ExternalLocations               map[string]any `json:"databricks_external_locations,omitempty"`
	ExternalMetadata                map[string]any `json:"databricks_external_metadata,omitempty"`
	ExternalMetadatas               map[string]any `json:"databricks_external_metadatas,omitempty"`
	Functions                       map[string]any `json:"databricks_functions,omitempty"`
	Group                           map[string]any `json:"databricks_group,omitempty"`
	InstancePool                    map[string]any `json:"databricks_instance_pool,omitempty"`
	InstanceProfiles                map[string]any `json:"databricks_instance_profiles,omitempty"`
	Job                             map[string]any `json:"databricks_job,omitempty"`
	Jobs                            map[string]any `json:"databricks_jobs,omitempty"`
	MaterializedFeaturesFeatureTag  map[string]any `json:"databricks_materialized_features_feature_tag,omitempty"`
	MaterializedFeaturesFeatureTags map[string]any `json:"databricks_materialized_features_feature_tags,omitempty"`
	Metastore                       map[string]any `json:"databricks_metastore,omitempty"`
	Metastores                      map[string]any `json:"databricks_metastores,omitempty"`
	MlflowExperiment                map[string]any `json:"databricks_mlflow_experiment,omitempty"`
	MlflowModel                     map[string]any `json:"databricks_mlflow_model,omitempty"`
	MlflowModels                    map[string]any `json:"databricks_mlflow_models,omitempty"`
	MwsCredentials                  map[string]any `json:"databricks_mws_credentials,omitempty"`
	MwsNetworkConnectivityConfig    map[string]any `json:"databricks_mws_network_connectivity_config,omitempty"`
	MwsNetworkConnectivityConfigs   map[string]any `json:"databricks_mws_network_connectivity_configs,omitempty"`
	MwsWorkspaces                   map[string]any `json:"databricks_mws_workspaces,omitempty"`
	NodeType                        map[string]any `json:"databricks_node_type,omitempty"`
	Notebook                        map[string]any `json:"databricks_notebook,omitempty"`
	NotebookPaths                   map[string]any `json:"databricks_notebook_paths,omitempty"`
	NotificationDestinations        map[string]any `json:"databricks_notification_destinations,omitempty"`
	OnlineStore                     map[string]any `json:"databricks_online_store,omitempty"`
	OnlineStores                    map[string]any `json:"databricks_online_stores,omitempty"`
	Pipelines                       map[string]any `json:"databricks_pipelines,omitempty"`
	QualityMonitorV2                map[string]any `json:"databricks_quality_monitor_v2,omitempty"`
	QualityMonitorsV2               map[string]any `json:"databricks_quality_monitors_v2,omitempty"`
	RecipientFederationPolicies     map[string]any `json:"databricks_recipient_federation_policies,omitempty"`
	RecipientFederationPolicy       map[string]any `json:"databricks_recipient_federation_policy,omitempty"`
	RegisteredModel                 map[string]any `json:"databricks_registered_model,omitempty"`
	RegisteredModelVersions         map[string]any `json:"databricks_registered_model_versions,omitempty"`
	Schema                          map[string]any `json:"databricks_schema,omitempty"`
	Schemas                         map[string]any `json:"databricks_schemas,omitempty"`
	ServicePrincipal                map[string]any `json:"databricks_service_principal,omitempty"`
	ServicePrincipals               map[string]any `json:"databricks_service_principals,omitempty"`
	ServingEndpoints                map[string]any `json:"databricks_serving_endpoints,omitempty"`
	Share                           map[string]any `json:"databricks_share,omitempty"`
	Shares                          map[string]any `json:"databricks_shares,omitempty"`
	SparkVersion                    map[string]any `json:"databricks_spark_version,omitempty"`
	SqlWarehouse                    map[string]any `json:"databricks_sql_warehouse,omitempty"`
	SqlWarehouses                   map[string]any `json:"databricks_sql_warehouses,omitempty"`
	StorageCredential               map[string]any `json:"databricks_storage_credential,omitempty"`
	StorageCredentials              map[string]any `json:"databricks_storage_credentials,omitempty"`
	Table                           map[string]any `json:"databricks_table,omitempty"`
	Tables                          map[string]any `json:"databricks_tables,omitempty"`
	User                            map[string]any `json:"databricks_user,omitempty"`
	Views                           map[string]any `json:"databricks_views,omitempty"`
	Volume                          map[string]any `json:"databricks_volume,omitempty"`
	Volumes                         map[string]any `json:"databricks_volumes,omitempty"`
	WorkspaceNetworkOption          map[string]any `json:"databricks_workspace_network_option,omitempty"`
	Zones                           map[string]any `json:"databricks_zones,omitempty"`
}

func NewDataSources() *DataSources {
	return &DataSources{
		AccountNetworkPolicies:          make(map[string]any),
		AccountNetworkPolicy:            make(map[string]any),
		AlertV2:                         make(map[string]any),
		AlertsV2:                        make(map[string]any),
		App:                             make(map[string]any),
		Apps:                            make(map[string]any),
		AwsAssumeRolePolicy:             make(map[string]any),
		AwsBucketPolicy:                 make(map[string]any),
		AwsCrossaccountPolicy:           make(map[string]any),
		AwsUnityCatalogAssumeRolePolicy: make(map[string]any),
		AwsUnityCatalogPolicy:           make(map[string]any),
		BudgetPolicies:                  make(map[string]any),
		BudgetPolicy:                    make(map[string]any),
		Catalog:                         make(map[string]any),
		Catalogs:                        make(map[string]any),
		CleanRoomsCleanRoom:             make(map[string]any),
		CleanRoomsCleanRooms:            make(map[string]any),
		Cluster:                         make(map[string]any),
		ClusterPolicy:                   make(map[string]any),
		Clusters:                        make(map[string]any),
		CurrentConfig:                   make(map[string]any),
		CurrentMetastore:                make(map[string]any),
		CurrentUser:                     make(map[string]any),
		Dashboards:                      make(map[string]any),
		DatabaseInstance:                make(map[string]any),
		DatabaseInstances:               make(map[string]any),
		DbfsFile:                        make(map[string]any),
		DbfsFilePaths:                   make(map[string]any),
		Directory:                       make(map[string]any),
		ExternalLocation:                make(map[string]any),
		ExternalLocations:               make(map[string]any),
		ExternalMetadata:                make(map[string]any),
		ExternalMetadatas:               make(map[string]any),
		Functions:                       make(map[string]any),
		Group:                           make(map[string]any),
		InstancePool:                    make(map[string]any),
		InstanceProfiles:                make(map[string]any),
		Job:                             make(map[string]any),
		Jobs:                            make(map[string]any),
		MaterializedFeaturesFeatureTag:  make(map[string]any),
		MaterializedFeaturesFeatureTags: make(map[string]any),
		Metastore:                       make(map[string]any),
		Metastores:                      make(map[string]any),
		MlflowExperiment:                make(map[string]any),
		MlflowModel:                     make(map[string]any),
		MlflowModels:                    make(map[string]any),
		MwsCredentials:                  make(map[string]any),
		MwsNetworkConnectivityConfig:    make(map[string]any),
		MwsNetworkConnectivityConfigs:   make(map[string]any),
		MwsWorkspaces:                   make(map[string]any),
		NodeType:                        make(map[string]any),
		Notebook:                        make(map[string]any),
		NotebookPaths:                   make(map[string]any),
		NotificationDestinations:        make(map[string]any),
		OnlineStore:                     make(map[string]any),
		OnlineStores:                    make(map[string]any),
		Pipelines:                       make(map[string]any),
		QualityMonitorV2:                make(map[string]any),
		QualityMonitorsV2:               make(map[string]any),
		RecipientFederationPolicies:     make(map[string]any),
		RecipientFederationPolicy:       make(map[string]any),
		RegisteredModel:                 make(map[string]any),
		RegisteredModelVersions:         make(map[string]any),
		Schema:                          make(map[string]any),
		Schemas:                         make(map[string]any),
		ServicePrincipal:                make(map[string]any),
		ServicePrincipals:               make(map[string]any),
		ServingEndpoints:                make(map[string]any),
		Share:                           make(map[string]any),
		Shares:                          make(map[string]any),
		SparkVersion:                    make(map[string]any),
		SqlWarehouse:                    make(map[string]any),
		SqlWarehouses:                   make(map[string]any),
		StorageCredential:               make(map[string]any),
		StorageCredentials:              make(map[string]any),
		Table:                           make(map[string]any),
		Tables:                          make(map[string]any),
		User:                            make(map[string]any),
		Views:                           make(map[string]any),
		Volume:                          make(map[string]any),
		Volumes:                         make(map[string]any),
		WorkspaceNetworkOption:          make(map[string]any),
		Zones:                           make(map[string]any),
	}
}
