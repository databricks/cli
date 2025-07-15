// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePipelineClusterAutoscale struct {
	MaxWorkers int    `json:"max_workers"`
	MinWorkers int    `json:"min_workers"`
	Mode       string `json:"mode,omitempty"`
}

type ResourcePipelineClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeIops       int    `json:"ebs_volume_iops,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeThroughput int    `json:"ebs_volume_throughput,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourcePipelineClusterAzureAttributesLogAnalyticsInfo struct {
	LogAnalyticsPrimaryKey  string `json:"log_analytics_primary_key,omitempty"`
	LogAnalyticsWorkspaceId string `json:"log_analytics_workspace_id,omitempty"`
}

type ResourcePipelineClusterAzureAttributes struct {
	Availability     string                                                  `json:"availability,omitempty"`
	FirstOnDemand    int                                                     `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice  int                                                     `json:"spot_bid_max_price,omitempty"`
	LogAnalyticsInfo *ResourcePipelineClusterAzureAttributesLogAnalyticsInfo `json:"log_analytics_info,omitempty"`
}

type ResourcePipelineClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourcePipelineClusterClusterLogConfVolumes struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterClusterLogConf struct {
	Dbfs    *ResourcePipelineClusterClusterLogConfDbfs    `json:"dbfs,omitempty"`
	S3      *ResourcePipelineClusterClusterLogConfS3      `json:"s3,omitempty"`
	Volumes *ResourcePipelineClusterClusterLogConfVolumes `json:"volumes,omitempty"`
}

type ResourcePipelineClusterGcpAttributes struct {
	Availability         string `json:"availability,omitempty"`
	GoogleServiceAccount string `json:"google_service_account,omitempty"`
	LocalSsdCount        int    `json:"local_ssd_count,omitempty"`
	ZoneId               string `json:"zone_id,omitempty"`
}

type ResourcePipelineClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourcePipelineClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScripts struct {
	Abfss     *ResourcePipelineClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourcePipelineClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourcePipelineClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourcePipelineClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourcePipelineClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourcePipelineClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourcePipelineClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourcePipelineCluster struct {
	ApplyPolicyDefaultValues  bool                                    `json:"apply_policy_default_values,omitempty"`
	CustomTags                map[string]string                       `json:"custom_tags,omitempty"`
	DriverInstancePoolId      string                                  `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                  `json:"driver_node_type_id,omitempty"`
	EnableLocalDiskEncryption bool                                    `json:"enable_local_disk_encryption,omitempty"`
	InstancePoolId            string                                  `json:"instance_pool_id,omitempty"`
	Label                     string                                  `json:"label,omitempty"`
	NodeTypeId                string                                  `json:"node_type_id,omitempty"`
	NumWorkers                int                                     `json:"num_workers,omitempty"`
	PolicyId                  string                                  `json:"policy_id,omitempty"`
	SparkConf                 map[string]string                       `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                       `json:"spark_env_vars,omitempty"`
	SshPublicKeys             []string                                `json:"ssh_public_keys,omitempty"`
	Autoscale                 *ResourcePipelineClusterAutoscale       `json:"autoscale,omitempty"`
	AwsAttributes             *ResourcePipelineClusterAwsAttributes   `json:"aws_attributes,omitempty"`
	AzureAttributes           *ResourcePipelineClusterAzureAttributes `json:"azure_attributes,omitempty"`
	ClusterLogConf            *ResourcePipelineClusterClusterLogConf  `json:"cluster_log_conf,omitempty"`
	GcpAttributes             *ResourcePipelineClusterGcpAttributes   `json:"gcp_attributes,omitempty"`
	InitScripts               []ResourcePipelineClusterInitScripts    `json:"init_scripts,omitempty"`
}

type ResourcePipelineDeployment struct {
	Kind             string `json:"kind"`
	MetadataFilePath string `json:"metadata_file_path,omitempty"`
}

type ResourcePipelineEnvironment struct {
	Dependencies []string `json:"dependencies,omitempty"`
}

type ResourcePipelineEventLog struct {
	Catalog string `json:"catalog,omitempty"`
	Name    string `json:"name"`
	Schema  string `json:"schema,omitempty"`
}

type ResourcePipelineFilters struct {
	Exclude []string `json:"exclude,omitempty"`
	Include []string `json:"include,omitempty"`
}

type ResourcePipelineGatewayDefinition struct {
	ConnectionId          string `json:"connection_id,omitempty"`
	ConnectionName        string `json:"connection_name"`
	GatewayStorageCatalog string `json:"gateway_storage_catalog"`
	GatewayStorageName    string `json:"gateway_storage_name,omitempty"`
	GatewayStorageSchema  string `json:"gateway_storage_schema"`
}

type ResourcePipelineIngestionDefinitionObjectsReportTableConfiguration struct {
	ExcludeColumns                 []string `json:"exclude_columns,omitempty"`
	IncludeColumns                 []string `json:"include_columns,omitempty"`
	PrimaryKeys                    []string `json:"primary_keys,omitempty"`
	SalesforceIncludeFormulaFields bool     `json:"salesforce_include_formula_fields,omitempty"`
	ScdType                        string   `json:"scd_type,omitempty"`
	SequenceBy                     []string `json:"sequence_by,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjectsReport struct {
	DestinationCatalog string                                                              `json:"destination_catalog"`
	DestinationSchema  string                                                              `json:"destination_schema"`
	DestinationTable   string                                                              `json:"destination_table,omitempty"`
	SourceUrl          string                                                              `json:"source_url"`
	TableConfiguration *ResourcePipelineIngestionDefinitionObjectsReportTableConfiguration `json:"table_configuration,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjectsSchemaTableConfiguration struct {
	ExcludeColumns                 []string `json:"exclude_columns,omitempty"`
	IncludeColumns                 []string `json:"include_columns,omitempty"`
	PrimaryKeys                    []string `json:"primary_keys,omitempty"`
	SalesforceIncludeFormulaFields bool     `json:"salesforce_include_formula_fields,omitempty"`
	ScdType                        string   `json:"scd_type,omitempty"`
	SequenceBy                     []string `json:"sequence_by,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjectsSchema struct {
	DestinationCatalog string                                                              `json:"destination_catalog"`
	DestinationSchema  string                                                              `json:"destination_schema"`
	SourceCatalog      string                                                              `json:"source_catalog,omitempty"`
	SourceSchema       string                                                              `json:"source_schema"`
	TableConfiguration *ResourcePipelineIngestionDefinitionObjectsSchemaTableConfiguration `json:"table_configuration,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjectsTableTableConfiguration struct {
	ExcludeColumns                 []string `json:"exclude_columns,omitempty"`
	IncludeColumns                 []string `json:"include_columns,omitempty"`
	PrimaryKeys                    []string `json:"primary_keys,omitempty"`
	SalesforceIncludeFormulaFields bool     `json:"salesforce_include_formula_fields,omitempty"`
	ScdType                        string   `json:"scd_type,omitempty"`
	SequenceBy                     []string `json:"sequence_by,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjectsTable struct {
	DestinationCatalog string                                                             `json:"destination_catalog"`
	DestinationSchema  string                                                             `json:"destination_schema"`
	DestinationTable   string                                                             `json:"destination_table,omitempty"`
	SourceCatalog      string                                                             `json:"source_catalog,omitempty"`
	SourceSchema       string                                                             `json:"source_schema,omitempty"`
	SourceTable        string                                                             `json:"source_table"`
	TableConfiguration *ResourcePipelineIngestionDefinitionObjectsTableTableConfiguration `json:"table_configuration,omitempty"`
}

