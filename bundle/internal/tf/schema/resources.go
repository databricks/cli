// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type Resources struct {
	AwsS3Mount               map[string]*ResourceAwsS3Mount               `json:"databricks_aws_s3_mount,omitempty"`
	AzureAdlsGen1Mount       map[string]*ResourceAzureAdlsGen1Mount       `json:"databricks_azure_adls_gen1_mount,omitempty"`
	AzureAdlsGen2Mount       map[string]*ResourceAzureAdlsGen2Mount       `json:"databricks_azure_adls_gen2_mount,omitempty"`
	AzureBlobMount           map[string]*ResourceAzureBlobMount           `json:"databricks_azure_blob_mount,omitempty"`
	Catalog                  map[string]*ResourceCatalog                  `json:"databricks_catalog,omitempty"`
	Cluster                  map[string]*ResourceCluster                  `json:"databricks_cluster,omitempty"`
	ClusterPolicy            map[string]*ResourceClusterPolicy            `json:"databricks_cluster_policy,omitempty"`
	DbfsFile                 map[string]*ResourceDbfsFile                 `json:"databricks_dbfs_file,omitempty"`
	Directory                map[string]*ResourceDirectory                `json:"databricks_directory,omitempty"`
	Entitlements             map[string]*ResourceEntitlements             `json:"databricks_entitlements,omitempty"`
	ExternalLocation         map[string]*ResourceExternalLocation         `json:"databricks_external_location,omitempty"`
	GitCredential            map[string]*ResourceGitCredential            `json:"databricks_git_credential,omitempty"`
	GlobalInitScript         map[string]*ResourceGlobalInitScript         `json:"databricks_global_init_script,omitempty"`
	Grants                   map[string]*ResourceGrants                   `json:"databricks_grants,omitempty"`
	Group                    map[string]*ResourceGroup                    `json:"databricks_group,omitempty"`
	GroupInstanceProfile     map[string]*ResourceGroupInstanceProfile     `json:"databricks_group_instance_profile,omitempty"`
	GroupMember              map[string]*ResourceGroupMember              `json:"databricks_group_member,omitempty"`
	GroupRole                map[string]*ResourceGroupRole                `json:"databricks_group_role,omitempty"`
	InstancePool             map[string]*ResourceInstancePool             `json:"databricks_instance_pool,omitempty"`
	InstanceProfile          map[string]*ResourceInstanceProfile          `json:"databricks_instance_profile,omitempty"`
	IpAccessList             map[string]*ResourceIpAccessList             `json:"databricks_ip_access_list,omitempty"`
	Job                      map[string]*ResourceJob                      `json:"databricks_job,omitempty"`
	Library                  map[string]*ResourceLibrary                  `json:"databricks_library,omitempty"`
	Metastore                map[string]*ResourceMetastore                `json:"databricks_metastore,omitempty"`
	MetastoreAssignment      map[string]*ResourceMetastoreAssignment      `json:"databricks_metastore_assignment,omitempty"`
	MetastoreDataAccess      map[string]*ResourceMetastoreDataAccess      `json:"databricks_metastore_data_access,omitempty"`
	MlflowExperiment         map[string]*ResourceMlflowExperiment         `json:"databricks_mlflow_experiment,omitempty"`
	MlflowModel              map[string]*ResourceMlflowModel              `json:"databricks_mlflow_model,omitempty"`
	MlflowWebhook            map[string]*ResourceMlflowWebhook            `json:"databricks_mlflow_webhook,omitempty"`
	Mount                    map[string]*ResourceMount                    `json:"databricks_mount,omitempty"`
	MwsCredentials           map[string]*ResourceMwsCredentials           `json:"databricks_mws_credentials,omitempty"`
	MwsCustomerManagedKeys   map[string]*ResourceMwsCustomerManagedKeys   `json:"databricks_mws_customer_managed_keys,omitempty"`
	MwsLogDelivery           map[string]*ResourceMwsLogDelivery           `json:"databricks_mws_log_delivery,omitempty"`
	MwsNetworks              map[string]*ResourceMwsNetworks              `json:"databricks_mws_networks,omitempty"`
	MwsPermissionAssignment  map[string]*ResourceMwsPermissionAssignment  `json:"databricks_mws_permission_assignment,omitempty"`
	MwsPrivateAccessSettings map[string]*ResourceMwsPrivateAccessSettings `json:"databricks_mws_private_access_settings,omitempty"`
	MwsStorageConfigurations map[string]*ResourceMwsStorageConfigurations `json:"databricks_mws_storage_configurations,omitempty"`
	MwsVpcEndpoint           map[string]*ResourceMwsVpcEndpoint           `json:"databricks_mws_vpc_endpoint,omitempty"`
	MwsWorkspaces            map[string]*ResourceMwsWorkspaces            `json:"databricks_mws_workspaces,omitempty"`
	Notebook                 map[string]*ResourceNotebook                 `json:"databricks_notebook,omitempty"`
	OboToken                 map[string]*ResourceOboToken                 `json:"databricks_obo_token,omitempty"`
	PermissionAssignment     map[string]*ResourcePermissionAssignment     `json:"databricks_permission_assignment,omitempty"`
	Permissions              map[string]*ResourcePermissions              `json:"databricks_permissions,omitempty"`
	Pipeline                 map[string]*ResourcePipeline                 `json:"databricks_pipeline,omitempty"`
	Recipient                map[string]*ResourceRecipient                `json:"databricks_recipient,omitempty"`
	Repo                     map[string]*ResourceRepo                     `json:"databricks_repo,omitempty"`
	Schema                   map[string]*ResourceSchema                   `json:"databricks_schema,omitempty"`
	Secret                   map[string]*ResourceSecret                   `json:"databricks_secret,omitempty"`
	SecretAcl                map[string]*ResourceSecretAcl                `json:"databricks_secret_acl,omitempty"`
	SecretScope              map[string]*ResourceSecretScope              `json:"databricks_secret_scope,omitempty"`
	ServicePrincipal         map[string]*ResourceServicePrincipal         `json:"databricks_service_principal,omitempty"`
	ServicePrincipalRole     map[string]*ResourceServicePrincipalRole     `json:"databricks_service_principal_role,omitempty"`
	ServicePrincipalSecret   map[string]*ResourceServicePrincipalSecret   `json:"databricks_service_principal_secret,omitempty"`
	Share                    map[string]*ResourceShare                    `json:"databricks_share,omitempty"`
	SqlDashboard             map[string]*ResourceSqlDashboard             `json:"databricks_sql_dashboard,omitempty"`
	SqlEndpoint              map[string]*ResourceSqlEndpoint              `json:"databricks_sql_endpoint,omitempty"`
	SqlGlobalConfig          map[string]*ResourceSqlGlobalConfig          `json:"databricks_sql_global_config,omitempty"`
	SqlPermissions           map[string]*ResourceSqlPermissions           `json:"databricks_sql_permissions,omitempty"`
	SqlQuery                 map[string]*ResourceSqlQuery                 `json:"databricks_sql_query,omitempty"`
	SqlVisualization         map[string]*ResourceSqlVisualization         `json:"databricks_sql_visualization,omitempty"`
	SqlWidget                map[string]*ResourceSqlWidget                `json:"databricks_sql_widget,omitempty"`
	StorageCredential        map[string]*ResourceStorageCredential        `json:"databricks_storage_credential,omitempty"`
	Table                    map[string]*ResourceTable                    `json:"databricks_table,omitempty"`
	Token                    map[string]*ResourceToken                    `json:"databricks_token,omitempty"`
	User                     map[string]*ResourceUser                     `json:"databricks_user,omitempty"`
	UserInstanceProfile      map[string]*ResourceUserInstanceProfile      `json:"databricks_user_instance_profile,omitempty"`
	UserRole                 map[string]*ResourceUserRole                 `json:"databricks_user_role,omitempty"`
	WorkspaceConf            map[string]*ResourceWorkspaceConf            `json:"databricks_workspace_conf,omitempty"`
}

