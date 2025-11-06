from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class FileLibrary:
    """"""

    path: VariableOrOptional[str] = None
    """
    The absolute path of the source code.
    """

    @classmethod
    def from_dict(cls, value: "FileLibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FileLibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class FileLibraryDict(TypedDict, total=False):
    """"""

    path: VariableOrOptional[str]
    """
    The absolute path of the source code.
    """


FileLibraryParam = FileLibraryDict | FileLibrary