type ResourcePipelineIngestionDefinitionObjects struct {
	Report *ResourcePipelineIngestionDefinitionObjectsReport `json:"report,omitempty"`
	Schema *ResourcePipelineIngestionDefinitionObjectsSchema `json:"schema,omitempty"`
	Table  *ResourcePipelineIngestionDefinitionObjectsTable  `json:"table,omitempty"`
}

type ResourcePipelineIngestionDefinitionTableConfiguration struct {
	ExcludeColumns                 []string `json:"exclude_columns,omitempty"`
	IncludeColumns                 []string `json:"include_columns,omitempty"`
	PrimaryKeys                    []string `json:"primary_keys,omitempty"`
	SalesforceIncludeFormulaFields bool     `json:"salesforce_include_formula_fields,omitempty"`
	ScdType                        string   `json:"scd_type,omitempty"`
	SequenceBy                     []string `json:"sequence_by,omitempty"`
}

type ResourcePipelineIngestionDefinition struct {
	ConnectionName     string                                                 `json:"connection_name,omitempty"`
	IngestionGatewayId string                                                 `json:"ingestion_gateway_id,omitempty"`
	SourceType         string                                                 `json:"source_type,omitempty"`
	Objects            []ResourcePipelineIngestionDefinitionObjects           `json:"objects,omitempty"`
	TableConfiguration *ResourcePipelineIngestionDefinitionTableConfiguration `json:"table_configuration,omitempty"`
}

type ResourcePipelineLatestUpdates struct {
	CreationTime string `json:"creation_time,omitempty"`
	State        string `json:"state,omitempty"`
	UpdateId     string `json:"update_id,omitempty"`
}

