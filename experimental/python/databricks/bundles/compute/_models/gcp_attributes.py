from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.gcp_availability import (
    GcpAvailability,
    GcpAvailabilityParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GcpAttributes:
    """
    Attributes set during cluster creation which are related to GCP.
    """

    availability: VariableOrOptional[GcpAvailability] = None

    boot_disk_size: VariableOrOptional[int] = None
    """
    Boot disk size in GB
    """

    google_service_account: VariableOrOptional[str] = None
    """
    If provided, the cluster will impersonate the google service account when accessing
    gcloud services (like GCS). The google service account
    must have previously been added to the Databricks environment by an account
    administrator.
    """

    local_ssd_count: VariableOrOptional[int] = None
    """
    If provided, each node (workers and driver) in the cluster will have this number of local SSDs attached.
    Each local SSD is 375GB in size.
    Refer to [GCP documentation](https://cloud.google.com/compute/docs/disks/local-ssd#choose_number_local_ssds)
    for the supported number of local SSDs for each instance type.
    """

    zone_id: VariableOrOptional[str] = None
    """
    Identifier for the availability zone in which the cluster resides.
    This can be one of the following:
    - "HA" => High availability, spread nodes across availability zones for a Databricks deployment region [default].
    - "AUTO" => Databricks picks an availability zone to schedule the cluster on.
    - A GCP availability zone => Pick One of the available zones for (machine type + region) from
    https://cloud.google.com/compute/docs/regions-zones.
    """

    @classmethod
    def from_dict(cls, value: "GcpAttributesDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GcpAttributesDict":
        return _transform_to_json_value(self)  # type:ignore


class GcpAttributesDict(TypedDict, total=False):
    """"""

    availability: VariableOrOptional[GcpAvailabilityParam]

    boot_disk_size: VariableOrOptional[int]
    """
    Boot disk size in GB
    """

    google_service_account: VariableOrOptional[str]
    """
    If provided, the cluster will impersonate the google service account when accessing
    gcloud services (like GCS). The google service account
    must have previously been added to the Databricks environment by an account
    administrator.
    """

    local_ssd_count: VariableOrOptional[int]
    """
    If provided, each node (workers and driver) in the cluster will have this number of local SSDs attached.
    Each local SSD is 375GB in size.
    Refer to [GCP documentation](https://cloud.google.com/compute/docs/disks/local-ssd#choose_number_local_ssds)
    for the supported number of local SSDs for each instance type.
    """

    zone_id: VariableOrOptional[str]
    """
    Identifier for the availability zone in which the cluster resides.
    This can be one of the following:
    - "HA" => High availability, spread nodes across availability zones for a Databricks deployment region [default].
    - "AUTO" => Databricks picks an availability zone to schedule the cluster on.
    - A GCP availability zone => Pick One of the available zones for (machine type + region) from
    https://cloud.google.com/compute/docs/regions-zones.
    """


GcpAttributesParam = GcpAttributesDict | GcpAttributes
