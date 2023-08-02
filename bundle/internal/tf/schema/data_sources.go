// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSources struct {
	AwsAssumeRolePolicy   map[string]*DataSourceAwsAssumeRolePolicy   `json:"databricks_aws_assume_role_policy,omitempty"`
	AwsBucketPolicy       map[string]*DataSourceAwsBucketPolicy       `json:"databricks_aws_bucket_policy,omitempty"`
	AwsCrossaccountPolicy map[string]*DataSourceAwsCrossaccountPolicy `json:"databricks_aws_crossaccount_policy,omitempty"`
	Catalogs              map[string]*DataSourceCatalogs              `json:"databricks_catalogs,omitempty"`
	Cluster               map[string]*DataSourceCluster               `json:"databricks_cluster,omitempty"`
	ClusterPolicy         map[string]*DataSourceClusterPolicy         `json:"databricks_cluster_policy,omitempty"`
	Clusters              map[string]*DataSourceClusters              `json:"databricks_clusters,omitempty"`
	CurrentUser           map[string]*DataSourceCurrentUser           `json:"databricks_current_user,omitempty"`
	DbfsFile              map[string]*DataSourceDbfsFile              `json:"databricks_dbfs_file,omitempty"`
	DbfsFilePaths         map[string]*DataSourceDbfsFilePaths         `json:"databricks_dbfs_file_paths,omitempty"`
	Directory             map[string]*DataSourceDirectory             `json:"databricks_directory,omitempty"`
	Group                 map[string]*DataSourceGroup                 `json:"databricks_group,omitempty"`
	InstancePool          map[string]*DataSourceInstancePool          `json:"databricks_instance_pool,omitempty"`
	Job                   map[string]*DataSourceJob                   `json:"databricks_job,omitempty"`
	Jobs                  map[string]*DataSourceJobs                  `json:"databricks_jobs,omitempty"`
	Metastore             map[string]*DataSourceMetastore             `json:"databricks_metastore,omitempty"`
	Metastores            map[string]*DataSourceMetastores            `json:"databricks_metastores,omitempty"`
	MwsCredentials        map[string]*DataSourceMwsCredentials        `json:"databricks_mws_credentials,omitempty"`
	MwsWorkspaces         map[string]*DataSourceMwsWorkspaces         `json:"databricks_mws_workspaces,omitempty"`
	NodeType              map[string]*DataSourceNodeType              `json:"databricks_node_type,omitempty"`
	Notebook              map[string]*DataSourceNotebook              `json:"databricks_notebook,omitempty"`
	NotebookPaths         map[string]*DataSourceNotebookPaths         `json:"databricks_notebook_paths,omitempty"`
	Pipelines             map[string]*DataSourcePipelines             `json:"databricks_pipelines,omitempty"`
	Schemas               map[string]*DataSourceSchemas               `json:"databricks_schemas,omitempty"`
	ServicePrincipal      map[string]*DataSourceServicePrincipal      `json:"databricks_service_principal,omitempty"`
	ServicePrincipals     map[string]*DataSourceServicePrincipals     `json:"databricks_service_principals,omitempty"`
	Share                 map[string]*DataSourceShare                 `json:"databricks_share,omitempty"`
	Shares                map[string]*DataSourceShares                `json:"databricks_shares,omitempty"`
	SparkVersion          map[string]*DataSourceSparkVersion          `json:"databricks_spark_version,omitempty"`
	SqlWarehouse          map[string]*DataSourceSqlWarehouse          `json:"databricks_sql_warehouse,omitempty"`
	SqlWarehouses         map[string]*DataSourceSqlWarehouses         `json:"databricks_sql_warehouses,omitempty"`
	Tables                map[string]*DataSourceTables                `json:"databricks_tables,omitempty"`
	User                  map[string]*DataSourceUser                  `json:"databricks_user,omitempty"`
	Views                 map[string]*DataSourceViews                 `json:"databricks_views,omitempty"`
	Zones                 map[string]*DataSourceZones                 `json:"databricks_zones,omitempty"`
}

func NewDataSources() *DataSources {
	return &DataSources{
		AwsAssumeRolePolicy:   make(map[string]*DataSourceAwsAssumeRolePolicy),
		AwsBucketPolicy:       make(map[string]*DataSourceAwsBucketPolicy),
		AwsCrossaccountPolicy: make(map[string]*DataSourceAwsCrossaccountPolicy),
		Catalogs:              make(map[string]*DataSourceCatalogs),
		Cluster:               make(map[string]*DataSourceCluster),
		ClusterPolicy:         make(map[string]*DataSourceClusterPolicy),
		Clusters:              make(map[string]*DataSourceClusters),
		CurrentUser:           make(map[string]*DataSourceCurrentUser),
		DbfsFile:              make(map[string]*DataSourceDbfsFile),
		DbfsFilePaths:         make(map[string]*DataSourceDbfsFilePaths),
		Directory:             make(map[string]*DataSourceDirectory),
		Group:                 make(map[string]*DataSourceGroup),
		InstancePool:          make(map[string]*DataSourceInstancePool),
		Job:                   make(map[string]*DataSourceJob),
		Jobs:                  make(map[string]*DataSourceJobs),
		Metastore:             make(map[string]*DataSourceMetastore),
		Metastores:            make(map[string]*DataSourceMetastores),
		MwsCredentials:        make(map[string]*DataSourceMwsCredentials),
		MwsWorkspaces:         make(map[string]*DataSourceMwsWorkspaces),
		NodeType:              make(map[string]*DataSourceNodeType),
		Notebook:              make(map[string]*DataSourceNotebook),
		NotebookPaths:         make(map[string]*DataSourceNotebookPaths),
		Pipelines:             make(map[string]*DataSourcePipelines),
		Schemas:               make(map[string]*DataSourceSchemas),
		ServicePrincipal:      make(map[string]*DataSourceServicePrincipal),
		ServicePrincipals:     make(map[string]*DataSourceServicePrincipals),
		Share:                 make(map[string]*DataSourceShare),
		Shares:                make(map[string]*DataSourceShares),
		SparkVersion:          make(map[string]*DataSourceSparkVersion),
		SqlWarehouse:          make(map[string]*DataSourceSqlWarehouse),
		SqlWarehouses:         make(map[string]*DataSourceSqlWarehouses),
		Tables:                make(map[string]*DataSourceTables),
		User:                  make(map[string]*DataSourceUser),
		Views:                 make(map[string]*DataSourceViews),
		Zones:                 make(map[string]*DataSourceZones),
	}
}