type ResourcePipelineLibraryFile struct {
	Path string `json:"path"`
}

type ResourcePipelineLibraryGlob struct {
	Include string `json:"include"`
}

type ResourcePipelineLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourcePipelineLibraryNotebook struct {
	Path string `json:"path"`
}

type ResourcePipelineLibrary struct {
	Jar      string                           `json:"jar,omitempty"`
	Whl      string                           `json:"whl,omitempty"`
	File     *ResourcePipelineLibraryFile     `json:"file,omitempty"`
	Glob     *ResourcePipelineLibraryGlob     `json:"glob,omitempty"`
	Maven    *ResourcePipelineLibraryMaven    `json:"maven,omitempty"`
	Notebook *ResourcePipelineLibraryNotebook `json:"notebook,omitempty"`
}

type ResourcePipelineNotification struct {
	Alerts          []string `json:"alerts,omitempty"`
	EmailRecipients []string `json:"email_recipients,omitempty"`
}

type ResourcePipelineRestartWindow struct {
	DaysOfWeek []string `json:"days_of_week,omitempty"`
	StartHour  int      `json:"start_hour"`
	TimeZoneId string   `json:"time_zone_id,omitempty"`
}

type ResourcePipelineRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type ResourcePipelineTriggerCron struct {
	QuartzCronSchedule string `json:"quartz_cron_schedule,omitempty"`
	TimezoneId         string `json:"timezone_id,omitempty"`
}

type ResourcePipelineTriggerManual struct {
}

type ResourcePipelineTrigger struct {
	Cron   *ResourcePipelineTriggerCron   `json:"cron,omitempty"`
	Manual *ResourcePipelineTriggerManual `json:"manual,omitempty"`
}

type ResourcePipeline struct {
	AllowDuplicateNames  bool                                 `json:"allow_duplicate_names,omitempty"`
	BudgetPolicyId       string                               `json:"budget_policy_id,omitempty"`
	Catalog              string                               `json:"catalog,omitempty"`
	Cause                string                               `json:"cause,omitempty"`
	Channel              string                               `json:"channel,omitempty"`
	ClusterId            string                               `json:"cluster_id,omitempty"`
	Configuration        map[string]string                    `json:"configuration,omitempty"`
	Continuous           bool                                 `json:"continuous,omitempty"`
	CreatorUserName      string                               `json:"creator_user_name,omitempty"`
	Development          bool                                 `json:"development,omitempty"`
	Edition              string                               `json:"edition,omitempty"`
	ExpectedLastModified int                                  `json:"expected_last_modified,omitempty"`
	Health               string                               `json:"health,omitempty"`
	Id                   string                               `json:"id,omitempty"`
	LastModified         int                                  `json:"last_modified,omitempty"`
	Name                 string                               `json:"name,omitempty"`
	Photon               bool                                 `json:"photon,omitempty"`
	RootPath             string                               `json:"root_path,omitempty"`
	RunAsUserName        string                               `json:"run_as_user_name,omitempty"`
	Schema               string                               `json:"schema,omitempty"`
	Serverless           bool                                 `json:"serverless,omitempty"`
	State                string                               `json:"state,omitempty"`
	Storage              string                               `json:"storage,omitempty"`
	Tags                 map[string]string                    `json:"tags,omitempty"`
	Target               string                               `json:"target,omitempty"`
	Url                  string                               `json:"url,omitempty"`
	Cluster              []ResourcePipelineCluster            `json:"cluster,omitempty"`
	Deployment           *ResourcePipelineDeployment          `json:"deployment,omitempty"`
	Environment          *ResourcePipelineEnvironment         `json:"environment,omitempty"`
	EventLog             *ResourcePipelineEventLog            `json:"event_log,omitempty"`
	Filters              *ResourcePipelineFilters             `json:"filters,omitempty"`
	GatewayDefinition    *ResourcePipelineGatewayDefinition   `json:"gateway_definition,omitempty"`
	IngestionDefinition  *ResourcePipelineIngestionDefinition `json:"ingestion_definition,omitempty"`
	LatestUpdates        []ResourcePipelineLatestUpdates      `json:"latest_updates,omitempty"`
	Library              []ResourcePipelineLibrary            `json:"library,omitempty"`
	Notification         []ResourcePipelineNotification       `json:"notification,omitempty"`
	RestartWindow        *ResourcePipelineRestartWindow       `json:"restart_window,omitempty"`
	RunAs                *ResourcePipelineRunAs               `json:"run_as,omitempty"`
	Trigger              *ResourcePipelineTrigger             `json:"trigger,omitempty"`
}
