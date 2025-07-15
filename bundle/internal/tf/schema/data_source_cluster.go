// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceClusterClusterInfoAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceClusterClusterInfoAwsAttributes struct {
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

type DataSourceClusterClusterInfoAzureAttributesLogAnalyticsInfo struct {
	LogAnalyticsPrimaryKey  string `json:"log_analytics_primary_key,omitempty"`
	LogAnalyticsWorkspaceId string `json:"log_analytics_workspace_id,omitempty"`
}

type DataSourceClusterClusterInfoAzureAttributes struct {
	Availability     string                                                       `json:"availability,omitempty"`
	FirstOnDemand    int                                                          `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice  int                                                          `json:"spot_bid_max_price,omitempty"`
	LogAnalyticsInfo *DataSourceClusterClusterInfoAzureAttributesLogAnalyticsInfo `json:"log_analytics_info,omitempty"`
}

type DataSourceClusterClusterInfoClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceClusterClusterInfoClusterLogConfVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoClusterLogConf struct {
	Dbfs    *DataSourceClusterClusterInfoClusterLogConfDbfs    `json:"dbfs,omitempty"`
	S3      *DataSourceClusterClusterInfoClusterLogConfS3      `json:"s3,omitempty"`
	Volumes *DataSourceClusterClusterInfoClusterLogConfVolumes `json:"volumes,omitempty"`
}

type DataSourceClusterClusterInfoClusterLogStatus struct {
	LastAttempted int    `json:"last_attempted,omitempty"`
	LastException string `json:"last_exception,omitempty"`
}

type DataSourceClusterClusterInfoDockerImageBasicAuth struct {
	Password string `json:"password,omitempty"`
	Username string `json:"username,omitempty"`
}

type DataSourceClusterClusterInfoDockerImage struct {
	Url       string                                            `json:"url,omitempty"`
	BasicAuth *DataSourceClusterClusterInfoDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceClusterClusterInfoDriverNodeAwsAttributes struct {
	IsSpot bool `json:"is_spot,omitempty"`
}

type DataSourceClusterClusterInfoDriver struct {
	HostPrivateIp     string                                               `json:"host_private_ip,omitempty"`
	InstanceId        string                                               `json:"instance_id,omitempty"`
	NodeId            string                                               `json:"node_id,omitempty"`
	PrivateIp         string                                               `json:"private_ip,omitempty"`
	PublicDns         string                                               `json:"public_dns,omitempty"`
	StartTimestamp    int                                                  `json:"start_timestamp,omitempty"`
	NodeAwsAttributes *DataSourceClusterClusterInfoDriverNodeAwsAttributes `json:"node_aws_attributes,omitempty"`
}

type DataSourceClusterClusterInfoExecutorsNodeAwsAttributes struct {
	IsSpot bool `json:"is_spot,omitempty"`
}

type DataSourceClusterClusterInfoExecutors struct {
	HostPrivateIp     string                                                  `json:"host_private_ip,omitempty"`
	InstanceId        string                                                  `json:"instance_id,omitempty"`
	NodeId            string                                                  `json:"node_id,omitempty"`
	PrivateIp         string                                                  `json:"private_ip,omitempty"`
	PublicDns         string                                                  `json:"public_dns,omitempty"`
	StartTimestamp    int                                                     `json:"start_timestamp,omitempty"`
	NodeAwsAttributes *DataSourceClusterClusterInfoExecutorsNodeAwsAttributes `json:"node_aws_attributes,omitempty"`
}

type DataSourceClusterClusterInfoGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceClusterClusterInfoInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceClusterClusterInfoInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScripts struct {
	Abfss     *DataSourceClusterClusterInfoInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceClusterClusterInfoInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceClusterClusterInfoInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceClusterClusterInfoInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceClusterClusterInfoInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceClusterClusterInfoInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *DataSourceClusterClusterInfoInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceClusterClusterInfoSpecAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceClusterClusterInfoSpecAwsAttributes struct {
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

type DataSourceClusterClusterInfoSpecAzureAttributesLogAnalyticsInfo struct {
	LogAnalyticsPrimaryKey  string `json:"log_analytics_primary_key,omitempty"`
	LogAnalyticsWorkspaceId string `json:"log_analytics_workspace_id,omitempty"`
}

type DataSourceClusterClusterInfoSpecAzureAttributes struct {
	Availability     string                                                           `json:"availability,omitempty"`
	FirstOnDemand    int                                                              `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice  int                                                              `json:"spot_bid_max_price,omitempty"`
	LogAnalyticsInfo *DataSourceClusterClusterInfoSpecAzureAttributesLogAnalyticsInfo `json:"log_analytics_info,omitempty"`
}

type DataSourceClusterClusterInfoSpecClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceClusterClusterInfoSpecClusterLogConfVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecClusterLogConf struct {
	Dbfs    *DataSourceClusterClusterInfoSpecClusterLogConfDbfs    `json:"dbfs,omitempty"`
	S3      *DataSourceClusterClusterInfoSpecClusterLogConfS3      `json:"s3,omitempty"`
	Volumes *DataSourceClusterClusterInfoSpecClusterLogConfVolumes `json:"volumes,omitempty"`
}

type DataSourceClusterClusterInfoSpecClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type DataSourceClusterClusterInfoSpecClusterMountInfo struct {
	LocalMountDirPath     string                                                                 `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                                 `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *DataSourceClusterClusterInfoSpecClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type DataSourceClusterClusterInfoSpecDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceClusterClusterInfoSpecDockerImage struct {
	Url       string                                                `json:"url"`
	BasicAuth *DataSourceClusterClusterInfoSpecDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceClusterClusterInfoSpecGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type DataSourceClusterClusterInfoSpecInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScriptsFile struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type DataSourceClusterClusterInfoSpecInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoSpecInitScripts struct {
	Abfss     *DataSourceClusterClusterInfoSpecInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *DataSourceClusterClusterInfoSpecInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *DataSourceClusterClusterInfoSpecInitScriptsFile      `json:"file,omitempty"`
	Gcs       *DataSourceClusterClusterInfoSpecInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *DataSourceClusterClusterInfoSpecInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *DataSourceClusterClusterInfoSpecInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *DataSourceClusterClusterInfoSpecInitScriptsWorkspace `json:"workspace,omitempty"`
}

type DataSourceClusterClusterInfoSpecLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceClusterClusterInfoSpecLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type DataSourceClusterClusterInfoSpecLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type DataSourceClusterClusterInfoSpecLibrary struct {
	Egg          string                                        `json:"egg,omitempty"`
	Jar          string                                        `json:"jar,omitempty"`
	Requirements string                                        `json:"requirements,omitempty"`
	Whl          string                                        `json:"whl,omitempty"`
	Cran         *DataSourceClusterClusterInfoSpecLibraryCran  `json:"cran,omitempty"`
	Maven        *DataSourceClusterClusterInfoSpecLibraryMaven `json:"maven,omitempty"`
	Pypi         *DataSourceClusterClusterInfoSpecLibraryPypi  `json:"pypi,omitempty"`
}

type DataSourceClusterClusterInfoSpecWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceClusterClusterInfoSpecWorkloadType struct {
	Clients *DataSourceClusterClusterInfoSpecWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceClusterClusterInfoSpec struct {
	ApplyPolicyDefaultValues   bool                                               `json:"apply_policy_default_values,omitempty"`
	ClusterId                  string                                             `json:"cluster_id,omitempty"`
	ClusterName                string                                             `json:"cluster_name,omitempty"`
	CustomTags                 map[string]string                                  `json:"custom_tags,omitempty"`
	DataSecurityMode           string                                             `json:"data_security_mode,omitempty"`
	DriverInstancePoolId       string                                             `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId           string                                             `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk          bool                                               `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption  bool                                               `json:"enable_local_disk_encryption,omitempty"`
	IdempotencyToken           string                                             `json:"idempotency_token,omitempty"`
	InstancePoolId             string                                             `json:"instance_pool_id,omitempty"`
	IsSingleNode               bool                                               `json:"is_single_node,omitempty"`
	Kind                       string                                             `json:"kind,omitempty"`
	NodeTypeId                 string                                             `json:"node_type_id,omitempty"`
	NumWorkers                 int                                                `json:"num_workers,omitempty"`
	PolicyId                   string                                             `json:"policy_id,omitempty"`
	RemoteDiskThroughput       int                                                `json:"remote_disk_throughput,omitempty"`
	RuntimeEngine              string                                             `json:"runtime_engine,omitempty"`
	SingleUserName             string                                             `json:"single_user_name,omitempty"`
	SparkConf                  map[string]string                                  `json:"spark_conf,omitempty"`
	SparkEnvVars               map[string]string                                  `json:"spark_env_vars,omitempty"`
	SparkVersion               string                                             `json:"spark_version,omitempty"`
	SshPublicKeys              []string                                           `json:"ssh_public_keys,omitempty"`
	TotalInitialRemoteDiskSize int                                                `json:"total_initial_remote_disk_size,omitempty"`
	UseMlRuntime               bool                                               `json:"use_ml_runtime,omitempty"`
	Autoscale                  *DataSourceClusterClusterInfoSpecAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes              *DataSourceClusterClusterInfoSpecAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes            *DataSourceClusterClusterInfoSpecAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf             *DataSourceClusterClusterInfoSpecClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo           []DataSourceClusterClusterInfoSpecClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage                *DataSourceClusterClusterInfoSpecDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes              *DataSourceClusterClusterInfoSpecGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts                []DataSourceClusterClusterInfoSpecInitScripts      `json:"init_scripts,omitempty"`
	Library                    []DataSourceClusterClusterInfoSpecLibrary          `json:"library,omitempty"`
	WorkloadType               *DataSourceClusterClusterInfoSpecWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceClusterClusterInfoTerminationReason struct {
	Code       string            `json:"code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Type       string            `json:"type,omitempty"`
}

type DataSourceClusterClusterInfoWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type DataSourceClusterClusterInfoWorkloadType struct {
	Clients *DataSourceClusterClusterInfoWorkloadTypeClients `json:"clients,omitempty"`
}

type DataSourceClusterClusterInfo struct {
	AutoterminationMinutes     int                                            `json:"autotermination_minutes,omitempty"`
	ClusterCores               int                                            `json:"cluster_cores,omitempty"`
	ClusterId                  string                                         `json:"cluster_id,omitempty"`
	ClusterMemoryMb            int                                            `json:"cluster_memory_mb,omitempty"`
	ClusterName                string                                         `json:"cluster_name,omitempty"`
	ClusterSource              string                                         `json:"cluster_source,omitempty"`
	CreatorUserName            string                                         `json:"creator_user_name,omitempty"`
	CustomTags                 map[string]string                              `json:"custom_tags,omitempty"`
	DataSecurityMode           string                                         `json:"data_security_mode,omitempty"`
	DefaultTags                map[string]string                              `json:"default_tags,omitempty"`
	DriverInstancePoolId       string                                         `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId           string                                         `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk          bool                                           `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption  bool                                           `json:"enable_local_disk_encryption,omitempty"`
	InstancePoolId             string                                         `json:"instance_pool_id,omitempty"`
	IsSingleNode               bool                                           `json:"is_single_node,omitempty"`
	JdbcPort                   int                                            `json:"jdbc_port,omitempty"`
	Kind                       string                                         `json:"kind,omitempty"`
	LastRestartedTime          int                                            `json:"last_restarted_time,omitempty"`
	LastStateLossTime          int                                            `json:"last_state_loss_time,omitempty"`
	NodeTypeId                 string                                         `json:"node_type_id,omitempty"`
	NumWorkers                 int                                            `json:"num_workers,omitempty"`
	PolicyId                   string                                         `json:"policy_id,omitempty"`
	RemoteDiskThroughput       int                                            `json:"remote_disk_throughput,omitempty"`
	RuntimeEngine              string                                         `json:"runtime_engine,omitempty"`
	SingleUserName             string                                         `json:"single_user_name,omitempty"`
	SparkConf                  map[string]string                              `json:"spark_conf,omitempty"`
	SparkContextId             int                                            `json:"spark_context_id,omitempty"`
	SparkEnvVars               map[string]string                              `json:"spark_env_vars,omitempty"`
	SparkVersion               string                                         `json:"spark_version,omitempty"`
	SshPublicKeys              []string                                       `json:"ssh_public_keys,omitempty"`
	StartTime                  int                                            `json:"start_time,omitempty"`
	State                      string                                         `json:"state,omitempty"`
	StateMessage               string                                         `json:"state_message,omitempty"`
	TerminatedTime             int                                            `json:"terminated_time,omitempty"`
	TotalInitialRemoteDiskSize int                                            `json:"total_initial_remote_disk_size,omitempty"`
	UseMlRuntime               bool                                           `json:"use_ml_runtime,omitempty"`
	Autoscale                  *DataSourceClusterClusterInfoAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes              *DataSourceClusterClusterInfoAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes            *DataSourceClusterClusterInfoAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf             *DataSourceClusterClusterInfoClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterLogStatus           *DataSourceClusterClusterInfoClusterLogStatus  `json:"cluster_log_status,omitempty"`
	DockerImage                *DataSourceClusterClusterInfoDockerImage       `json:"docker_image,omitempty"`
	Driver                     *DataSourceClusterClusterInfoDriver            `json:"driver,omitempty"`
	Executors                  []DataSourceClusterClusterInfoExecutors        `json:"executors,omitempty"`
	GcpAttributes              *DataSourceClusterClusterInfoGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts                []DataSourceClusterClusterInfoInitScripts      `json:"init_scripts,omitempty"`
	Spec                       *DataSourceClusterClusterInfoSpec              `json:"spec,omitempty"`
	TerminationReason          *DataSourceClusterClusterInfoTerminationReason `json:"termination_reason,omitempty"`
	WorkloadType               *DataSourceClusterClusterInfoWorkloadType      `json:"workload_type,omitempty"`
}

type DataSourceCluster struct {
	ClusterId   string                        `json:"cluster_id,omitempty"`
	ClusterName string                        `json:"cluster_name,omitempty"`
	Id          string                        `json:"id,omitempty"`
	ClusterInfo *DataSourceClusterClusterInfo `json:"cluster_info,omitempty"`
}
