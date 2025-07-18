// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type Resources struct {
	AccessControlRuleSet                         map[string]any `json:"databricks_access_control_rule_set,omitempty"`
	AccountNetworkPolicy                         map[string]any `json:"databricks_account_network_policy,omitempty"`
	AibiDashboardEmbeddingAccessPolicySetting    map[string]any `json:"databricks_aibi_dashboard_embedding_access_policy_setting,omitempty"`
	AibiDashboardEmbeddingApprovedDomainsSetting map[string]any `json:"databricks_aibi_dashboard_embedding_approved_domains_setting,omitempty"`
	Alert                                        map[string]any `json:"databricks_alert,omitempty"`
	AlertV2                                      map[string]any `json:"databricks_alert_v2,omitempty"`
	App                                          map[string]any `json:"databricks_app,omitempty"`
	ArtifactAllowlist                            map[string]any `json:"databricks_artifact_allowlist,omitempty"`
	AutomaticClusterUpdateWorkspaceSetting       map[string]any `json:"databricks_automatic_cluster_update_workspace_setting,omitempty"`
	AwsS3Mount                                   map[string]any `json:"databricks_aws_s3_mount,omitempty"`
	AzureAdlsGen1Mount                           map[string]any `json:"databricks_azure_adls_gen1_mount,omitempty"`
	AzureAdlsGen2Mount                           map[string]any `json:"databricks_azure_adls_gen2_mount,omitempty"`
	AzureBlobMount                               map[string]any `json:"databricks_azure_blob_mount,omitempty"`
	Budget                                       map[string]any `json:"databricks_budget,omitempty"`
	BudgetPolicy                                 map[string]any `json:"databricks_budget_policy,omitempty"`
	Catalog                                      map[string]any `json:"databricks_catalog,omitempty"`
	CatalogWorkspaceBinding                      map[string]any `json:"databricks_catalog_workspace_binding,omitempty"`
	CleanRoomsCleanRoom                          map[string]any `json:"databricks_clean_rooms_clean_room,omitempty"`
	Cluster                                      map[string]any `json:"databricks_cluster,omitempty"`
	ClusterPolicy                                map[string]any `json:"databricks_cluster_policy,omitempty"`
	ComplianceSecurityProfileWorkspaceSetting    map[string]any `json:"databricks_compliance_security_profile_workspace_setting,omitempty"`
	Connection                                   map[string]any `json:"databricks_connection,omitempty"`
	Credential                                   map[string]any `json:"databricks_credential,omitempty"`
	CustomAppIntegration                         map[string]any `json:"databricks_custom_app_integration,omitempty"`
	Dashboard                                    map[string]any `json:"databricks_dashboard,omitempty"`
	DatabaseInstance                             map[string]any `json:"databricks_database_instance,omitempty"`
	DbfsFile                                     map[string]any `json:"databricks_dbfs_file,omitempty"`
	DefaultNamespaceSetting                      map[string]any `json:"databricks_default_namespace_setting,omitempty"`
	Directory                                    map[string]any `json:"databricks_directory,omitempty"`
	DisableLegacyAccessSetting                   map[string]any `json:"databricks_disable_legacy_access_setting,omitempty"`
	DisableLegacyDbfsSetting                     map[string]any `json:"databricks_disable_legacy_dbfs_setting,omitempty"`
	DisableLegacyFeaturesSetting                 map[string]any `json:"databricks_disable_legacy_features_setting,omitempty"`
	EnhancedSecurityMonitoringWorkspaceSetting   map[string]any `json:"databricks_enhanced_security_monitoring_workspace_setting,omitempty"`
	Entitlements                                 map[string]any `json:"databricks_entitlements,omitempty"`
	ExternalLocation                             map[string]any `json:"databricks_external_location,omitempty"`
	ExternalMetadata                             map[string]any `json:"databricks_external_metadata,omitempty"`
	File                                         map[string]any `json:"databricks_file,omitempty"`
	GitCredential                                map[string]any `json:"databricks_git_credential,omitempty"`
	GlobalInitScript                             map[string]any `json:"databricks_global_init_script,omitempty"`
	Grant                                        map[string]any `json:"databricks_grant,omitempty"`
	Grants                                       map[string]any `json:"databricks_grants,omitempty"`
	Group                                        map[string]any `json:"databricks_group,omitempty"`
	GroupInstanceProfile                         map[string]any `json:"databricks_group_instance_profile,omitempty"`
	GroupMember                                  map[string]any `json:"databricks_group_member,omitempty"`
	GroupRole                                    map[string]any `json:"databricks_group_role,omitempty"`
	InstancePool                                 map[string]any `json:"databricks_instance_pool,omitempty"`
	InstanceProfile                              map[string]any `json:"databricks_instance_profile,omitempty"`
	IpAccessList                                 map[string]any `json:"databricks_ip_access_list,omitempty"`
	Job                                          map[string]any `json:"databricks_job,omitempty"`
	LakehouseMonitor                             map[string]any `json:"databricks_lakehouse_monitor,omitempty"`
	Library                                      map[string]any `json:"databricks_library,omitempty"`
	MaterializedFeaturesFeatureTag               map[string]any `json:"databricks_materialized_features_feature_tag,omitempty"`
	Metastore                                    map[string]any `json:"databricks_metastore,omitempty"`
	MetastoreAssignment                          map[string]any `json:"databricks_metastore_assignment,omitempty"`
	MetastoreDataAccess                          map[string]any `json:"databricks_metastore_data_access,omitempty"`
	MlflowExperiment                             map[string]any `json:"databricks_mlflow_experiment,omitempty"`
	MlflowModel                                  map[string]any `json:"databricks_mlflow_model,omitempty"`
	MlflowWebhook                                map[string]any `json:"databricks_mlflow_webhook,omitempty"`
	ModelServing                                 map[string]any `json:"databricks_model_serving,omitempty"`
	ModelServingProvisionedThroughput            map[string]any `json:"databricks_model_serving_provisioned_throughput,omitempty"`
	Mount                                        map[string]any `json:"databricks_mount,omitempty"`
	MwsCredentials                               map[string]any `json:"databricks_mws_credentials,omitempty"`
	MwsCustomerManagedKeys                       map[string]any `json:"databricks_mws_customer_managed_keys,omitempty"`
	MwsLogDelivery                               map[string]any `json:"databricks_mws_log_delivery,omitempty"`
	MwsNccBinding                                map[string]any `json:"databricks_mws_ncc_binding,omitempty"`
	MwsNccPrivateEndpointRule                    map[string]any `json:"databricks_mws_ncc_private_endpoint_rule,omitempty"`
	MwsNetworkConnectivityConfig                 map[string]any `json:"databricks_mws_network_connectivity_config,omitempty"`
	MwsNetworks                                  map[string]any `json:"databricks_mws_networks,omitempty"`
	MwsPermissionAssignment                      map[string]any `json:"databricks_mws_permission_assignment,omitempty"`
	MwsPrivateAccessSettings                     map[string]any `json:"databricks_mws_private_access_settings,omitempty"`
	MwsStorageConfigurations                     map[string]any `json:"databricks_mws_storage_configurations,omitempty"`
	MwsVpcEndpoint                               map[string]any `json:"databricks_mws_vpc_endpoint,omitempty"`
	MwsWorkspaces                                map[string]any `json:"databricks_mws_workspaces,omitempty"`
	Notebook                                     map[string]any `json:"databricks_notebook,omitempty"`
	NotificationDestination                      map[string]any `json:"databricks_notification_destination,omitempty"`
	OboToken                                     map[string]any `json:"databricks_obo_token,omitempty"`
	OnlineStore                                  map[string]any `json:"databricks_online_store,omitempty"`
	OnlineTable                                  map[string]any `json:"databricks_online_table,omitempty"`
	PermissionAssignment                         map[string]any `json:"databricks_permission_assignment,omitempty"`
	Permissions                                  map[string]any `json:"databricks_permissions,omitempty"`
	Pipeline                                     map[string]any `json:"databricks_pipeline,omitempty"`
	Provider                                     map[string]any `json:"databricks_provider,omitempty"`
	QualityMonitor                               map[string]any `json:"databricks_quality_monitor,omitempty"`
	QualityMonitorV2                             map[string]any `json:"databricks_quality_monitor_v2,omitempty"`
	Query                                        map[string]any `json:"databricks_query,omitempty"`
	Recipient                                    map[string]any `json:"databricks_recipient,omitempty"`
	RecipientFederationPolicy                    map[string]any `json:"databricks_recipient_federation_policy,omitempty"`
	RegisteredModel                              map[string]any `json:"databricks_registered_model,omitempty"`
	Repo                                         map[string]any `json:"databricks_repo,omitempty"`
	RestrictWorkspaceAdminsSetting               map[string]any `json:"databricks_restrict_workspace_admins_setting,omitempty"`
	Schema                                       map[string]any `json:"databricks_schema,omitempty"`
	Secret                                       map[string]any `json:"databricks_secret,omitempty"`
	SecretAcl                                    map[string]any `json:"databricks_secret_acl,omitempty"`
	SecretScope                                  map[string]any `json:"databricks_secret_scope,omitempty"`
	ServicePrincipal                             map[string]any `json:"databricks_service_principal,omitempty"`
	ServicePrincipalRole                         map[string]any `json:"databricks_service_principal_role,omitempty"`
	ServicePrincipalSecret                       map[string]any `json:"databricks_service_principal_secret,omitempty"`
	Share                                        map[string]any `json:"databricks_share,omitempty"`
	SqlAlert                                     map[string]any `json:"databricks_sql_alert,omitempty"`
	SqlDashboard                                 map[string]any `json:"databricks_sql_dashboard,omitempty"`
	SqlEndpoint                                  map[string]any `json:"databricks_sql_endpoint,omitempty"`
	SqlGlobalConfig                              map[string]any `json:"databricks_sql_global_config,omitempty"`
	SqlPermissions                               map[string]any `json:"databricks_sql_permissions,omitempty"`
	SqlQuery                                     map[string]any `json:"databricks_sql_query,omitempty"`
	SqlTable                                     map[string]any `json:"databricks_sql_table,omitempty"`
	SqlVisualization                             map[string]any `json:"databricks_sql_visualization,omitempty"`
	SqlWidget                                    map[string]any `json:"databricks_sql_widget,omitempty"`
	StorageCredential                            map[string]any `json:"databricks_storage_credential,omitempty"`
	SystemSchema                                 map[string]any `json:"databricks_system_schema,omitempty"`
	Table                                        map[string]any `json:"databricks_table,omitempty"`
	Token                                        map[string]any `json:"databricks_token,omitempty"`
	User                                         map[string]any `json:"databricks_user,omitempty"`
	UserInstanceProfile                          map[string]any `json:"databricks_user_instance_profile,omitempty"`
	UserRole                                     map[string]any `json:"databricks_user_role,omitempty"`
	VectorSearchEndpoint                         map[string]any `json:"databricks_vector_search_endpoint,omitempty"`
	VectorSearchIndex                            map[string]any `json:"databricks_vector_search_index,omitempty"`
	Volume                                       map[string]any `json:"databricks_volume,omitempty"`
	WorkspaceBinding                             map[string]any `json:"databricks_workspace_binding,omitempty"`
	WorkspaceConf                                map[string]any `json:"databricks_workspace_conf,omitempty"`
	WorkspaceFile                                map[string]any `json:"databricks_workspace_file,omitempty"`
	WorkspaceNetworkOption                       map[string]any `json:"databricks_workspace_network_option,omitempty"`
}

