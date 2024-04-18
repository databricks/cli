// Generated from Databricks Terraform provider schema. DO NOT EDIT.

package schema

type ResourceInstancePoolAwsAttributes struct {
	Availability        string `json:"availability,omitempty"`
	SpotBidPricePercent int    `json:"spot_bid_price_percent,omitempty"`
	ZoneId              string `json:"zone_id,omitempty"`
}

type ResourceInstancePoolAzureAttributes struct {
	Availability    string `json:"availability,omitempty"`
	SpotBidMaxPrice int    `json:"spot_bid_max_price,omitempty"`
}

type ResourceInstancePoolDiskSpecDiskType struct {
	AzureDiskVolumeType string `json:"azure_disk_volume_type,omitempty"`
	EbsVolumeType       string `json:"ebs_volume_type,omitempty"`
}

type ResourceInstancePoolDiskSpec struct {
	DiskCount int                                   `json:"disk_count,omitempty"`
	DiskSize  int                                   `json:"disk_size,omitempty"`
	DiskType  *ResourceInstancePoolDiskSpecDiskType `json:"disk_type,omitempty"`
}

type ResourceInstancePoolGcpAttributes struct {
	GcpAvailability string `json:"gcp_availability,omitempty"`
	LocalSsdCount   int    `json:"local_ssd_count,omitempty"`
	ZoneId          string `json:"zone_id,omitempty"`
}

type ResourceInstancePoolInstancePoolFleetAttributesFleetOnDemandOption struct {
	AllocationStrategy      string `json:"allocation_strategy"`
	InstancePoolsToUseCount int    `json:"instance_pools_to_use_count,omitempty"`
}

type ResourceInstancePoolInstancePoolFleetAttributesFleetSpotOption struct {
	AllocationStrategy      string `json:"allocation_strategy"`
	InstancePoolsToUseCount int    `json:"instance_pools_to_use_count,omitempty"`
}

type ResourceInstancePoolInstancePoolFleetAttributesLaunchTemplateOverride struct {
	AvailabilityZone string `json:"availability_zone"`
	InstanceType     string `json:"instance_type"`
}

type ResourceInstancePoolInstancePoolFleetAttributes struct {
	FleetOnDemandOption    *ResourceInstancePoolInstancePoolFleetAttributesFleetOnDemandOption     `json:"fleet_on_demand_option,omitempty"`
	FleetSpotOption        *ResourceInstancePoolInstancePoolFleetAttributesFleetSpotOption         `json:"fleet_spot_option,omitempty"`
	LaunchTemplateOverride []ResourceInstancePoolInstancePoolFleetAttributesLaunchTemplateOverride `json:"launch_template_override,omitempty"`
}

type ResourceInstancePoolPreloadedDockerImageBasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type ResourceInstancePoolPreloadedDockerImage struct {
	Url       string                                             `json:"url"`
	BasicAuth *ResourceInstancePoolPreloadedDockerImageBasicAuth `json:"basic_auth,omitempty"`
}

type ResourceInstancePool struct {
	CustomTags                         map[string]string                                `json:"custom_tags,omitempty"`
	EnableElasticDisk                  bool                                             `json:"enable_elastic_disk,omitempty"`
	Id                                 string                                           `json:"id,omitempty"`
	IdleInstanceAutoterminationMinutes int                                              `json:"idle_instance_autotermination_minutes"`
	InstancePoolId                     string                                           `json:"instance_pool_id,omitempty"`
	InstancePoolName                   string                                           `json:"instance_pool_name"`
	MaxCapacity                        int                                              `json:"max_capacity,omitempty"`
	MinIdleInstances                   int                                              `json:"min_idle_instances,omitempty"`
	NodeTypeId                         string                                           `json:"node_type_id,omitempty"`
	PreloadedSparkVersions             []string                                         `json:"preloaded_spark_versions,omitempty"`
	AwsAttributes                      *ResourceInstancePoolAwsAttributes               `json:"aws_attributes,omitempty"`
	AzureAttributes                    *ResourceInstancePoolAzureAttributes             `json:"azure_attributes,omitempty"`
	DiskSpec                           *ResourceInstancePoolDiskSpec                    `json:"disk_spec,omitempty"`
	GcpAttributes                      *ResourceInstancePoolGcpAttributes               `json:"gcp_attributes,omitempty"`
	InstancePoolFleetAttributes        *ResourceInstancePoolInstancePoolFleetAttributes `json:"instance_pool_fleet_attributes,omitempty"`
	PreloadedDockerImage               []ResourceInstancePoolPreloadedDockerImage       `json:"preloaded_docker_image,omitempty"`
}
