// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceClusterAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type ResourceClusterAwsAttributes struct {
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

type ResourceClusterAzureAttributesLogAnalyticsInfo struct {
	LogAnalyticsPrimaryKey  string `json:"log_analytics_primary_key,omitempty"`
	LogAnalyticsWorkspaceId string `json:"log_analytics_workspace_id,omitempty"`
}

type ResourceClusterAzureAttributes struct {
	Availability     string                                          `json:"availability,omitempty"`
	FirstOnDemand    int                                             `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice  int                                             `json:"spot_bid_max_price,omitempty"`
	LogAnalyticsInfo *ResourceClusterAzureAttributesLogAnalyticsInfo `json:"log_analytics_info,omitempty"`
}

type ResourceClusterClusterLogConfDbfs struct {
	Destination string `json:"destination"`
}

type ResourceClusterClusterLogConfS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceClusterClusterLogConfVolumes struct {
	Destination string `json:"destination"`
}

type ResourceClusterClusterLogConf struct {
	Dbfs    *ResourceClusterClusterLogConfDbfs    `json:"dbfs,omitempty"`
	S3      *ResourceClusterClusterLogConfS3      `json:"s3,omitempty"`
	Volumes *ResourceClusterClusterLogConfVolumes `json:"volumes,omitempty"`
}

type ResourceClusterClusterMountInfoNetworkFilesystemInfo struct {
	MountOptions  string `json:"mount_options,omitempty"`
	ServerAddress string `json:"server_address"`
}

type ResourceClusterClusterMountInfo struct {
	LocalMountDirPath     string                                                `json:"local_mount_dir_path"`
	RemoteMountDirPath    string                                                `json:"remote_mount_dir_path,omitempty"`
	NetworkFilesystemInfo *ResourceClusterClusterMountInfoNetworkFilesystemInfo `json:"network_filesystem_info,omitempty"`
}

type ResourceClusterDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceClusterDockerImage struct {
	Url       string                               `json:"url"`
	BasicAuth *ResourceClusterDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceClusterGcpAttributes struct {
	Availability            string `json:"availability,omitempty"`
	BootDiskSize            int    `json:"boot_disk_size,omitempty"`
	GoogleServiceAccount    string `json:"google_service_account,omitempty"`
	LocalSsdCount           int    `json:"local_ssd_count,omitempty"`
	UsePreemptibleExecutors bool   `json:"use_preemptible_executors,omitempty"`
	ZoneId                  string `json:"zone_id,omitempty"`
}

type ResourceClusterInitScriptsAbfss struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScriptsFile struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScriptsGcs struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScriptsS3 struct {
	CannedAcl        string `json:"canned_acl,omitempty"`
	Destination      string `json:"destination"`
	EnableEncryption bool   `json:"enable_encryption,omitempty"`
	EncryptionType   string `json:"encryption_type,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	KmsKey           string `json:"kms_key,omitempty"`
	Region           string `json:"region,omitempty"`
}

type ResourceClusterInitScriptsVolumes struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScriptsWorkspace struct {
	Destination string `json:"destination"`
}

type ResourceClusterInitScripts struct {
	Abfss     *ResourceClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourceClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourceClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourceClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourceClusterInitScriptsS3        `json:"s3,omitempty"`
	Volumes   *ResourceClusterInitScriptsVolumes   `json:"volumes,omitempty"`
	Workspace *ResourceClusterInitScriptsWorkspace `json:"workspace,omitempty"`
}

type ResourceClusterLibraryCran struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceClusterLibraryMaven struct {
	Coordinates string   `json:"coordinates"`
	Exclusions  []string `json:"exclusions,omitempty"`
	Repo        string   `json:"repo,omitempty"`
}

type ResourceClusterLibraryPypi struct {
	Package string `json:"package"`
	Repo    string `json:"repo,omitempty"`
}

type ResourceClusterLibrary struct {
	Egg          string                       `json:"egg,omitempty"`
	Jar          string                       `json:"jar,omitempty"`
	Requirements string                       `json:"requirements,omitempty"`
	Whl          string                       `json:"whl,omitempty"`
	Cran         *ResourceClusterLibraryCran  `json:"cran,omitempty"`
	Maven        *ResourceClusterLibraryMaven `json:"maven,omitempty"`
	Pypi         *ResourceClusterLibraryPypi  `json:"pypi,omitempty"`
}

type ResourceClusterWorkloadTypeClients struct {
	Jobs      bool `json:"jobs,omitempty"`
	Notebooks bool `json:"notebooks,omitempty"`
}

type ResourceClusterWorkloadType struct {
	Clients *ResourceClusterWorkloadTypeClients `json:"clients,omitempty"`
}

type ResourceCluster struct {
	ApplyPolicyDefaultValues   bool                              `json:"apply_policy_default_values,omitempty"`
	AutoterminationMinutes     int                               `json:"autotermination_minutes,omitempty"`
	ClusterId                  string                            `json:"cluster_id,omitempty"`
	ClusterName                string                            `json:"cluster_name,omitempty"`
	CustomTags                 map[string]string                 `json:"custom_tags,omitempty"`
	DataSecurityMode           string                            `json:"data_security_mode,omitempty"`
	DefaultTags                map[string]string                 `json:"default_tags,omitempty"`
	DriverInstancePoolId       string                            `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId           string                            `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk          bool                              `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption  bool                              `json:"enable_local_disk_encryption,omitempty"`
	Id                         string                            `json:"id,omitempty"`
	IdempotencyToken           string                            `json:"idempotency_token,omitempty"`
	InstancePoolId             string                            `json:"instance_pool_id,omitempty"`
	IsPinned                   bool                              `json:"is_pinned,omitempty"`
	IsSingleNode               bool                              `json:"is_single_node,omitempty"`
	Kind                       string                            `json:"kind,omitempty"`
	NoWait                     bool                              `json:"no_wait,omitempty"`
	NodeTypeId                 string                            `json:"node_type_id,omitempty"`
	NumWorkers                 int                               `json:"num_workers,omitempty"`
	PolicyId                   string                            `json:"policy_id,omitempty"`
	RemoteDiskThroughput       int                               `json:"remote_disk_throughput,omitempty"`
	RuntimeEngine              string                            `json:"runtime_engine,omitempty"`
	SingleUserName             string                            `json:"single_user_name,omitempty"`
	SparkConf                  map[string]string                 `json:"spark_conf,omitempty"`
	SparkEnvVars               map[string]string                 `json:"spark_env_vars,omitempty"`
	SparkVersion               string                            `json:"spark_version"`
	SshPublicKeys              []string                          `json:"ssh_public_keys,omitempty"`
	State                      string                            `json:"state,omitempty"`
	TotalInitialRemoteDiskSize int                               `json:"total_initial_remote_disk_size,omitempty"`
	Url                        string                            `json:"url,omitempty"`
	UseMlRuntime               bool                              `json:"use_ml_runtime,omitempty"`
	Autoscale                  *ResourceClusterAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes              *ResourceClusterAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes            *ResourceClusterAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf             *ResourceClusterClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterMountInfo           []ResourceClusterClusterMountInfo `json:"cluster_mount_info,omitempty"`
	DockerImage                *ResourceClusterDockerImage       `json:"docker_image,omitempty"`
	GcpAttributes              *ResourceClusterGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts                []ResourceClusterInitScripts      `json:"init_scripts,omitempty"`
	Library                    []ResourceClusterLibrary          `json:"library,omitempty"`
	WorkloadType               *ResourceClusterWorkloadType      `json:"workload_type,omitempty"`
}
