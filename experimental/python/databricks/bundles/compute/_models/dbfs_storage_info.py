from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DbfsStorageInfo:
    """
    A storage location in DBFS
    """

    destination: VariableOr[str]
    """
    dbfs destination, e.g. `dbfs:/my/path`
    """

    @classmethod
    def from_dict(cls, value: "DbfsStorageInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DbfsStorageInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class DbfsStorageInfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    dbfs destination, e.g. `dbfs:/my/path`
    """


DbfsStorageInfoParam = DbfsStorageInfoDict | DbfsStorageInfo
