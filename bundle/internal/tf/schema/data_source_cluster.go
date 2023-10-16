// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceClusterClusterInfoAutoscale struct {
	MaxWorkers int `json:"max_workers,omitempty"`
	MinWorkers int `json:"min_workers,omitempty"`
}

type DataSourceClusterClusterInfoAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceClusterClusterInfoAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
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

type DataSourceClusterClusterInfoClusterLogConf struct {
	Dbfs *DataSourceClusterClusterInfoClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *DataSourceClusterClusterInfoClusterLogConfS3   `json:"s3,omitempty"`
}

type DataSourceClusterClusterInfoClusterLogStatus struct {
	LastAttempted int    `json:"last_attempted,omitempty"`
	LastException string `json:"last_exception,omitempty"`
}

type DataSourceClusterClusterInfoDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceClusterClusterInfoDockerImage struct {
	Url       string                                            `json:"url"`
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
	Destination string `json:"destination,omitempty"`
}

type DataSourceClusterClusterInfoInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type DataSourceClusterClusterInfoInitScriptsFile struct {
	Destination string `json:"destination,omitempty"`
}

type DataSourceClusterClusterInfoInitScriptsGcs struct {
	Destination string `json:"destination,omitempty"`
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
	Destination string `json:"destination,omitempty"`
}

type DataSourceClusterClusterInfoInitScriptsWorkspace struct {
	Destination string `json:"destination,omitempty"`
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

type DataSourceClusterClusterInfoTerminationReason struct {
	Code       string            `json:"code,omitempty"`
	Parameters map[string]string `json:"parameters,omitempty"`
	Type       string            `json:"type,omitempty"`
}

type DataSourceClusterClusterInfo struct {
	AutoterminationMinutes    int                                            `json:"autotermination_minutes,omitempty"`
	ClusterCores              int                                            `json:"cluster_cores,omitempty"`
	ClusterId                 string                                         `json:"cluster_id,omitempty"`
	ClusterMemoryMb           int                                            `json:"cluster_memory_mb,omitempty"`
	ClusterName               string                                         `json:"cluster_name,omitempty"`
	ClusterSource             string                                         `json:"cluster_source,omitempty"`
	CreatorUserName           string                                         `json:"creator_user_name,omitempty"`
	CustomTags                map[string]string                              `json:"custom_tags,omitempty"`
	DataSecurityMode          string                                         `json:"data_security_mode,omitempty"`
	DefaultTags               map[string]string                              `json:"default_tags"`
	DriverInstancePoolId      string                                         `json:"driver_instance_pool_id,omitempty"`
	DriverNodeTypeId          string                                         `json:"driver_node_type_id,omitempty"`
	EnableElasticDisk         bool                                           `json:"enable_elastic_disk,omitempty"`
	EnableLocalDiskEncryption bool                                           `json:"enable_local_disk_encryption,omitempty"`
	InstancePoolId            string                                         `json:"instance_pool_id,omitempty"`
	JdbcPort                  int                                            `json:"jdbc_port,omitempty"`
	LastActivityTime          int                                            `json:"last_activity_time,omitempty"`
	LastStateLossTime         int                                            `json:"last_state_loss_time,omitempty"`
	NodeTypeId                string                                         `json:"node_type_id,omitempty"`
	NumWorkers                int                                            `json:"num_workers,omitempty"`
	PolicyId                  string                                         `json:"policy_id,omitempty"`
	RuntimeEngine             string                                         `json:"runtime_engine,omitempty"`
	SingleUserName            string                                         `json:"single_user_name,omitempty"`
	SparkConf                 map[string]string                              `json:"spark_conf,omitempty"`
	SparkContextId            int                                            `json:"spark_context_id,omitempty"`
	SparkEnvVars              map[string]string                              `json:"spark_env_vars,omitempty"`
	SparkVersion              string                                         `json:"spark_version"`
	SshPublicKeys             []string                                       `json:"ssh_public_keys,omitempty"`
	StartTime                 int                                            `json:"start_time,omitempty"`
	State                     string                                         `json:"state"`
	StateMessage              string                                         `json:"state_message,omitempty"`
	TerminateTime             int                                            `json:"terminate_time,omitempty"`
	Autoscale                 *DataSourceClusterClusterInfoAutoscale         `json:"autoscale,omitempty"`
	AwsAttributes             *DataSourceClusterClusterInfoAwsAttributes     `json:"aws_attributes,omitempty"`
	AzureAttributes           *DataSourceClusterClusterInfoAzureAttributes   `json:"azure_attributes,omitempty"`
	ClusterLogConf            *DataSourceClusterClusterInfoClusterLogConf    `json:"cluster_log_conf,omitempty"`
	ClusterLogStatus          *DataSourceClusterClusterInfoClusterLogStatus  `json:"cluster_log_status,omitempty"`
	DockerImage               *DataSourceClusterClusterInfoDockerImage       `json:"docker_image,omitempty"`
	Driver                    *DataSourceClusterClusterInfoDriver            `json:"driver,omitempty"`
	Executors                 []DataSourceClusterClusterInfoExecutors        `json:"executors,omitempty"`
	GcpAttributes             *DataSourceClusterClusterInfoGcpAttributes     `json:"gcp_attributes,omitempty"`
	InitScripts               []DataSourceClusterClusterInfoInitScripts      `json:"init_scripts,omitempty"`
	TerminationReason         *DataSourceClusterClusterInfoTerminationReason `json:"termination_reason,omitempty"`
}

type DataSourceCluster struct {
	ClusterId   string                        `json:"cluster_id,omitempty"`
	ClusterName string                        `json:"cluster_name,omitempty"`
	Id          string                        `json:"id,omitempty"`
	ClusterInfo *DataSourceClusterClusterInfo `json:"cluster_info,omitempty"`
}
