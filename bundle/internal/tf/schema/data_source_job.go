// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceJobJobSettingsSettingsContinuous struct {
	PauseStatus string `json:"pause_status,omitempty"`
}

type DataSourceJobJobSettingsSettingsDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsEmailNotifications struct {
	AlertOnLastAttempt    bool     `json:"alert_on_last_attempt,omitempty"`
	NoAlertForSkippedRuns bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnFailure             []string `json:"on_failure,omitempty"`
	OnStart               []string `json:"on_start,omitempty"`
	OnSuccess             []string `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsGitSource struct {
	Branch   string `json:"branch,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Provider string `json:"provider,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Url      string `json:"url"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConf struct {
	Dbfs *DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                                     `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                                     `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterDockerImage struct {
	Url       string                                                                    `json:"url"`
	BasicAuth *DataSourceJobJobSettingsSettingsJobClusterNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsFile struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsGcs struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Workspace *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterWorkloadType struct {
	Clients *DataSourceJobJobSettingsSettingsJobClusterNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewCluster struct {
	ApplyPolicyDefaultValues  bool                                                                   `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                                                    `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                                                 `json:"cluster_id,omitempty"`
	ClusterName               string                                                                 `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                                      `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                                                 `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                                                 `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                                                 `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                                                   `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                                                   `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                                                 `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                                                 `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                                                 `json:"node_type_id,omitempty"`
	NumWorkers                int                                                                    `json:"num_workers"`
	PolicyId                  string                                                                 `json:"policy_id,omitempty"`
	RuntimeEngine             string                                                                 `json:"runtime_engine,omitempty"`
	SingleUserName            string                                                                 `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                                      `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                                      `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                                                 `json:"spark_version"`
	SshPublicKeys             []string                                                               `json:"ssh_public_keys,omitempty"`
	Autoscale                 *DataSourceJobJobSettingsSettingsJobClusterNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *DataSourceJobJobSettingsSettingsJobClusterNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *DataSourceJobJobSettingsSettingsJobClusterNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []DataSourceJobJobSettingsSettingsJobClusterNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *DataSourceJobJobSettingsSettingsJobClusterNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *DataSourceJobJobSettingsSettingsJobClusterNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *DataSourceJobJobSettingsSettingsJobClusterNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobCluster struct {
	JobClusterKey string                                                `json:"job_cluster_key,omitempty"`
	NewCluster    *DataSourceJobJobSettingsSettingsJobClusterNewCluster `json:"new_cluster,omitempty"`
}

type DataSourceJobJobSettingsSettingsLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsLibrary struct {
	Egg   string                                        `json:"egg,omitempty"`
	Jar   string                                        `json:"jar,omitempty"`
	Whl   string                                        `json:"whl,omitempty"`
	Cran  *DataSourceJobJobSettingsSettingsLibraryCran  `json:"cran,omitempty"`
	Maven *DataSourceJobJobSettingsSettingsLibraryMaven `json:"maven,omitempty"`
	Pypi  *DataSourceJobJobSettingsSettingsLibraryPypi  `json:"pypi,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterClusterLogConf struct {
	Dbfs *DataSourceJobJobSettingsSettingsNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *DataSourceJobJobSettingsSettingsNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type DataSourceJobJobSettingsSettingsNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                           `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                           `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *DataSourceJobJobSettingsSettingsNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceJobJobSettingsSettingsNewClusterDockerImage struct {
	Url       string                                                          `json:"url"`
	BasicAuth *DataSourceJobJobSettingsSettingsNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsFile struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsGcs struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Workspace *DataSourceJobJobSettingsSettingsNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterWorkloadType struct {
	Clients *DataSourceJobJobSettingsSettingsNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewCluster struct {
	ApplyPolicyDefaultValues  bool                                                         `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                                          `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                                       `json:"cluster_id,omitempty"`
	ClusterName               string                                                       `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                            `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                                       `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                                       `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                                       `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                                         `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                                         `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                                       `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                                       `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                                       `json:"node_type_id,omitempty"`
	NumWorkers                int                                                          `json:"num_workers"`
	PolicyId                  string                                                       `json:"policy_id,omitempty"`
	RuntimeEngine             string                                                       `json:"runtime_engine,omitempty"`
	SingleUserName            string                                                       `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                            `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                            `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                                       `json:"spark_version"`
	SshPublicKeys             []string                                                     `json:"ssh_public_keys,omitempty"`
	Autoscale                 *DataSourceJobJobSettingsSettingsNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *DataSourceJobJobSettingsSettingsNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *DataSourceJobJobSettingsSettingsNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *DataSourceJobJobSettingsSettingsNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []DataSourceJobJobSettingsSettingsNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *DataSourceJobJobSettingsSettingsNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *DataSourceJobJobSettingsSettingsNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []DataSourceJobJobSettingsSettingsNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *DataSourceJobJobSettingsSettingsNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceJobJobSettingsSettingsNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsNotificationSettings struct {
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type DataSourceJobJobSettingsSettingsPipelineTask struct {
	PipelineId string `json:"pipeline_id"`
}

type DataSourceJobJobSettingsSettingsPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsQueue struct {
}

type DataSourceJobJobSettingsSettingsRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type DataSourceJobJobSettingsSettingsSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDependsOn struct {
	TaskKey string `json:"task_key,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskEmailNotifications struct {
	AlertOnLastAttempt    bool     `json:"alert_on_last_attempt,omitempty"`
	NoAlertForSkippedRuns bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnFailure             []string `json:"on_failure,omitempty"`
	OnStart               []string `json:"on_start,omitempty"`
	OnSuccess             []string `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskLibrary struct {
	Egg   string                                            `json:"egg,omitempty"`
	Jar   string                                            `json:"jar,omitempty"`
	Whl   string                                            `json:"whl,omitempty"`
	Cran  *DataSourceJobJobSettingsSettingsTaskLibraryCran  `json:"cran,omitempty"`
	Maven *DataSourceJobJobSettingsSettingsTaskLibraryMaven `json:"maven,omitempty"`
	Pypi  *DataSourceJobJobSettingsSettingsTaskLibraryPypi  `json:"pypi,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConf struct {
	Dbfs *DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                               `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                               `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *DataSourceJobJobSettingsSettingsTaskNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterDockerImage struct {
	Url       string                                                              `json:"url"`
	BasicAuth *DataSourceJobJobSettingsSettingsTaskNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsFile struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsGcs struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Workspace *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterWorkloadType struct {
	Clients *DataSourceJobJobSettingsSettingsTaskNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewCluster struct {
	ApplyPolicyDefaultValues  bool                                                             `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                                              `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                                           `json:"cluster_id,omitempty"`
	ClusterName               string                                                           `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                                `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                                           `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                                           `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                                           `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                                             `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                                             `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                                           `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                                           `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                                           `json:"node_type_id,omitempty"`
	NumWorkers                int                                                              `json:"num_workers"`
	PolicyId                  string                                                           `json:"policy_id,omitempty"`
	RuntimeEngine             string                                                           `json:"runtime_engine,omitempty"`
	SingleUserName            string                                                           `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                                `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                                `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                                           `json:"spark_version"`
	SshPublicKeys             []string                                                         `json:"ssh_public_keys,omitempty"`
	Autoscale                 *DataSourceJobJobSettingsSettingsTaskNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *DataSourceJobJobSettingsSettingsTaskNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *DataSourceJobJobSettingsSettingsTaskNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *DataSourceJobJobSettingsSettingsTaskNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []DataSourceJobJobSettingsSettingsTaskNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *DataSourceJobJobSettingsSettingsTaskNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *DataSourceJobJobSettingsSettingsTaskNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []DataSourceJobJobSettingsSettingsTaskNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *DataSourceJobJobSettingsSettingsTaskNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskPipelineTask struct {
	PipelineId string `json:"pipeline_id"`
}

type DataSourceJobJobSettingsSettingsTaskPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskAlert struct {
	AlertId string `json:"alert_id"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskDashboard struct {
	DashboardId string `json:"dashboard_id"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskFile struct {
	Path string `json:"path"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskQuery struct {
	QueryId string `json:"query_id"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTask struct {
	Parameters  map[string]string                                     `json:"parameters,omitempty"`
	WarehouseId string                                                `json:"warehouse_id,omitempty"`
	Alert       *DataSourceJobJobSettingsSettingsTaskSqlTaskAlert     `json:"alert,omitempty"`
	Dashboard   *DataSourceJobJobSettingsSettingsTaskSqlTaskDashboard `json:"dashboard,omitempty"`
	File        *DataSourceJobJobSettingsSettingsTaskSqlTaskFile      `json:"file,omitempty"`
	Query       *DataSourceJobJobSettingsSettingsTaskSqlTaskQuery     `json:"query,omitempty"`
}

type DataSourceJobJobSettingsSettingsTask struct {
	Description            string                                                  `json:"description,omitempty"`
	ExistingClusterId      string                                                  `json:"existing_cluster_id,omitempty"`
	JobClusterKey          string                                                  `json:"job_cluster_key,omitempty"`
	MaxRetries             int                                                     `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                                     `json:"min_retry_interval_millis,omitempty"`
	RetryOnTimeout         bool                                                    `json:"retry_on_timeout,omitempty"`
	RunIf                  string                                                  `json:"run_if,omitempty"`
	TaskKey                string                                                  `json:"task_key,omitempty"`
	TimeoutSeconds         int                                                     `json:"timeout_seconds,omitempty"`
	DbtTask                *DataSourceJobJobSettingsSettingsTaskDbtTask            `json:"dbt_task,omitempty"`
	DependsOn              []DataSourceJobJobSettingsSettingsTaskDependsOn         `json:"depends_on,omitempty"`
	EmailNotifications     *DataSourceJobJobSettingsSettingsTaskEmailNotifications `json:"email_notifications,omitempty"`
	Library                []DataSourceJobJobSettingsSettingsTaskLibrary           `json:"library,omitempty"`
	NewCluster             *DataSourceJobJobSettingsSettingsTaskNewCluster         `json:"new_cluster,omitempty"`
	NotebookTask           *DataSourceJobJobSettingsSettingsTaskNotebookTask       `json:"notebook_task,omitempty"`
	PipelineTask           *DataSourceJobJobSettingsSettingsTaskPipelineTask       `json:"pipeline_task,omitempty"`
	PythonWheelTask        *DataSourceJobJobSettingsSettingsTaskPythonWheelTask    `json:"python_wheel_task,omitempty"`
	SparkJarTask           *DataSourceJobJobSettingsSettingsTaskSparkJarTask       `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *DataSourceJobJobSettingsSettingsTaskSparkPythonTask    `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *DataSourceJobJobSettingsSettingsTaskSparkSubmitTask    `json:"spark_submit_task,omitempty"`
	SqlTask                *DataSourceJobJobSettingsSettingsTaskSqlTask            `json:"sql_task,omitempty"`
}

type DataSourceJobJobSettingsSettingsTriggerFileArrival struct {
	MinTimeBetweenTriggerSeconds int    `json:"min_time_between_trigger_seconds,omitempty"`
	Url                          string `json:"url"`
	WaitAfterLastChangeSeconds   int    `json:"wait_after_last_change_seconds,omitempty"`
}

type DataSourceJobJobSettingsSettingsTrigger struct {
	PauseStatus string                                              `json:"pause_status,omitempty"`
	FileArrival *DataSourceJobJobSettingsSettingsTriggerFileArrival `json:"file_arrival,omitempty"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnFailure struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnStart struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnSuccess struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotifications struct {
	OnFailure []DataSourceJobJobSettingsSettingsWebhookNotificationsOnFailure `json:"on_failure,omitempty"`
	OnStart   []DataSourceJobJobSettingsSettingsWebhookNotificationsOnStart   `json:"on_start,omitempty"`
	OnSuccess []DataSourceJobJobSettingsSettingsWebhookNotificationsOnSuccess `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettings struct {
	ExistingClusterId      string                                                `json:"existing_cluster_id,omitempty"`
	Format                 string                                                `json:"format,omitempty"`
	MaxConcurrentRuns      int                                                   `json:"max_concurrent_runs,omitempty"`
	MaxRetries             int                                                   `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                                   `json:"min_retry_interval_millis,omitempty"`
	Name                   string                                                `json:"name,omitempty"`
	RetryOnTimeout         bool                                                  `json:"retry_on_timeout,omitempty"`
	Tags                   map[string]string                                     `json:"tags,omitempty"`
	TimeoutSeconds         int                                                   `json:"timeout_seconds,omitempty"`
	Continuous             *DataSourceJobJobSettingsSettingsContinuous           `json:"continuous,omitempty"`
	DbtTask                *DataSourceJobJobSettingsSettingsDbtTask              `json:"dbt_task,omitempty"`
	EmailNotifications     *DataSourceJobJobSettingsSettingsEmailNotifications   `json:"email_notifications,omitempty"`
	GitSource              *DataSourceJobJobSettingsSettingsGitSource            `json:"git_source,omitempty"`
	JobCluster             []DataSourceJobJobSettingsSettingsJobCluster          `json:"job_cluster,omitempty"`
	Library                []DataSourceJobJobSettingsSettingsLibrary             `json:"library,omitempty"`
	NewCluster             *DataSourceJobJobSettingsSettingsNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *DataSourceJobJobSettingsSettingsNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *DataSourceJobJobSettingsSettingsNotificationSettings `json:"notification_settings,omitempty"`
	PipelineTask           *DataSourceJobJobSettingsSettingsPipelineTask         `json:"pipeline_task,omitempty"`
	PythonWheelTask        *DataSourceJobJobSettingsSettingsPythonWheelTask      `json:"python_wheel_task,omitempty"`
	Queue                  *DataSourceJobJobSettingsSettingsQueue                `json:"queue,omitempty"`
	RunAs                  *DataSourceJobJobSettingsSettingsRunAs                `json:"run_as,omitempty"`
	Schedule               *DataSourceJobJobSettingsSettingsSchedule             `json:"schedule,omitempty"`
	SparkJarTask           *DataSourceJobJobSettingsSettingsSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *DataSourceJobJobSettingsSettingsSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *DataSourceJobJobSettingsSettingsSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	Task                   []DataSourceJobJobSettingsSettingsTask                `json:"task,omitempty"`
	Trigger                *DataSourceJobJobSettingsSettingsTrigger              `json:"trigger,omitempty"`
	WebhookNotifications   *DataSourceJobJobSettingsSettingsWebhookNotifications `json:"webhook_notifications,omitempty"`
}

type DataSourceJobJobSettings struct {
	CreatedTime     int                               `json:"created_time,omitempty"`
	CreatorUserName string                            `json:"creator_user_name,omitempty"`
	JobId           int                               `json:"job_id,omitempty"`
	RunAsUserName   string                            `json:"run_as_user_name,omitempty"`
	Settings        *DataSourceJobJobSettingsSettings `json:"settings,omitempty"`
}

type DataSourceJob struct {
	Id          string                    `json:"id,omitempty"`
	JobId       string                    `json:"job_id,omitempty"`
	JobName     string                    `json:"job_name,omitempty"`
	Name        string                    `json:"name,omitempty"`
	JobSettings *DataSourceJobJobSettings `json:"job_settings,omitempty"`
}
