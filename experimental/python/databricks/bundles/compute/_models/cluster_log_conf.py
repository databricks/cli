from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.dbfs_storage_info import (
    DbfsStorageInfo,
    DbfsStorageInfoParam,
)
from databricks.bundles.compute._models.s3_storage_info import (
    S3StorageInfo,
    S3StorageInfoParam,
)
from databricks.bundles.compute._models.volumes_storage_info import (
    VolumesStorageInfo,
    VolumesStorageInfoParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ClusterLogConf:
    """
    Cluster log delivery config
    """

    dbfs: VariableOrOptional[DbfsStorageInfo] = None
    """
    destination needs to be provided. e.g.
    `{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
    """

    s3: VariableOrOptional[S3StorageInfo] = None
    """
    destination and either the region or endpoint need to be provided. e.g.
    `{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }`
    Cluster iam role is used to access s3, please make sure the cluster iam role in
    `instance_profile_arn` has permission to write data to the s3 destination.
    """

    volumes: VariableOrOptional[VolumesStorageInfo] = None
    """
    destination needs to be provided, e.g.
    `{ "volumes": { "destination": "/Volumes/catalog/schema/volume/cluster_log" } }`
    """

    @classmethod
    def from_dict(cls, value: "ClusterLogConfDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ClusterLogConfDict":
        return _transform_to_json_value(self)  # type:ignore


class ClusterLogConfDict(TypedDict, total=False):
    """"""

    dbfs: VariableOrOptional[DbfsStorageInfoParam]
    """
    destination needs to be provided. e.g.
    `{ "dbfs" : { "destination" : "dbfs:/home/cluster_log" } }`
    """

    s3: VariableOrOptional[S3StorageInfoParam]
    """
    destination and either the region or endpoint need to be provided. e.g.
    `{ "s3": { "destination" : "s3://cluster_log_bucket/prefix", "region" : "us-west-2" } }`
    Cluster iam role is used to access s3, please make sure the cluster iam role in
    `instance_profile_arn` has permission to write data to the s3 destination.
    """

    volumes: VariableOrOptional[VolumesStorageInfoParam]
    """
    destination needs to be provided, e.g.
    `{ "volumes": { "destination": "/Volumes/catalog/schema/volume/cluster_log" } }`
    """


ClusterLogConfParam = ClusterLogConfDict | ClusterLogConf
