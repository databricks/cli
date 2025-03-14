from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class VolumesStorageInfo:
    """"""

    destination: VariableOr[str]
    """
    Unity Catalog Volumes file destination, e.g. `/Volumes/my-init.sh`
    """

    @classmethod
    def from_dict(cls, value: "VolumesStorageInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "VolumesStorageInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class VolumesStorageInfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    Unity Catalog Volumes file destination, e.g. `/Volumes/my-init.sh`
    """


VolumesStorageInfoParam = VolumesStorageInfoDict | VolumesStorageInfo
