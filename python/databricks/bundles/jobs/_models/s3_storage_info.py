from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class S3StorageInfo:
    """
    A storage location in Amazon S3
    """

    destination: VariableOr[str]
    """
    S3 destination, e.g. `s3://my-bucket/some-prefix` Note that logs will be delivered using
    cluster iam role, please make sure you set cluster iam role and the role has write access to the
    destination. Please also note that you cannot use AWS keys to deliver logs.
    """

    canned_acl: VariableOrOptional[str] = None
    """
    (Optional) Set canned access control list for the logs, e.g. `bucket-owner-full-control`.
    If `canned_cal` is set, please make sure the cluster iam role has `s3:PutObjectAcl` permission on
    the destination bucket and prefix. The full list of possible canned acl can be found at
    http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl.
    Please also note that by default only the object owner gets full controls. If you are using cross account
    role for writing data, you may want to set `bucket-owner-full-control` to make bucket owner able to
    read the logs.
    """

    enable_encryption: VariableOrOptional[bool] = None
    """
    (Optional) Flag to enable server side encryption, `false` by default.
    """

    encryption_type: VariableOrOptional[str] = None
    """
    (Optional) The encryption type, it could be `sse-s3` or `sse-kms`. It will be used only when
    encryption is enabled and the default type is `sse-s3`.
    """

    endpoint: VariableOrOptional[str] = None
    """
    S3 endpoint, e.g. `https://s3-us-west-2.amazonaws.com`. Either region or endpoint needs to be set.
    If both are set, endpoint will be used.
    """

    kms_key: VariableOrOptional[str] = None
    """
    (Optional) Kms key which will be used if encryption is enabled and encryption type is set to `sse-kms`.
    """

    region: VariableOrOptional[str] = None
    """
    S3 region, e.g. `us-west-2`. Either region or endpoint needs to be set. If both are set,
    endpoint will be used.
    """

    @classmethod
    def from_dict(cls, value: "S3StorageInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "S3StorageInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class S3StorageInfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    S3 destination, e.g. `s3://my-bucket/some-prefix` Note that logs will be delivered using
    cluster iam role, please make sure you set cluster iam role and the role has write access to the
    destination. Please also note that you cannot use AWS keys to deliver logs.
    """

    canned_acl: VariableOrOptional[str]
    """
    (Optional) Set canned access control list for the logs, e.g. `bucket-owner-full-control`.
    If `canned_cal` is set, please make sure the cluster iam role has `s3:PutObjectAcl` permission on
    the destination bucket and prefix. The full list of possible canned acl can be found at
    http://docs.aws.amazon.com/AmazonS3/latest/dev/acl-overview.html#canned-acl.
    Please also note that by default only the object owner gets full controls. If you are using cross account
    role for writing data, you may want to set `bucket-owner-full-control` to make bucket owner able to
    read the logs.
    """

    enable_encryption: VariableOrOptional[bool]
    """
    (Optional) Flag to enable server side encryption, `false` by default.
    """

    encryption_type: VariableOrOptional[str]
    """
    (Optional) The encryption type, it could be `sse-s3` or `sse-kms`. It will be used only when
    encryption is enabled and the default type is `sse-s3`.
    """

    endpoint: VariableOrOptional[str]
    """
    S3 endpoint, e.g. `https://s3-us-west-2.amazonaws.com`. Either region or endpoint needs to be set.
    If both are set, endpoint will be used.
    """

    kms_key: VariableOrOptional[str]
    """
    (Optional) Kms key which will be used if encryption is enabled and encryption type is set to `sse-kms`.
    """

    region: VariableOrOptional[str]
    """
    S3 region, e.g. `us-west-2`. Either region or endpoint needs to be set. If both are set,
    endpoint will be used.
    """


S3StorageInfoParam = S3StorageInfoDict | S3StorageInfo
