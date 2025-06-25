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
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsDeployment struct {
	Kind             string `json:"kind"`
	MetadataFilePath string `json:"metadata_file_path,omitempty"`
}

type DataSourceJobJobSettingsSettingsEmailNotifications struct {
	NoAlertForSkippedRuns              bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []string `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsEnvironmentSpec struct {
	Client             string   `json:"client,omitempty"`
	Dependencies       []string `json:"dependencies,omitempty"`
	EnvironmentVersion string   `json:"environment_version,omitempty"`
	JarDependencies    []string `json:"jar_dependencies,omitempty"`
}

type DataSourceJobJobSettingsSettingsEnvironment struct {
	EnvironmentKey string                                           `json:"environment_key"`
	Spec           *DataSourceJobJobSettingsSettingsEnvironmentSpec `json:"spec,omitempty"`
}

type DataSourceJobJobSettingsSettingsGitSourceJobSource struct {
	DirtyState          string `json:"dirty_state,omitempty"`
	ImportFromGitBranch string `json:"import_from_git_branch"`
	JobConfigPath       string `json:"job_config_path"`
}

type DataSourceJobJobSettingsSettingsGitSource struct {
	Branch    string                                              `json:"branch,omitempty"`
	Commit    string                                              `json:"commit,omitempty"`
	Provider  string                                              `json:"provider,omitempty"`
	Tag       string                                              `json:"tag,omitempty"`
	Url       string                                              `json:"url"`
	JobSource *DataSourceJobJobSettingsSettingsGitSourceJobSource `json:"job_source,omitempty"`
}

type DataSourceJobJobSettingsSettingsHealthRules struct {
	Metric string `json:"metric"`
	Op     string `json:"op"`
	Value  int    `json:"value"`
}

type DataSourceJobJobSettingsSettingsHealth struct {
	Rules []DataSourceJobJobSettingsSettingsHealthRules `json:"rules,omitempty"`
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
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
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

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceJobJobSettingsSettingsJobClusterNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
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
	SparkVersion              string                                                                 `json:"spark_version,omitempty"`
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
	JobClusterKey string                                                `json:"job_cluster_key"`
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
	Egg          string                                        `json:"egg,omitempty"`
	Jar          string                                        `json:"jar,omitempty"`
	Requirements string                                        `json:"requirements,omitempty"`
	Whl          string                                        `json:"whl,omitempty"`
	Cran         *DataSourceJobJobSettingsSettingsLibraryCran  `json:"cran,omitempty"`
	Maven        *DataSourceJobJobSettingsSettingsLibraryMaven `json:"maven,omitempty"`
	Pypi         *DataSourceJobJobSettingsSettingsLibraryPypi  `json:"pypi,omitempty"`
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
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
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

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceJobJobSettingsSettingsNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
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
	SparkVersion              string                                                       `json:"spark_version,omitempty"`
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
	WarehouseId    string            `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsNotificationSettings struct {
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type DataSourceJobJobSettingsSettingsParameter struct {
	Default string `json:"default"`
	Name    string `json:"name"`
}

type DataSourceJobJobSettingsSettingsPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type DataSourceJobJobSettingsSettingsPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsQueue struct {
	Enabled bool `json:"enabled"`
}

type DataSourceJobJobSettingsSettingsRunAs struct {
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	UserName             string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
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

type DataSourceJobJobSettingsSettingsTaskConditionTask struct {
	Left  string `json:"left"`
	Op    string `json:"op"`
	Right string `json:"right"`
}

type DataSourceJobJobSettingsSettingsTaskDashboardTaskSubscriptionSubscribers struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDashboardTaskSubscription struct {
	CustomSubject string                                                                     `json:"custom_subject,omitempty"`
	Paused        bool                                                                       `json:"paused,omitempty"`
	Subscribers   []DataSourceJobJobSettingsSettingsTaskDashboardTaskSubscriptionSubscribers `json:"subscribers,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDashboardTask struct {
	DashboardId  string                                                         `json:"dashboard_id,omitempty"`
	WarehouseId  string                                                         `json:"warehouse_id,omitempty"`
	Subscription *DataSourceJobJobSettingsSettingsTaskDashboardTaskSubscription `json:"subscription,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskDependsOn struct {
	Outcome string `json:"outcome,omitempty"`
	TaskKey string `json:"task_key"`
}

type DataSourceJobJobSettingsSettingsTaskEmailNotifications struct {
	NoAlertForSkippedRuns              bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []string `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskConditionTask struct {
	Left  string `json:"left"`
	Op    string `json:"op"`
	Right string `json:"right"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTaskSubscriptionSubscribers struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTaskSubscription struct {
	CustomSubject string                                                                                    `json:"custom_subject,omitempty"`
	Paused        bool                                                                                      `json:"paused,omitempty"`
	Subscribers   []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTaskSubscriptionSubscribers `json:"subscribers,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTask struct {
	DashboardId  string                                                                        `json:"dashboard_id,omitempty"`
	WarehouseId  string                                                                        `json:"warehouse_id,omitempty"`
	Subscription *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTaskSubscription `json:"subscription,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDbtTask struct {
	Catalog           string   `json:"catalog,omitempty"`
	Commands          []string `json:"commands"`
	ProfilesDirectory string   `json:"profiles_directory,omitempty"`
	ProjectDirectory  string   `json:"project_directory,omitempty"`
	Schema            string   `json:"schema,omitempty"`
	Source            string   `json:"source,omitempty"`
	WarehouseId       string   `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDependsOn struct {
	Outcome string `json:"outcome,omitempty"`
	TaskKey string `json:"task_key"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskEmailNotifications struct {
	NoAlertForSkippedRuns              bool     `json:"no_alert_for_skipped_runs,omitempty"`
	OnDurationWarningThresholdExceeded []string `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []string `json:"on_failure,omitempty"`
	OnStart                            []string `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []string `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []string `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskHealthRules struct {
	Metric string `json:"metric"`
	Op     string `json:"op"`
	Value  int    `json:"value"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskHealth struct {
	Rules []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskHealthRules `json:"rules,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibrary struct {
	Egg          string                                                           `json:"egg,omitempty"`
	Jar          string                                                           `json:"jar,omitempty"`
	Requirements string                                                           `json:"requirements,omitempty"`
	Whl          string                                                           `json:"whl,omitempty"`
	Cran         *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryCran  `json:"cran,omitempty"`
	Maven        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryMaven `json:"maven,omitempty"`
	Pypi         *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibraryPypi  `json:"pypi,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConf struct {
	Dbfs *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                                                              `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                                              `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterDockerImage struct {
	Url       string                                                                             `json:"url"`
	BasicAuth *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterWorkloadType struct {
	Clients *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewCluster struct {
	ApplyPolicyDefaultValues  bool                                                                            `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes    int                                                                             `json:"autotermination_minutes,omitempty"`
	ClusterId                 string                                                                          `json:"cluster_id,omitempty"`
	ClusterName               string                                                                          `json:"cluster_name,omitempty"`
	CustomTags                map[string]string                                                               `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                                                          `json:"data_security_mode,omitempty"`
	DriverInstancePoolId      string                                                                          `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                                                          `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                                                            `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                                                            `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken          string                                                                          `json:"idempotency_token,omitempty"`
	InstancePoolId            string                                                                          `json:"instance_pool_id,omitempty"`
	NodeTypeId                string                                                                          `json:"node_type_id,omitempty"`
	NumWorkers                int                                                                             `json:"num_workers"`
	PolicyId                  string                                                                          `json:"policy_id,omitempty"`
	RuntimeEngine             string                                                                          `json:"runtime_engine,omitempty"`
	SingleUserName            string                                                                          `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                                                               `json:"spark_conf,omitempty"`
	SparkEnvVars              map[string]string                                                               `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                                                          `json:"spark_version,omitempty"`
	SshPublicKeys             []string                                                                        `json:"ssh_public_keys,omitempty"`
	Autoscale                 *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo          []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage               *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes             *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterInitScripts      `json:"init_scripts,omitempty"`
	WorkloadType              *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewClusterWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNotebookTask struct {
	BaseParameters map[string]string `json:"base_parameters,omitempty"`
	NotebookPath   string            `json:"notebook_path"`
	Source         string            `json:"source,omitempty"`
	WarehouseId    string            `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNotificationSettings struct {
	AlertOnLastAttempt     bool `json:"alert_on_last_attempt,omitempty"`
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTaskPowerBiModel struct {
	AuthenticationMethod string `json:"authentication_method,omitempty"`
	ModelName            string `json:"model_name,omitempty"`
	OverwriteExisting    bool   `json:"overwrite_existing,omitempty"`
	StorageMode          string `json:"storage_mode,omitempty"`
	WorkspaceName        string `json:"workspace_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTaskTables struct {
	Catalog     string `json:"catalog,omitempty"`
	Name        string `json:"name,omitempty"`
	Schema      string `json:"schema,omitempty"`
	StorageMode string `json:"storage_mode,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTask struct {
	ConnectionResourceName string                                                                      `json:"connection_resource_name,omitempty"`
	RefreshAfterUpdate     bool                                                                        `json:"refresh_after_update,omitempty"`
	WarehouseId            string                                                                      `json:"warehouse_id,omitempty"`
	PowerBiModel           *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTaskPowerBiModel `json:"power_bi_model,omitempty"`
	Tables                 []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTaskTables      `json:"tables,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkJarTask struct {
	JarUri        string   `json:"jar_uri,omitempty"`
	MainClassName string   `json:"main_class_name,omitempty"`
	Parameters    []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkPythonTask struct {
	Parameters []string `json:"parameters,omitempty"`
	PythonFile string   `json:"python_file"`
	Source     string   `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkSubmitTask struct {
	Parameters []string `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskAlertSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskAlert struct {
	AlertId            string                                                                         `json:"alert_id"`
	PauseSubscriptions bool                                                                           `json:"pause_subscriptions,omitempty"`
	Subscriptions      []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskAlertSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskDashboardSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskDashboard struct {
	CustomSubject      string                                                                             `json:"custom_subject,omitempty"`
	DashboardId        string                                                                             `json:"dashboard_id"`
	PauseSubscriptions bool                                                                               `json:"pause_subscriptions,omitempty"`
	Subscriptions      []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskDashboardSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskFile struct {
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskQuery struct {
	QueryId string `json:"query_id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTask struct {
	Parameters  map[string]string                                                    `json:"parameters,omitempty"`
	WarehouseId string                                                               `json:"warehouse_id"`
	Alert       *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskAlert     `json:"alert,omitempty"`
	Dashboard   *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskDashboard `json:"dashboard,omitempty"`
	File        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskFile      `json:"file,omitempty"`
	Query       *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTaskQuery     `json:"query,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnFailure struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnStart struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnStreamingBacklogExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnSuccess struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnStreamingBacklogExceeded         `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTaskTask struct {
	Description            string                                                                   `json:"description,omitempty"`
	EnvironmentKey         string                                                                   `json:"environment_key,omitempty"`
	ExistingClusterId      string                                                                   `json:"existing_cluster_id,omitempty"`
	JobClusterKey          string                                                                   `json:"job_cluster_key,omitempty"`
	MaxRetries             int                                                                      `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                                                      `json:"min_retry_interval_millis,omitempty"`
	RetryOnTimeout         bool                                                                     `json:"retry_on_timeout,omitempty"`
	RunIf                  string                                                                   `json:"run_if,omitempty"`
	TaskKey                string                                                                   `json:"task_key"`
	TimeoutSeconds         int                                                                      `json:"timeout_seconds,omitempty"`
	ConditionTask          *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskConditionTask        `json:"condition_task,omitempty"`
	DashboardTask          *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDashboardTask        `json:"dashboard_task,omitempty"`
	DbtTask                *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDbtTask              `json:"dbt_task,omitempty"`
	DependsOn              []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskDependsOn           `json:"depends_on,omitempty"`
	EmailNotifications     *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskEmailNotifications   `json:"email_notifications,omitempty"`
	Health                 *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskHealth               `json:"health,omitempty"`
	Library                []DataSourceJobJobSettingsSettingsTaskForEachTaskTaskLibrary             `json:"library,omitempty"`
	NewCluster             *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskNotificationSettings `json:"notification_settings,omitempty"`
	PipelineTask           *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPipelineTask         `json:"pipeline_task,omitempty"`
	PowerBiTask            *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPowerBiTask          `json:"power_bi_task,omitempty"`
	PythonWheelTask        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskPythonWheelTask      `json:"python_wheel_task,omitempty"`
	RunJobTask             *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskRunJobTask           `json:"run_job_task,omitempty"`
	SparkJarTask           *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	SqlTask                *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskSqlTask              `json:"sql_task,omitempty"`
	WebhookNotifications   *DataSourceJobJobSettingsSettingsTaskForEachTaskTaskWebhookNotifications `json:"webhook_notifications,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskForEachTask struct {
	Concurrency int                                                  `json:"concurrency,omitempty"`
	Inputs      string                                               `json:"inputs"`
	Task        *DataSourceJobJobSettingsSettingsTaskForEachTaskTask `json:"task,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskHealthRules struct {
	Metric string `json:"metric"`
	Op     string `json:"op"`
	Value  int    `json:"value"`
}

type DataSourceJobJobSettingsSettingsTaskHealth struct {
	Rules []DataSourceJobJobSettingsSettingsTaskHealthRules `json:"rules,omitempty"`
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
	Egg          string                                            `json:"egg,omitempty"`
	Jar          string                                            `json:"jar,omitempty"`
	Requirements string                                            `json:"requirements,omitempty"`
	Whl          string                                            `json:"whl,omitempty"`
	Cran         *DataSourceJobJobSettingsSettingsTaskLibraryCran  `json:"cran,omitempty"`
	Maven        *DataSourceJobJobSettingsSettingsTaskLibraryMaven `json:"maven,omitempty"`
	Pypi         *DataSourceJobJobSettingsSettingsTaskLibraryPypi  `json:"pypi,omitempty"`
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
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
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

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceJobJobSettingsSettingsTaskNewClusterInitScripts struct {
	Abfss     *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceJobJobSettingsSettingsTaskNewClusterInitScriptsVolumes   `json:"volumes,omitempty"`
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
	SparkVersion              string                                                           `json:"spark_version,omitempty"`
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
	WarehouseId    string            `json:"warehouse_id,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskNotificationSettings struct {
	AlertOnLastAttempt     bool `json:"alert_on_last_attempt,omitempty"`
	NoAlertForCanceledRuns bool `json:"no_alert_for_canceled_runs,omitempty"`
	NoAlertForSkippedRuns  bool `json:"no_alert_for_skipped_runs,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskPipelineTask struct {
	FullRefresh bool   `json:"full_refresh,omitempty"`
	PipelineId  string `json:"pipeline_id"`
}

type DataSourceJobJobSettingsSettingsTaskPowerBiTaskPowerBiModel struct {
	AuthenticationMethod string `json:"authentication_method,omitempty"`
	ModelName            string `json:"model_name,omitempty"`
	OverwriteExisting    bool   `json:"overwrite_existing,omitempty"`
	StorageMode          string `json:"storage_mode,omitempty"`
	WorkspaceName        string `json:"workspace_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskPowerBiTaskTables struct {
	Catalog     string `json:"catalog,omitempty"`
	Name        string `json:"name,omitempty"`
	Schema      string `json:"schema,omitempty"`
	StorageMode string `json:"storage_mode,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskPowerBiTask struct {
	ConnectionResourceName string                                                       `json:"connection_resource_name,omitempty"`
	RefreshAfterUpdate     bool                                                         `json:"refresh_after_update,omitempty"`
	WarehouseId            string                                                       `json:"warehouse_id,omitempty"`
	PowerBiModel           *DataSourceJobJobSettingsSettingsTaskPowerBiTaskPowerBiModel `json:"power_bi_model,omitempty"`
	Tables                 []DataSourceJobJobSettingsSettingsTaskPowerBiTaskTables      `json:"tables,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskPythonWheelTask struct {
	EntryPoint      string            `json:"entry_point,omitempty"`
	NamedParameters map[string]string `json:"named_parameters,omitempty"`
	PackageName     string            `json:"package_name,omitempty"`
	Parameters      []string          `json:"parameters,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskRunJobTask struct {
	JobId         int               `json:"job_id"`
	JobParameters map[string]string `json:"job_parameters,omitempty"`
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

type DataSourceJobJobSettingsSettingsTaskSqlTaskAlertSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskAlert struct {
	AlertId            string                                                          `json:"alert_id"`
	PauseSubscriptions bool                                                            `json:"pause_subscriptions,omitempty"`
	Subscriptions      []DataSourceJobJobSettingsSettingsTaskSqlTaskAlertSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskDashboardSubscriptions struct {
	DestinationId string `json:"destination_id,omitempty"`
	UserName      string `json:"user_name,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskDashboard struct {
	CustomSubject      string                                                              `json:"custom_subject,omitempty"`
	DashboardId        string                                                              `json:"dashboard_id"`
	PauseSubscriptions bool                                                                `json:"pause_subscriptions,omitempty"`
	Subscriptions      []DataSourceJobJobSettingsSettingsTaskSqlTaskDashboardSubscriptions `json:"subscriptions,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskFile struct {
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTaskQuery struct {
	QueryId string `json:"query_id"`
}

type DataSourceJobJobSettingsSettingsTaskSqlTask struct {
	Parameters  map[string]string                                     `json:"parameters,omitempty"`
	WarehouseId string                                                `json:"warehouse_id"`
	Alert       *DataSourceJobJobSettingsSettingsTaskSqlTaskAlert     `json:"alert,omitempty"`
	Dashboard   *DataSourceJobJobSettingsSettingsTaskSqlTaskDashboard `json:"dashboard,omitempty"`
	File        *DataSourceJobJobSettingsSettingsTaskSqlTaskFile      `json:"file,omitempty"`
	Query       *DataSourceJobJobSettingsSettingsTaskSqlTaskQuery     `json:"query,omitempty"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnFailure struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnStart struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnStreamingBacklogExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnSuccess struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsTaskWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnStreamingBacklogExceeded         `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []DataSourceJobJobSettingsSettingsTaskWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettingsTask struct {
	Description            string                                                    `json:"description,omitempty"`
	EnvironmentKey         string                                                    `json:"environment_key,omitempty"`
	ExistingClusterId      string                                                    `json:"existing_cluster_id,omitempty"`
	JobClusterKey          string                                                    `json:"job_cluster_key,omitempty"`
	MaxRetries             int                                                       `json:"max_retries,omitempty"`
	MinRetryIntervalMillis int                                                       `json:"min_retry_interval_millis,omitempty"`
	RetryOnTimeout         bool                                                      `json:"retry_on_timeout,omitempty"`
	RunIf                  string                                                    `json:"run_if,omitempty"`
	TaskKey                string                                                    `json:"task_key"`
	TimeoutSeconds         int                                                       `json:"timeout_seconds,omitempty"`
	ConditionTask          *DataSourceJobJobSettingsSettingsTaskConditionTask        `json:"condition_task,omitempty"`
	DashboardTask          *DataSourceJobJobSettingsSettingsTaskDashboardTask        `json:"dashboard_task,omitempty"`
	DbtTask                *DataSourceJobJobSettingsSettingsTaskDbtTask              `json:"dbt_task,omitempty"`
	DependsOn              []DataSourceJobJobSettingsSettingsTaskDependsOn           `json:"depends_on,omitempty"`
	EmailNotifications     *DataSourceJobJobSettingsSettingsTaskEmailNotifications   `json:"email_notifications,omitempty"`
	ForEachTask            *DataSourceJobJobSettingsSettingsTaskForEachTask          `json:"for_each_task,omitempty"`
	Health                 *DataSourceJobJobSettingsSettingsTaskHealth               `json:"health,omitempty"`
	Library                []DataSourceJobJobSettingsSettingsTaskLibrary             `json:"library,omitempty"`
	NewCluster             *DataSourceJobJobSettingsSettingsTaskNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *DataSourceJobJobSettingsSettingsTaskNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *DataSourceJobJobSettingsSettingsTaskNotificationSettings `json:"notification_settings,omitempty"`
	PipelineTask           *DataSourceJobJobSettingsSettingsTaskPipelineTask         `json:"pipeline_task,omitempty"`
	PowerBiTask            *DataSourceJobJobSettingsSettingsTaskPowerBiTask          `json:"power_bi_task,omitempty"`
	PythonWheelTask        *DataSourceJobJobSettingsSettingsTaskPythonWheelTask      `json:"python_wheel_task,omitempty"`
	RunJobTask             *DataSourceJobJobSettingsSettingsTaskRunJobTask           `json:"run_job_task,omitempty"`
	SparkJarTask           *DataSourceJobJobSettingsSettingsTaskSparkJarTask         `json:"spark_jar_task,omitempty"`
	SparkPythonTask        *DataSourceJobJobSettingsSettingsTaskSparkPythonTask      `json:"spark_python_task,omitempty"`
	SparkSubmitTask        *DataSourceJobJobSettingsSettingsTaskSparkSubmitTask      `json:"spark_submit_task,omitempty"`
	SqlTask                *DataSourceJobJobSettingsSettingsTaskSqlTask              `json:"sql_task,omitempty"`
	WebhookNotifications   *DataSourceJobJobSettingsSettingsTaskWebhookNotifications `json:"webhook_notifications,omitempty"`
}

type DataSourceJobJobSettingsSettingsTriggerFileArrival struct {
	MinTimeBetweenTriggersSeconds int    `json:"min_time_between_triggers_seconds,omitempty"`
	Url                           string `json:"url"`
	WaitAfterLastChangeSeconds    int    `json:"wait_after_last_change_seconds,omitempty"`
}

type DataSourceJobJobSettingsSettingsTriggerPeriodic struct {
	Interval int    `json:"interval"`
	Unit     string `json:"unit"`
}

type DataSourceJobJobSettingsSettingsTriggerTableUpdate struct {
	Condition                     string   `json:"condition,omitempty"`
	MinTimeBetweenTriggersSeconds int      `json:"min_time_between_triggers_seconds,omitempty"`
	TableNames                    []string `json:"table_names"`
	WaitAfterLastChangeSeconds    int      `json:"wait_after_last_change_seconds,omitempty"`
}

type DataSourceJobJobSettingsSettingsTrigger struct {
	PauseStatus string                                              `json:"pause_status,omitempty"`
	FileArrival *DataSourceJobJobSettingsSettingsTriggerFileArrival `json:"file_arrival,omitempty"`
	Periodic    *DataSourceJobJobSettingsSettingsTriggerPeriodic    `json:"periodic,omitempty"`
	TableUpdate *DataSourceJobJobSettingsSettingsTriggerTableUpdate `json:"table_update,omitempty"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnDurationWarningThresholdExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnFailure struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnStart struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnStreamingBacklogExceeded struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotificationsOnSuccess struct {
	Id string `json:"id"`
}

type DataSourceJobJobSettingsSettingsWebhookNotifications struct {
	OnDurationWarningThresholdExceeded []DataSourceJobJobSettingsSettingsWebhookNotificationsOnDurationWarningThresholdExceeded `json:"on_duration_warning_threshold_exceeded,omitempty"`
	OnFailure                          []DataSourceJobJobSettingsSettingsWebhookNotificationsOnFailure                          `json:"on_failure,omitempty"`
	OnStart                            []DataSourceJobJobSettingsSettingsWebhookNotificationsOnStart                            `json:"on_start,omitempty"`
	OnStreamingBacklogExceeded         []DataSourceJobJobSettingsSettingsWebhookNotificationsOnStreamingBacklogExceeded         `json:"on_streaming_backlog_exceeded,omitempty"`
	OnSuccess                          []DataSourceJobJobSettingsSettingsWebhookNotificationsOnSuccess                          `json:"on_success,omitempty"`
}

type DataSourceJobJobSettingsSettings struct {
	Description            string                                                `json:"description,omitempty"`
	EditMode               string                                                `json:"edit_mode,omitempty"`
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
	Deployment             *DataSourceJobJobSettingsSettingsDeployment           `json:"deployment,omitempty"`
	EmailNotifications     *DataSourceJobJobSettingsSettingsEmailNotifications   `json:"email_notifications,omitempty"`
	Environment            []DataSourceJobJobSettingsSettingsEnvironment         `json:"environment,omitempty"`
	GitSource              *DataSourceJobJobSettingsSettingsGitSource            `json:"git_source,omitempty"`
	Health                 *DataSourceJobJobSettingsSettingsHealth               `json:"health,omitempty"`
	JobCluster             []DataSourceJobJobSettingsSettingsJobCluster          `json:"job_cluster,omitempty"`
	Library                []DataSourceJobJobSettingsSettingsLibrary             `json:"library,omitempty"`
	NewCluster             *DataSourceJobJobSettingsSettingsNewCluster           `json:"new_cluster,omitempty"`
	NotebookTask           *DataSourceJobJobSettingsSettingsNotebookTask         `json:"notebook_task,omitempty"`
	NotificationSettings   *DataSourceJobJobSettingsSettingsNotificationSettings `json:"notification_settings,omitempty"`
	Parameter              []DataSourceJobJobSettingsSettingsParameter           `json:"parameter,omitempty"`
	PipelineTask           *DataSourceJobJobSettingsSettingsPipelineTask         `json:"pipeline_task,omitempty"`
	PythonWheelTask        *DataSourceJobJobSettingsSettingsPythonWheelTask      `json:"python_wheel_task,omitempty"`
	Queue                  *DataSourceJobJobSettingsSettingsQueue                `json:"queue,omitempty"`
	RunAs                  *DataSourceJobJobSettingsSettingsRunAs                `json:"run_as,omitempty"`
	RunJobTask             *DataSourceJobJobSettingsSettingsRunJobTask           `json:"run_job_task,omitempty"`
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
