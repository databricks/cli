// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type DataSourceInstancePoolPoolInfoAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type DataSourceInstancePoolPoolInfoAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type DataSourceInstancePoolPoolInfoDiskSpecDiskType struct {
	AzureDiskVolumeType string `json:"azure_disk_volume_type,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
}

type DataSourceInstancePoolPoolInfoDiskSpec struct {
	DiskCount int                                             `json:"disk_count,omitempty"`
	DiskSize  int                                             `json:"disk_size,omitempty"`
	DiskType  *DataSourceInstancePoolPoolInfoDiskSpecDiskType `json:"disk_type,omitempty"`
}

type DataSourceInstancePoolPoolInfoGcpAttributes struct {
	GcpAvailability string `json:"gcp_availability,omitempty"`
	LocalSsdCount   int    `json:"local_ssd_count,omitempty"`
	ZoneId          string `json:"zone_id,omitempty"`
}

type DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesFleetOnDemandOption struct {
	AllocationStrategy      string `json:"allocation_strategy"`
	InstancePoolsToUseCount int    `json:"instance_pools_to_use_count,omitempty"`
}

type DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesFleetSpotOption struct {
	AllocationStrategy      string `json:"allocation_strategy"`
	InstancePoolsToUseCount int    `json:"instance_pools_to_use_count,omitempty"`
}

type DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesLaunchTemplateOverride struct {
	AvailabilityZone string `json:"availability_zone"`
	InstanceType     string `json:"instance_type"`
}

type DataSourceInstancePoolPoolInfoInstancePoolFleetAttributes struct {
	FleetOnDemandOption    *DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesFleetOnDemandOption     `json:"fleet_on_demand_option,omitempty"`
	FleetSpotOption        *DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesFleetSpotOption         `json:"fleet_spot_option,omitempty"`
	LaunchTemplateOverride []DataSourceInstancePoolPoolInfoInstancePoolFleetAttributesLaunchTemplateOverride `json:"launch_template_override,omitempty"`
}

type DataSourceInstancePoolPoolInfoPreloadedDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type DataSourceInstancePoolPoolInfoPreloadedDockerImage struct {
	Url       string                                                       `json:"url"`
	BasicAuth *DataSourceInstancePoolPoolInfoPreloadedDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type DataSourceInstancePoolPoolInfoStats struct {
	IdleCount        int `json:"idle_count,omitempty"`
	PendingIdleCount int `json:"pending_idle_count,omitempty"`
	PendingUsedCount int `json:"pending_used_count,omitempty"`
	UsedCount        int `json:"used_count,omitempty"`
}

type DataSourceInstancePoolPoolInfo struct {
	CustomTags                         map[string]string                                           `json:"custom_tags,omitempty"`
	DefaultTags                        map[string]string                                           `json:"default_tags,omitempty"`
	EnableElasticDisk                  bool                                                        `json:"enable_elastic_disk,omitempty"`
	IdleInstanceAutoterminationMinutes int                                                         `json:"idle_instance_autotermination_minutes"`
	InstancePoolId                     string                                                      `json:"instance_pool_id,omitempty"`
	InstancePoolName                   string                                                      `json:"instance_pool_name"`
	MaxCapacity                        int                                                         `json:"max_capacity,omitempty"`
	MinIdleInstances                   int                                                         `json:"min_idle_instances,omitempty"`
	NodeTypeId                         string                                                      `json:"node_type_id,omitempty"`
	PreloadedSparkVersions             []string                                                    `json:"preloaded_spark_versions,omitempty"`
	State                              string                                                      `json:"state,omitempty"`
	AwsAttributes                      *DataSourceInstancePoolPoolInfoAwsAttributes                `json:"aws_attributes,omitempty"`
	AzureAttributes                    *DataSourceInstancePoolPoolInfoAzureAttributes              `json:"azure_attributes,omitempty"`
	DiskSpec                           *DataSourceInstancePoolPoolInfoDiskSpec                     `json:"disk_spec,omitempty"`
	GcpAttributes                      *DataSourceInstancePoolPoolInfoGcpAttributes                `json:"gcp_attributes,omitempty"`
	InstancePoolFleetAttributes        []DataSourceInstancePoolPoolInfoInstancePoolFleetAttributes `json:"instance_pool_fleet_attributes,omitempty"`
	PreloadedDockerImage               []DataSourceInstancePoolPoolInfoPreloadedDockerImage        `json:"preloaded_docker_image,omitempty"`
	Stats                              *DataSourceInstancePoolPoolInfoStats                        `json:"stats,omitempty"`
}

type DataSourceInstancePool struct {
	Id       string                          `json:"id,omitempty"`
	Name     string                          `json:"name"`
	PoolInfo *DataSourceInstancePoolPoolInfo `json:"pool_info,omitempty"`
}
