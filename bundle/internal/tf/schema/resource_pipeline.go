// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourcePipelineClusterAutoscale struct {
	MaxWorkers int    `json:"max_workers,omitempty"`
	MinWorkers int    `json:"min_workers,omitempty"`
	Mode       string `json:"mode,omitempty"`
}

type ResourcePipelineClusterAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	EbsVolumeCount      int    `json:"ebs_volume_count,omitempty"`
	EbsVolumeSize       int    `json:"ebs_volume_size,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
	FirstOnDemand       int    `json:"first_on_demand,omitempty"`
	InstanceProfileArn  string `json:"instance_profile_arn,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourcePipelineClusterAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	FirstOnDemand   int    `json:"first_on_demand,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
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

type ResourcePipelineClusterClusterLogConf struct {
	Dbfs *ResourcePipelineClusterClusterLogConfDbfs `json:"dbfs,omitempty"`
	S3   *ResourcePipelineClusterClusterLogConfS3   `json:"s3,omitempty"`
}

type ResourcePipelineClusterGcpAttributes struct {
	Availability         string `json:"availability,omitempty"`
	GoogleServiceAccount string `json:"google_service_account,omitempty"`
	LocalSsdCount        int    `json:"local_ssd_count,omitempty"`
	ZoneId               string `json:"zone_id,omitempty"`
}

type ResourcePipelineClusterInitScriptsAbfss struct {
	Destination string `json:"destination,omitempty"`
}

type ResourcePipelineClusterInitScriptsDbfs struct {
	Destination string `json:"destination"`
}

type ResourcePipelineClusterInitScriptsFile struct {
	Destination string `json:"destination,omitempty"`
}

type ResourcePipelineClusterInitScriptsGcs struct {
	Destination string `json:"destination,omitempty"`
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

type ResourcePipelineClusterInitScriptsWorkspace struct {
	Destination string `json:"destination,omitempty"`
}

type ResourcePipelineClusterInitScripts struct {
	Abfss     *ResourcePipelineClusterInitScriptsAbfss     `json:"abfss,omitempty"`
	Dbfs      *ResourcePipelineClusterInitScriptsDbfs      `json:"dbfs,omitempty"`
	File      *ResourcePipelineClusterInitScriptsFile      `json:"file,omitempty"`
	Gcs       *ResourcePipelineClusterInitScriptsGcs       `json:"gcs,omitempty"`
	S3        *ResourcePipelineClusterInitScriptsS3        `json:"s3,omitempty"`
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

type ResourcePipelineFilters struct {
	Exclude []string `json:"exclude,omitempty"`
	Include []string `json:"include,omitempty"`
}

type ResourcePipelineLibraryFile struct {
	Path string `json:"path"`
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
	Maven    *ResourcePipelineLibraryMaven    `json:"maven,omitempty"`
	Notebook *ResourcePipelineLibraryNotebook `json:"notebook,omitempty"`
}

type ResourcePipelineNotification struct {
	Alerts          []string `json:"alerts"`
	EmailRecipients []string `json:"email_recipients"`
}

type ResourcePipeline struct {
	AllowDuplicateNames bool                           `json:"allow_duplicate_names,omitempty"`
	Catalog             string                         `json:"catalog,omitempty"`
	Channel             string                         `json:"channel,omitempty"`
	Configuration       map[string]string              `json:"configuration,omitempty"`
	Continuous          bool                           `json:"continuous,omitempty"`
	Development         bool                           `json:"development,omitempty"`
	Edition             string                         `json:"edition,omitempty"`
	Id                  string                         `json:"id,omitempty"`
	Name                string                         `json:"name,omitempty"`
	Photon              bool                           `json:"photon,omitempty"`
	Serverless          bool                           `json:"serverless,omitempty"`
	Storage             string                         `json:"storage,omitempty"`
	Target              string                         `json:"target,omitempty"`
	Url                 string                         `json:"url,omitempty"`
	Cluster             []ResourcePipelineCluster      `json:"cluster,omitempty"`
	Filters             *ResourcePipelineFilters       `json:"filters,omitempty"`
	Library             []ResourcePipelineLibrary      `json:"library,omitempty"`
	Notification        []ResourcePipelineNotification `json:"notification,omitempty"`
}