func NewResources() *Resources {
	return &Resources{
		AccessControlRuleSet:                         make(map[string]any),
		AccountNetworkPolicy:                         make(map[string]any),
		AibiDashboardEmbeddingAccessPolicySetting:    make(map[string]any),
		AibiDashboardEmbeddingApprovedDomainsSetting: make(map[string]any),
		Alert:                                  make(map[string]any),
		AlertV2:                                make(map[string]any),
		App:                                    make(map[string]any),
		ArtifactAllowlist:                      make(map[string]any),
		AutomaticClusterUpdateWorkspaceSetting: make(map[string]any),
		AwsS3Mount:                             make(map[string]any),
		AzureAdlsGen1Mount:                     make(map[string]any),
		AzureAdlsGen2Mount:                     make(map[string]any),
		AzureBlobMount:                         make(map[string]any),
		Budget:                                 make(map[string]any),
		BudgetPolicy:                           make(map[string]any),
		Catalog:                                make(map[string]any),
		CatalogWorkspaceBinding:                make(map[string]any),
		CleanRoomsCleanRoom:                    make(map[string]any),
		Cluster:                                make(map[string]any),
		ClusterPolicy:                          make(map[string]any),
		ComplianceSecurityProfileWorkspaceSetting: make(map[string]any),
		Connection:                   make(map[string]any),
		Credential:                   make(map[string]any),
		CustomAppIntegration:         make(map[string]any),
		Dashboard:                    make(map[string]any),
		DatabaseInstance:             make(map[string]any),
		DbfsFile:                     make(map[string]any),
		DefaultNamespaceSetting:      make(map[string]any),
		Directory:                    make(map[string]any),
		DisableLegacyAccessSetting:   make(map[string]any),
		DisableLegacyDbfsSetting:     make(map[string]any),
		DisableLegacyFeaturesSetting: make(map[string]any),
		EnhancedSecurityMonitoringWorkspaceSetting: make(map[string]any),
		Entitlements:                      make(map[string]any),
		ExternalLocation:                  make(map[string]any),
		ExternalMetadata:                  make(map[string]any),
		File:                              make(map[string]any),
		GitCredential:                     make(map[string]any),
		GlobalInitScript:                  make(map[string]any),
		Grant:                             make(map[string]any),
		Grants:                            make(map[string]any),
		Group:                             make(map[string]any),
		GroupInstanceProfile:              make(map[string]any),
		GroupMember:                       make(map[string]any),
		GroupRole:                         make(map[string]any),
		InstancePool:                      make(map[string]any),
		InstanceProfile:                   make(map[string]any),
		IpAccessList:                      make(map[string]any),
		Job:                               make(map[string]any),
		LakehouseMonitor:                  make(map[string]any),
		Library:                           make(map[string]any),
		MaterializedFeaturesFeatureTag:    make(map[string]any),
		Metastore:                         make(map[string]any),
		MetastoreAssignment:               make(map[string]any),
		MetastoreDataAccess:               make(map[string]any),
		MlflowExperiment:                  make(map[string]any),
		MlflowModel:                       make(map[string]any),
		MlflowWebhook:                     make(map[string]any),
		ModelServing:                      make(map[string]any),
		ModelServingProvisionedThroughput: make(map[string]any),
		Mount:                             make(map[string]any),
		MwsCredentials:                    make(map[string]any),
		MwsCustomerManagedKeys:            make(map[string]any),
		MwsLogDelivery:                    make(map[string]any),
		MwsNccBinding:                     make(map[string]any),
		MwsNccPrivateEndpointRule:         make(map[string]any),
		MwsNetworkConnectivityConfig:      make(map[string]any),
		MwsNetworks:                       make(map[string]any),
		MwsPermissionAssignment:           make(map[string]any),
		MwsPrivateAccessSettings:          make(map[string]any),
		MwsStorageConfigurations:          make(map[string]any),
		MwsVpcEndpoint:                    make(map[string]any),
		MwsWorkspaces:                     make(map[string]any),
		Notebook:                          make(map[string]any),
		NotificationDestination:           make(map[string]any),
		OboToken:                          make(map[string]any),
		OnlineStore:                       make(map[string]any),
		OnlineTable:                       make(map[string]any),
		PermissionAssignment:              make(map[string]any),
		Permissions:                       make(map[string]any),
		Pipeline:                          make(map[string]any),
		Provider:                          make(map[string]any),
		QualityMonitor:                    make(map[string]any),
		QualityMonitorV2:                  make(map[string]any),
		Query:                             make(map[string]any),
		Recipient:                         make(map[string]any),
		RecipientFederationPolicy:         make(map[string]any),
		RegisteredModel:                   make(map[string]any),
		Repo:                              make(map[string]any),
		RestrictWorkspaceAdminsSetting:    make(map[string]any),
		Schema:                            make(map[string]any),
		Secret:                            make(map[string]any),
		SecretAcl:                         make(map[string]any),
		SecretScope:                       make(map[string]any),
		ServicePrincipal:                  make(map[string]any),
		ServicePrincipalRole:              make(map[string]any),
		ServicePrincipalSecret:            make(map[string]any),
		Share:                             make(map[string]any),
		SqlAlert:                          make(map[string]any),
		SqlDashboard:                      make(map[string]any),
		SqlEndpoint:                       make(map[string]any),
		SqlGlobalConfig:                   make(map[string]any),
		SqlPermissions:                    make(map[string]any),
		SqlQuery:                          make(map[string]any),
		SqlTable:                          make(map[string]any),
		SqlVisualization:                  make(map[string]any),
		SqlWidget:                         make(map[string]any),
		StorageCredential:                 make(map[string]any),
		SystemSchema:                      make(map[string]any),
		Table:                             make(map[string]any),
		Token:                             make(map[string]any),
		User:                              make(map[string]any),
		UserInstanceProfile:               make(map[string]any),
		UserRole:                          make(map[string]any),
		VectorSearchEndpoint:              make(map[string]any),
		VectorSearchIndex:                 make(map[string]any),
		Volume:                            make(map[string]any),
		WorkspaceBinding:                  make(map[string]any),
		WorkspaceConf:                     make(map[string]any),
		WorkspaceFile:                     make(map[string]any),
		WorkspaceNetworkOption:            make(map[string]any),
	}
}
