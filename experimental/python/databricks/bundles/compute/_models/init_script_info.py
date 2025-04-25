from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.adlsgen2_info import (
    Adlsgen2Info,
    Adlsgen2InfoParam,
)
from databricks.bundles.compute._models.gcs_storage_info import (
    GcsStorageInfo,
    GcsStorageInfoParam,
)
from databricks.bundles.compute._models.local_file_info import (
    LocalFileInfo,
    LocalFileInfoParam,
)
from databricks.bundles.compute._models.s3_storage_info import (
    S3StorageInfo,
    S3StorageInfoParam,
)
from databricks.bundles.compute._models.volumes_storage_info import (
    VolumesStorageInfo,
    VolumesStorageInfoParam,
)
from databricks.bundles.compute._models.workspace_storage_info import (
    WorkspaceStorageInfo,
    WorkspaceStorageInfoParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class InitScriptInfo:
    """
    Config for an individual init script
    Next ID: 11
    """

    abfss: VariableOrOptional[Adlsgen2Info] = None
    """
    Contains the Azure Data Lake Storage destination path
    """

    file: VariableOrOptional[LocalFileInfo] = None
    """
    destination needs to be provided, e.g.
    `{ "file": { "destination": "file:/my/local/file.sh" } }`
    """

    gcs: VariableOrOptional[GcsStorageInfo] = None
    """
    destination needs to be provided, e.g.
    `{ "gcs": { "destination": "gs://my-bucket/file.sh" } }`
    """

    s3: VariableOrOptional[S3StorageInfo] = None
    """
    destination and either the region or endpoint need to be provided. e.g.
    `{ \"s3\": { \"destination\": \"s3://cluster_log_bucket/prefix\", \"region\": \"us-west-2\" } }`
    Cluster iam role is used to access s3, please make sure the cluster iam role in
    `instance_profile_arn` has permission to write data to the s3 destination.
    """

    volumes: VariableOrOptional[VolumesStorageInfo] = None
    """
    destination needs to be provided. e.g.
    `{ \"volumes\" : { \"destination\" : \"/Volumes/my-init.sh\" } }`
    """

    workspace: VariableOrOptional[WorkspaceStorageInfo] = None
    """
    destination needs to be provided, e.g.
    `{ "workspace": { "destination": "/cluster-init-scripts/setup-datadog.sh" } }`
    """

    @classmethod
    def from_dict(cls, value: "InitScriptInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "InitScriptInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class InitScriptInfoDict(TypedDict, total=False):
    """"""

    abfss: VariableOrOptional[Adlsgen2InfoParam]
    """
    Contains the Azure Data Lake Storage destination path
    """

    file: VariableOrOptional[LocalFileInfoParam]
    """
    destination needs to be provided, e.g.
    `{ "file": { "destination": "file:/my/local/file.sh" } }`
    """

    gcs: VariableOrOptional[GcsStorageInfoParam]
    """
    destination needs to be provided, e.g.
    `{ "gcs": { "destination": "gs://my-bucket/file.sh" } }`
    """

    s3: VariableOrOptional[S3StorageInfoParam]
    """
    destination and either the region or endpoint need to be provided. e.g.
    `{ \"s3\": { \"destination\": \"s3://cluster_log_bucket/prefix\", \"region\": \"us-west-2\" } }`
    Cluster iam role is used to access s3, please make sure the cluster iam role in
    `instance_profile_arn` has permission to write data to the s3 destination.
    """

    volumes: VariableOrOptional[VolumesStorageInfoParam]
    """
    destination needs to be provided. e.g.
    `{ \"volumes\" : { \"destination\" : \"/Volumes/my-init.sh\" } }`
    """

    workspace: VariableOrOptional[WorkspaceStorageInfoParam]
    """
    destination needs to be provided, e.g.
    `{ "workspace": { "destination": "/cluster-init-scripts/setup-datadog.sh" } }`
    """


InitScriptInfoParam = InitScriptInfoDict | InitScriptInfo
