from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.aws_availability import (
    AwsAvailability,
    AwsAvailabilityParam,
)
from databricks.bundles.compute._models.ebs_volume_type import (
    EbsVolumeType,
    EbsVolumeTypeParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AwsAttributes:
    """
    Attributes set during cluster creation which are related to Amazon Web Services.
    """

    availability: VariableOrOptional[AwsAvailability] = None

    ebs_volume_count: VariableOrOptional[int] = None
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int] = None
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int] = None
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeType] = None

    first_on_demand: VariableOrOptional[int] = None
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str] = None
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int] = None
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str] = None
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, a default zone will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """

    @classmethod
    def from_dict(cls, value: "AwsAttributesDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AwsAttributesDict":
        return _transform_to_json_value(self)  # type:ignore


class AwsAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[AwsAvailabilityParam]

    ebs_volume_count: VariableOrOptional[int]
    """
    The number of volumes launched for each instance. Users can choose up to 10 volumes.
    This feature is only enabled for supported node types. Legacy node types cannot specify
    custom EBS volumes.
    For node types with no instance store, at least one EBS volume needs to be specified;
    otherwise, cluster creation will fail.
    
    These EBS volumes will be mounted at `/ebs0`, `/ebs1`, and etc.
    Instance store volumes will be mounted at `/local_disk0`, `/local_disk1`, and etc.
    
    If EBS volumes are attached, Databricks will configure Spark to use only the EBS volumes for
    scratch storage because heterogenously sized scratch devices can lead to inefficient disk
    utilization. If no EBS volumes are attached, Databricks will configure Spark to use instance
    store volumes.
    
    Please note that if EBS volumes are specified, then the Spark configuration `spark.local.dir`
    will be overridden.
    """

    ebs_volume_iops: VariableOrOptional[int]
    """
    If using gp3 volumes, what IOPS to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_size: VariableOrOptional[int]
    """
    The size of each EBS volume (in GiB) launched for each instance. For general purpose
    SSD, this value must be within the range 100 - 4096. For throughput optimized HDD,
    this value must be within the range 500 - 4096.
    """

    ebs_volume_throughput: VariableOrOptional[int]
    """
    If using gp3 volumes, what throughput to use for the disk. If this is not set, the maximum performance of a gp2 volume with the same volume size will be used.
    """

    ebs_volume_type: VariableOrOptional[EbsVolumeTypeParam]

    first_on_demand: VariableOrOptional[int]
    """
    The first `first_on_demand` nodes of the cluster will be placed on on-demand instances.
    If this value is greater than 0, the cluster driver node in particular will be placed on an
    on-demand instance. If this value is greater than or equal to the current cluster size, all
    nodes will be placed on on-demand instances. If this value is less than the current cluster
    size, `first_on_demand` nodes will be placed on on-demand instances and the remainder will
    be placed on `availability` instances. Note that this value does not affect
    cluster size and cannot currently be mutated over the lifetime of a cluster.
    """

    instance_profile_arn: VariableOrOptional[str]
    """
    Nodes for this cluster will only be placed on AWS instances with this instance profile. If
    ommitted, nodes will be placed on instances without an IAM instance profile. The instance
    profile must have previously been added to the Databricks environment by an account
    administrator.
    
    This feature may only be available to certain customer plans.
    """

    spot_bid_price_percent: VariableOrOptional[int]
    """
    The bid price for AWS spot instances, as a percentage of the corresponding instance type's
    on-demand price.
    For example, if this field is set to 50, and the cluster needs a new `r3.xlarge` spot
    instance, then the bid price is half of the price of
    on-demand `r3.xlarge` instances. Similarly, if this field is set to 200, the bid price is twice
    the price of on-demand `r3.xlarge` instances. If not specified, the default value is 100.
    When spot instances are requested for this cluster, only spot instances whose bid price
    percentage matches this field will be considered.
    Note that, for safety, we enforce this field to be no more than 10000.
    """

    zone_id: VariableOrOptional[str]
    """
    Identifier for the availability zone/datacenter in which the cluster resides.
    This string will be of a form like "us-west-2a". The provided availability
    zone must be in the same region as the Databricks deployment. For example, "us-west-2a"
    is not a valid zone id if the Databricks deployment resides in the "us-east-1" region.
    This is an optional field at cluster creation, and if not specified, a default zone will be used.
    If the zone specified is "auto", will try to place cluster in a zone with high availability,
    and will retry placement in a different AZ if there is not enough capacity.
    
    The list of available zones as well as the default value can be found by using the
    `List Zones` method.
    """


AwsAttributesParam = AwsAttributesDict | AwsAttributes
