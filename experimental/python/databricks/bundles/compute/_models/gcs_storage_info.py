from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GcsStorageInfo:
    """
    A storage location in Google Cloud Platform's GCS
    """

    destination: VariableOr[str]
    """
    GCS destination/URI, e.g. `gs://my-bucket/some-prefix`
    """

    @classmethod
    def from_dict(cls, value: "GcsStorageInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GcsStorageInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class GcsStorageInfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    GCS destination/URI, e.g. `gs://my-bucket/some-prefix`
    """


GcsStorageInfoParam = GcsStorageInfoDict | GcsStorageInfo
