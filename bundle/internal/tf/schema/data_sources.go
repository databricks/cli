// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSources struct {
	AwsAssumeRolePolicy   map[string]any `json:"databricks_aws_assume_role_policy,omitempty"`
	AwsBucketPolicy       map[string]any `json:"databricks_aws_bucket_policy,omitempty"`
	AwsCrossaccountPolicy map[string]any `json:"databricks_aws_crossaccount_policy,omitempty"`
	AwsUnityCatalogPolicy map[string]any `json:"databricks_aws_unity_catalog_policy,omitempty"`
	Catalogs              map[string]any `json:"databricks_catalogs,omitempty"`
	Cluster               map[string]any `json:"databricks_cluster,omitempty"`
	ClusterPolicy         map[string]any `json:"databricks_cluster_policy,omitempty"`
	Clusters              map[string]any `json:"databricks_clusters,omitempty"`
	CurrentConfig         map[string]any `json:"databricks_current_config,omitempty"`
	CurrentMetastore      map[string]any `json:"databricks_current_metastore,omitempty"`
	CurrentUser           map[string]any `json:"databricks_current_user,omitempty"`
	DbfsFile              map[string]any `json:"databricks_dbfs_file,omitempty"`
	DbfsFilePaths         map[string]any `json:"databricks_dbfs_file_paths,omitempty"`
	Directory             map[string]any `json:"databricks_directory,omitempty"`
	Group                 map[string]any `json:"databricks_group,omitempty"`
	InstancePool          map[string]any `json:"databricks_instance_pool,omitempty"`
	InstanceProfiles      map[string]any `json:"databricks_instance_profiles,omitempty"`
	Job                   map[string]any `json:"databricks_job,omitempty"`
	Jobs                  map[string]any `json:"databricks_jobs,omitempty"`
	Metastore             map[string]any `json:"databricks_metastore,omitempty"`
	Metastores            map[string]any `json:"databricks_metastores,omitempty"`
	MlflowModel           map[string]any `json:"databricks_mlflow_model,omitempty"`
	MwsCredentials        map[string]any `json:"databricks_mws_credentials,omitempty"`
	MwsWorkspaces         map[string]any `json:"databricks_mws_workspaces,omitempty"`
	NodeType              map[string]any `json:"databricks_node_type,omitempty"`
	Notebook              map[string]any `json:"databricks_notebook,omitempty"`
	NotebookPaths         map[string]any `json:"databricks_notebook_paths,omitempty"`
	Pipelines             map[string]any `json:"databricks_pipelines,omitempty"`
	Schemas               map[string]any `json:"databricks_schemas,omitempty"`
	ServicePrincipal      map[string]any `json:"databricks_service_principal,omitempty"`
	ServicePrincipals     map[string]any `json:"databricks_service_principals,omitempty"`
	Share                 map[string]any `json:"databricks_share,omitempty"`
	Shares                map[string]any `json:"databricks_shares,omitempty"`
	SparkVersion          map[string]any `json:"databricks_spark_version,omitempty"`
	SqlWarehouse          map[string]any `json:"databricks_sql_warehouse,omitempty"`
	SqlWarehouses         map[string]any `json:"databricks_sql_warehouses,omitempty"`
	Tables                map[string]any `json:"databricks_tables,omitempty"`
	User                  map[string]any `json:"databricks_user,omitempty"`
	Views                 map[string]any `json:"databricks_views,omitempty"`
	Volumes               map[string]any `json:"databricks_volumes,omitempty"`
	Zones                 map[string]any `json:"databricks_zones,omitempty"`
}

func NewDataSources() *DataSources {
	return &DataSources{
		AwsAssumeRolePolicy:   make(map[string]any),
		AwsBucketPolicy:       make(map[string]any),
		AwsCrossaccountPolicy: make(map[string]any),
		AwsUnityCatalogPolicy: make(map[string]any),
		Catalogs:              make(map[string]any),
		Cluster:               make(map[string]any),
		ClusterPolicy:         make(map[string]any),
		Clusters:              make(map[string]any),
		CurrentConfig:         make(map[string]any),
		CurrentMetastore:      make(map[string]any),
		CurrentUser:           make(map[string]any),
		DbfsFile:              make(map[string]any),
		DbfsFilePaths:         make(map[string]any),
		Directory:             make(map[string]any),
		Group:                 make(map[string]any),
		InstancePool:          make(map[string]any),
		InstanceProfiles:      make(map[string]any),
		Job:                   make(map[string]any),
		Jobs:                  make(map[string]any),
		Metastore:             make(map[string]any),
		Metastores:            make(map[string]any),
		MlflowModel:           make(map[string]any),
		MwsCredentials:        make(map[string]any),
		MwsWorkspaces:         make(map[string]any),
		NodeType:              make(map[string]any),
		Notebook:              make(map[string]any),
		NotebookPaths:         make(map[string]any),
		Pipelines:             make(map[string]any),
		Schemas:               make(map[string]any),
		ServicePrincipal:      make(map[string]any),
		ServicePrincipals:     make(map[string]any),
		Share:                 make(map[string]any),
		Shares:                make(map[string]any),
		SparkVersion:          make(map[string]any),
		SqlWarehouse:          make(map[string]any),
		SqlWarehouses:         make(map[string]any),
		Tables:                make(map[string]any),
		User:                  make(map[string]any),
		Views:                 make(map[string]any),
		Volumes:               make(map[string]any),
		Zones:                 make(map[string]any),
	}
}
