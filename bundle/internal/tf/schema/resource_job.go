// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceJobComputeSpec struct {
	Kind string `json:"kind,omitempty"`
}

type ResourceJobCompute struct {
	ComputeKey string                  `json:"compute_key,omitempty"`
	Spec       *ResourceJobComputeSpec `json:"spec,omitempty"`
}

type ResourceJobContinuous struct {
	PauseStatus string `json:"pause_status,omitempty"`
}

type ResourceJobDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type ResourceJobDeployment struct {
	Kind             string `json:"kind"`
	MetadataFilePath string `json:"metadata_file_path,omitempty"`
}

type ResourceJobEmailNotifications struct {
	NoAlertForSkippedRuns              bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type ResourceJobGitSourceJobSource struct {
	DirtyState          string `json:"dirty_state,omitempty"`
	ImportFromGitBranch string `json:"import_from_git_branch"`
	JobConfigPath       string `json:"job_config_path"`
}

type ResourceJobGitSource struct {
	Branch    string                         `json:"branch,omitempty"`
	Commit    string                         `json:"commit,omitempty"`
	Provider  string                         `json:"provider,omitempty"`
	Tag       string                         `json:"tag,omitempty"`
	Url       string                         `json:"url"`
	JobSource *ResourceJobGitSourceJobSource `json:"job_source,omitempty"`
}

type ResourceJobHealthRules struct {
	Metric string `json:"metric,omitempty"`
	Op     string `json:"op,omitempty"`
	Value  int    `json:"value,omitempty"`
}

type ResourceJobHealth struct {
	Rules []ResourceJobHealthRules `json:"rules,omitempty"`
}

type ResourceJobJobClusterNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type ResourceJobJobClusterNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourceJobJobClusterNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type ResourceJobJobClusterNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobJobClusterNewClusterClusterLogConf struct {
	Dbfs *ResourceJobJobClusterNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *ResourceJobJobClusterNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type ResourceJobJobClusterNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type ResourceJobJobClusterNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *ResourceJobJobClusterNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type ResourceJobJobClusterNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceJobJobClusterNewClusterDockerImage struct {
	Url       string                                               `json:"url"`
	BasicAuth *ResourceJobJobClusterNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceJobJobClusterNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type ResourceJobJobClusterNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobJobClusterNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourceJobJobClusterNewClusterInitScripts struct {
	Abfss     *ResourceJobJobClusterNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourceJobJobClusterNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourceJobJobClusterNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourceJobJobClusterNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourceJobJobClusterNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourceJobJobClusterNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourceJobJobClusterNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourceJobJobClusterNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type ResourceJobJobClusterNewClusterWorkloadType struct {
	Clients *ResourceJobJobClusterNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type ResourceJobJobClusterNewCluster struct {
	ApplyPolicyDefaultValues  bool                                              `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                               `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                            `json:"cluster_id,omitempty"`
	ClusterName               string                                            `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                 `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                            `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                            `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                            `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                              `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                              `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                            `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                            `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                            `json:"node_type_id,omitempty"`
	NumWorkers                int                                               `json:"num_workers,omitempty"`
	PolicyId                  string                                            `json:"policy_id,omitempty"`
	RuntimeEngine             string                                            `json:"runtime_engine,omitempty"`
	SingleUserName            string                                            `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                 `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                 `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                            `json:"spark_version"`
	SshPublicKeys             []string                                          `json:"ssh_public_keys,omitempty"`
	Autoscale                 *ResourceJobJobClusterNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *ResourceJobJobClusterNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *ResourceJobJobClusterNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *ResourceJobJobClusterNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []ResourceJobJobClusterNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *ResourceJobJobClusterNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *ResourceJobJobClusterNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []ResourceJobJobClusterNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *ResourceJobJobClusterNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type ResourceJobJobCluster struct {
	JobClusterKey string                           `json:"job_cluster_key,omitempty"`
	NewCluster    *ResourceJobJobClusterNewCluster `json:"new_cluster,omitempty"`
}

type ResourceJobLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceJobLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobLibrary struct {
	Egg   string                   `json:"egg,omitempty"`
	Jar   string                   `json:"jar,omitempty"`
	Whl   string                   `json:"whl,omitempty"`
	Cran  *ResourceJobLibraryCran  `json:"cran,omitempty"`
	Maven *ResourceJobLibraryMaven `json:"maven,omitempty"`
	Pypi  *ResourceJobLibraryPypi  `json:"pypi,omitempty"`
}

type ResourceJobNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type ResourceJobNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourceJobNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type ResourceJobNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobNewClusterClusterLogConf struct {
	Dbfs *ResourceJobNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *ResourceJobNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type ResourceJobNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type ResourceJobNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                      `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                      `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *ResourceJobNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type ResourceJobNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceJobNewClusterDockerImage struct {
	Url       string                                     `json:"url"`
	BasicAuth *ResourceJobNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceJobNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type ResourceJobNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourceJobNewClusterInitScripts struct {
	Abfss     *ResourceJobNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourceJobNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourceJobNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourceJobNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourceJobNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourceJobNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourceJobNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourceJobNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type ResourceJobNewClusterWorkloadType struct {
	Clients *ResourceJobNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type ResourceJobNewCluster struct {
	ApplyPolicyDefaultValues  bool                                    `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                     `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                  `json:"cluster_id,omitempty"`
	ClusterName               string                                  `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                       `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                  `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                  `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                  `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                    `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                    `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                  `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                  `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                  `json:"node_type_id,omitempty"`
	NumWorkers                int                                     `json:"num_workers,omitempty"`
	PolicyId                  string                                  `json:"policy_id,omitempty"`
	RuntimeEngine             string                                  `json:"runtime_engine,omitempty"`
	SingleUserName            string                                  `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                       `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                       `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                  `json:"spark_version"`
	SshPublicKeys             []string                                `json:"ssh_public_keys,omitempty"`
	Autoscale                 *ResourceJobNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *ResourceJobNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *ResourceJobNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *ResourceJobNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []ResourceJobNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *ResourceJobNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *ResourceJobNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []ResourceJobNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *ResourceJobNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type ResourceJobNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
}

type ResourceJobNotificationSettings struct {
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type ResourceJobParameter struct {
	Default string `json:"default"`
	Name    string `json:"name"`
}

type ResourceJobPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type ResourceJobPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type ResourceJobQueue struct {
	Enabled bool `json:"enabled"`
}

type ResourceJobRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type ResourceJobRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
}

type ResourceJobSchedule struct {
	PauseStatus          string `json:"pause_status,omitempty"`
	QuartzCronExpression string `json:"quartz_cron_expression"`
	TimezoneId           string `json:"timezone_id"`
}

type ResourceJobSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type ResourceJobSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type ResourceJobSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type ResourceJobTaskConditionTask struct {
	Left  string `json:"left,omitempty"`
	Op    string `json:"op,omitempty"`
	Right string `json:"right,omitempty"`
}

type ResourceJobTaskDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type ResourceJobTaskDependsOn struct {
	Outcome string `json:"outcome,omitempty"`
	TaskKey string `json:"task_key"`
}

type ResourceJobTaskEmailNotifications struct {
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type ResourceJobTaskForEachTaskTaskConditionTask struct {
	Left  string `json:"left,omitempty"`
	Op    string `json:"op,omitempty"`
	Right string `json:"right,omitempty"`
}

type ResourceJobTaskForEachTaskTaskDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskDependsOn struct {
	Outcome string `json:"outcome,omitempty"`
	TaskKey string `json:"task_key"`
}

type ResourceJobTaskForEachTaskTaskEmailNotifications struct {
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type ResourceJobTaskForEachTaskTaskHealthRules struct {
	Metric string `json:"metric,omitempty"`
	Op     string `json:"op,omitempty"`
	Value  int    `json:"value,omitempty"`
}

type ResourceJobTaskForEachTaskTaskHealth struct {
	Rules []ResourceJobTaskForEachTaskTaskHealthRules `json:"rules,omitempty"`
}

type ResourceJobTaskForEachTaskTaskLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobTaskForEachTaskTaskLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceJobTaskForEachTaskTaskLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobTaskForEachTaskTaskLibrary struct {
	Egg   string                                      `json:"egg,omitempty"`
	Jar   string                                      `json:"jar,omitempty"`
	Whl   string                                      `json:"whl,omitempty"`
	Cran  *ResourceJobTaskForEachTaskTaskLibraryCran  `json:"cran,omitempty"`
	Maven *ResourceJobTaskForEachTaskTaskLibraryMaven `json:"maven,omitempty"`
	Pypi  *ResourceJobTaskForEachTaskTaskLibraryPypi  `json:"pypi,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterClusterLogConf struct {
	Dbfs *ResourceJobTaskForEachTaskTaskNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *ResourceJobTaskForEachTaskTaskNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type ResourceJobTaskForEachTaskTaskNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                         `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                         `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *ResourceJobTaskForEachTaskTaskNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceJobTaskForEachTaskTaskNewClusterDockerImage struct {
	Url       string                                                        `json:"url"`
	BasicAuth *ResourceJobTaskForEachTaskTaskNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskForEachTaskTaskNewClusterInitScripts struct {
	Abfss     *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourceJobTaskForEachTaskTaskNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewClusterWorkloadType struct {
	Clients *ResourceJobTaskForEachTaskTaskNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNewCluster struct {
	ApplyPolicyDefaultValues  bool                                                       `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                                        `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                                     `json:"cluster_id,omitempty"`
	ClusterName               string                                                     `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                          `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                                     `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                                     `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                                     `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                                       `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                                       `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                                     `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                                     `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                                     `json:"node_type_id,omitempty"`
	NumWorkers                int                                                        `json:"num_workers"`
	PolicyId                  string                                                     `json:"policy_id,omitempty"`
	RuntimeEngine             string                                                     `json:"runtime_engine,omitempty"`
	SingleUserName            string                                                     `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                          `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                          `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                                     `json:"spark_version"`
	SshPublicKeys             []string                                                   `json:"ssh_public_keys,omitempty"`
	Autoscale                 *ResourceJobTaskForEachTaskTaskNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *ResourceJobTaskForEachTaskTaskNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *ResourceJobTaskForEachTaskTaskNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *ResourceJobTaskForEachTaskTaskNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []ResourceJobTaskForEachTaskTaskNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *ResourceJobTaskForEachTaskTaskNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *ResourceJobTaskForEachTaskTaskNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []ResourceJobTaskForEachTaskTaskNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *ResourceJobTaskForEachTaskTaskNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
}

type ResourceJobTaskForEachTaskTaskNotificationSettings struct {
	AlertOnLastAttempt     bool `json:"alert_on_last_attempt,omitempty"`
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type ResourceJobTaskForEachTaskTaskPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type ResourceJobTaskForEachTaskTaskPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type ResourceJobTaskForEachTaskTaskRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskAlertSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskAlert struct {
	AlertId            string                                                    `json:"alert_id"`
	PauseSubscriptions bool                                                      `json:"pause_subscriptions,omitempty"`
	Subscriptions      []ResourceJobTaskForEachTaskTaskSqlTaskAlertSubscriptions `json:"subscriptions,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskDashboardSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskDashboard struct {
	CustomSubject      string                                                        `json:"custom_subject,omitempty"`
	DashboardId        string                                                        `json:"dashboard_id"`
	PauseSubscriptions bool                                                          `json:"pause_subscriptions,omitempty"`
	Subscriptions      []ResourceJobTaskForEachTaskTaskSqlTaskDashboardSubscriptions `json:"subscriptions,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskFile struct {
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

type ResourceJobTaskForEachTaskTaskSqlTaskQuery struct {
	QueryId string `json:"query_id"`
}

type ResourceJobTaskForEachTaskTaskSqlTask struct {
	Parameters  map[string]string                               `json:"parameters,omitempty"`
	WarehouseId string                                          `json:"warehouse_id,omitempty"`
	Alert       *ResourceJobTaskForEachTaskTaskSqlTaskAlert     `json:"alert,omitempty"`
	Dashboard   *ResourceJobTaskForEachTaskTaskSqlTaskDashboard `json:"dashboard,omitempty"`
	File        *ResourceJobTaskForEachTaskTaskSqlTaskFile      `json:"file,omitempty"`
	Query       *ResourceJobTaskForEachTaskTaskSqlTaskQuery     `json:"query,omitempty"`
}

type ResourceJobTaskForEachTaskTaskWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskWebhookNotificationsOnFailure struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskWebhookNotificationsOnStart struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskWebhookNotificationsOnSuccess struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskForEachTaskTaskWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []ResourceJobTaskForEachTaskTaskWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []ResourceJobTaskForEachTaskTaskWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []ResourceJobTaskForEachTaskTaskWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnSuccess                          []ResourceJobTaskForEachTaskTaskWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type ResourceJobTaskForEachTaskTask struct {
	ComputeKey             string                                              `json:"compute_key,omitempty"`
	Description            string                                              `json:"description,omitempty"`
	ExistingClusterId      string                                              `json:"existing_cluster_id,omitempty"`
	JobClusterKey          string                                              `json:"job_cluster_key,omitempty"`
	MaxRetries             int                                                 `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                                 `json:"min_retry_interval_millis,omitempty"`
	RetryOnTimeout         bool                                                `json:"retry_on_timeout,omitempty"`
	RunIf                  string                                              `json:"run_if,omitempty"`
	TaskKey                string                                              `json:"task_key,omitempty"`
	TimeoutSeconds         int                                                 `json:"timeout_seconds,omitempty"`
	ConditionTask          *ResourceJobTaskForEachTaskTaskConditionTask        `json:"condition_task,omitempty"`
	DbtTask                *ResourceJobTaskForEachTaskTaskDbtTask              `json:"dbt_task,omitempty"`
	DependsOn              []ResourceJobTaskForEachTaskTaskDependsOn           `json:"depends_on,omitempty"`
	EmailNotifications     *ResourceJobTaskForEachTaskTaskEmailNotifications   `json:"email_notifications,omitempty"`
	Health                 *ResourceJobTaskForEachTaskTaskHealth               `json:"health,omitempty"`
	Library                []ResourceJobTaskForEachTaskTaskLibrary             `json:"library,omitempty"`
	NewCluster             *ResourceJobTaskForEachTaskTaskNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *ResourceJobTaskForEachTaskTaskNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *ResourceJobTaskForEachTaskTaskNotificationSettings `json:"notification_settings,omitempty"`
	PipelineTask           *ResourceJobTaskForEachTaskTaskPipelineTask         `json:"pipeline_task,omitempty"`
	PythonWheelTask        *ResourceJobTaskForEachTaskTaskPythonWheelTask      `json:"python_wheel_task,omitempty"`
	RunJobTask             *ResourceJobTaskForEachTaskTaskRunJobTask           `json:"run_job_task,omitempty"`
	SparkJarTask           *ResourceJobTaskForEachTaskTaskSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *ResourceJobTaskForEachTaskTaskSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *ResourceJobTaskForEachTaskTaskSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	SqlTask                *ResourceJobTaskForEachTaskTaskSqlTask              `json:"sql_task,omitempty"`
	WebhookNotifications   *ResourceJobTaskForEachTaskTaskWebhookNotifications `json:"webhook_notifications,omitempty"`
}

type ResourceJobTaskForEachTask struct {
	Concurrency int                             `json:"concurrency,omitempty"`
	Inputs      string                          `json:"inputs"`
	Task        *ResourceJobTaskForEachTaskTask `json:"task,omitempty"`
}

type ResourceJobTaskHealthRules struct {
	Metric string `json:"metric,omitempty"`
	Op     string `json:"op,omitempty"`
	Value  int    `json:"value,omitempty"`
}

type ResourceJobTaskHealth struct {
	Rules []ResourceJobTaskHealthRules `json:"rules,omitempty"`
}

type ResourceJobTaskLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobTaskLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceJobTaskLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceJobTaskLibrary struct {
	Egg   string                       `json:"egg,omitempty"`
	Jar   string                       `json:"jar,omitempty"`
	Whl   string                       `json:"whl,omitempty"`
	Cran  *ResourceJobTaskLibraryCran  `json:"cran,omitempty"`
	Maven *ResourceJobTaskLibraryMaven `json:"maven,omitempty"`
	Pypi  *ResourceJobTaskLibraryPypi  `json:"pypi,omitempty"`
}

type ResourceJobTaskNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type ResourceJobTaskNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourceJobTaskNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type ResourceJobTaskNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobTaskNewClusterClusterLogConf struct {
	Dbfs *ResourceJobTaskNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *ResourceJobTaskNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type ResourceJobTaskNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type ResourceJobTaskNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                          `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                          `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *ResourceJobTaskNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type ResourceJobTaskNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceJobTaskNewClusterDockerImage struct {
	Url       string                                         `json:"url"`
	BasicAuth *ResourceJobTaskNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceJobTaskNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type ResourceJobTaskNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceJobTaskNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourceJobTaskNewClusterInitScripts struct {
	Abfss     *ResourceJobTaskNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourceJobTaskNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourceJobTaskNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourceJobTaskNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourceJobTaskNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourceJobTaskNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourceJobTaskNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourceJobTaskNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type ResourceJobTaskNewClusterWorkloadType struct {
	Clients *ResourceJobTaskNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type ResourceJobTaskNewCluster struct {
	ApplyPolicyDefaultValues  bool                                        `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                         `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                      `json:"cluster_id,omitempty"`
	ClusterName               string                                      `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                           `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                      `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                      `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                      `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                        `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                        `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                      `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                      `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                      `json:"node_type_id,omitempty"`
	NumWorkers                int                                         `json:"num_workers,omitempty"`
	PolicyId                  string                                      `json:"policy_id,omitempty"`
	RuntimeEngine             string                                      `json:"runtime_engine,omitempty"`
	SingleUserName            string                                      `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                           `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                           `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                      `json:"spark_version"`
	SshPublicKeys             []string                                    `json:"ssh_public_keys,omitempty"`
	Autoscale                 *ResourceJobTaskNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *ResourceJobTaskNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *ResourceJobTaskNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *ResourceJobTaskNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []ResourceJobTaskNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *ResourceJobTaskNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *ResourceJobTaskNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []ResourceJobTaskNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *ResourceJobTaskNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type ResourceJobTaskNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
}

type ResourceJobTaskNotificationSettings struct {
	AlertOnLastAttempt     bool `json:"alert_on_last_attempt,omitempty"`
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type ResourceJobTaskPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type ResourceJobTaskPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type ResourceJobTaskRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
}

type ResourceJobTaskSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type ResourceJobTaskSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type ResourceJobTaskSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type ResourceJobTaskSqlTaskAlertSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type ResourceJobTaskSqlTaskAlert struct {
	AlertId            string                                     `json:"alert_id"`
	PauseSubscriptions bool                                       `json:"pause_subscriptions,omitempty"`
	Subscriptions      []ResourceJobTaskSqlTaskAlertSubscriptions `json:"subscriptions,omitempty"`
}

type ResourceJobTaskSqlTaskDashboardSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type ResourceJobTaskSqlTaskDashboard struct {
	CustomSubject      string                                         `json:"custom_subject,omitempty"`
	DashboardId        string                                         `json:"dashboard_id"`
	PauseSubscriptions bool                                           `json:"pause_subscriptions,omitempty"`
	Subscriptions      []ResourceJobTaskSqlTaskDashboardSubscriptions `json:"subscriptions,omitempty"`
}

type ResourceJobTaskSqlTaskFile struct {
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

type ResourceJobTaskSqlTaskQuery struct {
	QueryId string `json:"query_id"`
}

type ResourceJobTaskSqlTask struct {
	Parameters  map[string]string                `json:"parameters,omitempty"`
	WarehouseId string                           `json:"warehouse_id,omitempty"`
	Alert       *ResourceJobTaskSqlTaskAlert     `json:"alert,omitempty"`
	Dashboard   *ResourceJobTaskSqlTaskDashboard `json:"dashboard,omitempty"`
	File        *ResourceJobTaskSqlTaskFile      `json:"file,omitempty"`
	Query       *ResourceJobTaskSqlTaskQuery     `json:"query,omitempty"`
}

type ResourceJobTaskWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskWebhookNotificationsOnFailure struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskWebhookNotificationsOnStart struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskWebhookNotificationsOnSuccess struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobTaskWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []ResourceJobTaskWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []ResourceJobTaskWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []ResourceJobTaskWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnSuccess                          []ResourceJobTaskWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type ResourceJobTask struct {
	ComputeKey             string                               `json:"compute_key,omitempty"`
	Description            string                               `json:"description,omitempty"`
	ExistingClusterId      string                               `json:"existing_cluster_id,omitempty"`
	JobClusterKey          string                               `json:"job_cluster_key,omitempty"`
	MaxRetries             int                                  `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                  `json:"min_retry_interval_millis,omitempty"`
	RetryOnTimeout         bool                                 `json:"retry_on_timeout,omitempty"`
	RunIf                  string                               `json:"run_if,omitempty"`
	TaskKey                string                               `json:"task_key,omitempty"`
	TimeoutSeconds         int                                  `json:"timeout_seconds,omitempty"`
	ConditionTask          *ResourceJobTaskConditionTask        `json:"condition_task,omitempty"`
	DbtTask                *ResourceJobTaskDbtTask              `json:"dbt_task,omitempty"`
	DependsOn              []ResourceJobTaskDependsOn           `json:"depends_on,omitempty"`
	EmailNotifications     *ResourceJobTaskEmailNotifications   `json:"email_notifications,omitempty"`
	ForEachTask            *ResourceJobTaskForEachTask          `json:"for_each_task,omitempty"`
	Health                 *ResourceJobTaskHealth               `json:"health,omitempty"`
	Library                []ResourceJobTaskLibrary             `json:"library,omitempty"`
	NewCluster             *ResourceJobTaskNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *ResourceJobTaskNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *ResourceJobTaskNotificationSettings `json:"notification_settings,omitempty"`
	PipelineTask           *ResourceJobTaskPipelineTask         `json:"pipeline_task,omitempty"`
	PythonWheelTask        *ResourceJobTaskPythonWheelTask      `json:"python_wheel_task,omitempty"`
	RunJobTask             *ResourceJobTaskRunJobTask           `json:"run_job_task,omitempty"`
	SparkJarTask           *ResourceJobTaskSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *ResourceJobTaskSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *ResourceJobTaskSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	SqlTask                *ResourceJobTaskSqlTask              `json:"sql_task,omitempty"`
	WebhookNotifications   *ResourceJobTaskWebhookNotifications `json:"webhook_notifications,omitempty"`
}

type ResourceJobTriggerFileArrival struct {
	MinTimeBetweenTriggersSeconds int    `json:"min_time_between_triggers_seconds,omitempty"`
	Url                           string `json:"url"`
	WaitAfterLastChangeSeconds    int    `json:"wait_after_last_change_seconds,omitempty"`
}

type ResourceJobTrigger struct {
	PauseStatus string                         `json:"pause_status,omitempty"`
	FileArrival *ResourceJobTriggerFileArrival `json:"file_arrival,omitempty"`
}

type ResourceJobWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobWebhookNotificationsOnFailure struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobWebhookNotificationsOnStart struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobWebhookNotificationsOnSuccess struct {
	Id string `json:"id,omitempty"`
}

type ResourceJobWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []ResourceJobWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []ResourceJobWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []ResourceJobWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnSuccess                          []ResourceJobWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type ResourceJob struct {
	AlwaysRunning          bool                             `json:"always_running,omitempty"`
	ControlRunState        bool                             `json:"control_run_state,omitempty"`
	Description            string                           `json:"description,omitempty"`
	EditMode               string                           `json:"edit_mode,omitempty"`
	ExistingClusterId      string                           `json:"existing_cluster_id,omitempty"`
	Format                 string                           `json:"format,omitempty"`
	Id                     string                           `json:"id,omitempty"`
	MaxConcurrentRuns      int                              `json:"max_concurrent_runs,omitempty"`
	MaxRetries             int                              `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                              `json:"min_retry_interval_millis,omitempty"`
	Name                   string                           `json:"name,omitempty"`
	RetryOnTimeout         bool                             `json:"retry_on_timeout,omitempty"`
	Tags                   map[string]string                `json:"tags,omitempty"`
	TimeoutSeconds         int                              `json:"timeout_seconds,omitempty"`
	Url                    string                           `json:"url,omitempty"`
	Compute                []ResourceJobCompute             `json:"compute,omitempty"`
	Continuous             *ResourceJobContinuous           `json:"continuous,omitempty"`
	DbtTask                *ResourceJobDbtTask              `json:"dbt_task,omitempty"`
	Deployment             *ResourceJobDeployment           `json:"deployment,omitempty"`
	EmailNotifications     *ResourceJobEmailNotifications   `json:"email_notifications,omitempty"`
	GitSource              *ResourceJobGitSource            `json:"git_source,omitempty"`
	Health                 *ResourceJobHealth               `json:"health,omitempty"`
	JobCluster             []ResourceJobJobCluster          `json:"job_cluster,omitempty"`
	Library                []ResourceJobLibrary             `json:"library,omitempty"`
	NewCluster             *ResourceJobNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *ResourceJobNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *ResourceJobNotificationSettings `json:"notification_settings,omitempty"`
	Parameter              []ResourceJobParameter           `json:"parameter,omitempty"`
	PipelineTask           *ResourceJobPipelineTask         `json:"pipeline_task,omitempty"`
	PythonWheelTask        *ResourceJobPythonWheelTask      `json:"python_wheel_task,omitempty"`
	Queue                  *ResourceJobQueue                `json:"queue,omitempty"`
	RunAs                  *ResourceJobRunAs                `json:"run_as,omitempty"`
	RunJobTask             *ResourceJobRunJobTask           `json:"run_job_task,omitempty"`
	Schedule               *ResourceJobSchedule             `json:"schedule,omitempty"`
	SparkJarTask           *ResourceJobSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *ResourceJobSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *ResourceJobSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	Task                   []ResourceJobTask                `json:"task,omitempty"`
	Trigger                *ResourceJobTrigger              `json:"trigger,omitempty"`
	WebhookNotifications   *ResourceJobWebhookNotifications `json:"webhook_notifications,omitempty"`
}
