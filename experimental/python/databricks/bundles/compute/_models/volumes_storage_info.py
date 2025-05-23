from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class VolumesStorageInfo:
    """
    A storage location back by UC Volumes.
    """

    destination: VariableOr[str]
    """
    UC Volumes destination, e.g. `/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
    or `dbfs:/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
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
    UC Volumes destination, e.g. `/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
    or `dbfs:/Volumes/catalog/schema/vol1/init-scripts/setup-datadog.sh`
    """


VolumesStorageInfoParam = VolumesStorageInfoDict | VolumesStorageInfo