func NewResources() *Resources {
	return &Resources{
		AwsS3Mount:               make(map[string]*ResourceAwsS3Mount),
		AzureAdlsGen1Mount:       make(map[string]*ResourceAzureAdlsGen1Mount),
		AzureAdlsGen2Mount:       make(map[string]*ResourceAzureAdlsGen2Mount),
		AzureBlobMount:           make(map[string]*ResourceAzureBlobMount),
		Catalog:                  make(map[string]*ResourceCatalog),
		Cluster:                  make(map[string]*ResourceCluster),
		ClusterPolicy:            make(map[string]*ResourceClusterPolicy),
		DbfsFile:                 make(map[string]*ResourceDbfsFile),
		Directory:                make(map[string]*ResourceDirectory),
		Entitlements:             make(map[string]*ResourceEntitlements),
		ExternalLocation:         make(map[string]*ResourceExternalLocation),
		GitCredential:            make(map[string]*ResourceGitCredential),
		GlobalInitScript:         make(map[string]*ResourceGlobalInitScript),
		Grants:                   make(map[string]*ResourceGrants),
		Group:                    make(map[string]*ResourceGroup),
		GroupInstanceProfile:     make(map[string]*ResourceGroupInstanceProfile),
		GroupMember:              make(map[string]*ResourceGroupMember),
		GroupRole:                make(map[string]*ResourceGroupRole),
		InstancePool:             make(map[string]*ResourceInstancePool),
		InstanceProfile:          make(map[string]*ResourceInstanceProfile),
		IpAccessList:             make(map[string]*ResourceIpAccessList),
		Job:                      make(map[string]*ResourceJob),
		Library:                  make(map[string]*ResourceLibrary),
		Metastore:                make(map[string]*ResourceMetastore),
		MetastoreAssignment:      make(map[string]*ResourceMetastoreAssignment),
		MetastoreDataAccess:      make(map[string]*ResourceMetastoreDataAccess),
		MlflowExperiment:         make(map[string]*ResourceMlflowExperiment),
		MlflowModel:              make(map[string]*ResourceMlflowModel),
		MlflowWebhook:            make(map[string]*ResourceMlflowWebhook),
		Mount:                    make(map[string]*ResourceMount),
		MwsCredentials:           make(map[string]*ResourceMwsCredentials),
		MwsCustomerManagedKeys:   make(map[string]*ResourceMwsCustomerManagedKeys),
		MwsLogDelivery:           make(map[string]*ResourceMwsLogDelivery),
		MwsNetworks:              make(map[string]*ResourceMwsNetworks),
		MwsPermissionAssignment:  make(map[string]*ResourceMwsPermissionAssignment),
		MwsPrivateAccessSettings: make(map[string]*ResourceMwsPrivateAccessSettings),
		MwsStorageConfigurations: make(map[string]*ResourceMwsStorageConfigurations),
		MwsVpcEndpoint:           make(map[string]*ResourceMwsVpcEndpoint),
		MwsWorkspaces:            make(map[string]*ResourceMwsWorkspaces),
		Notebook:                 make(map[string]*ResourceNotebook),
		OboToken:                 make(map[string]*ResourceOboToken),
		PermissionAssignment:     make(map[string]*ResourcePermissionAssignment),
		Permissions:              make(map[string]*ResourcePermissions),
		Pipeline:                 make(map[string]*ResourcePipeline),
		Recipient:                make(map[string]*ResourceRecipient),
		Repo:                     make(map[string]*ResourceRepo),
		Schema:                   make(map[string]*ResourceSchema),
		Secret:                   make(map[string]*ResourceSecret),
		SecretAcl:                make(map[string]*ResourceSecretAcl),
		SecretScope:              make(map[string]*ResourceSecretScope),
		ServicePrincipal:         make(map[string]*ResourceServicePrincipal),
		ServicePrincipalRole:     make(map[string]*ResourceServicePrincipalRole),
		ServicePrincipalSecret:   make(map[string]*ResourceServicePrincipalSecret),
		Share:                    make(map[string]*ResourceShare),
		SqlDashboard:             make(map[string]*ResourceSqlDashboard),
		SqlEndpoint:              make(map[string]*ResourceSqlEndpoint),
		SqlGlobalConfig:          make(map[string]*ResourceSqlGlobalConfig),
		SqlPermissions:           make(map[string]*ResourceSqlPermissions),
		SqlQuery:                 make(map[string]*ResourceSqlQuery),
		SqlVisualization:         make(map[string]*ResourceSqlVisualization),
		SqlWidget:                make(map[string]*ResourceSqlWidget),
		StorageCredential:        make(map[string]*ResourceStorageCredential),
		Table:                    make(map[string]*ResourceTable),
		Token:                    make(map[string]*ResourceToken),
		User:                     make(map[string]*ResourceUser),
		UserInstanceProfile:      make(map[string]*ResourceUserInstanceProfile),
		UserRole:                 make(map[string]*ResourceUserRole),
		WorkspaceConf:            make(map[string]*ResourceWorkspaceConf),
	}
}
